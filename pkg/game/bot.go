package game

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"

	"bot-template-go/pkg/engine"
	"bot-template-go/pkg/network"
)

// Bot implementa o ciclo de vida e a automação do Farm Ricochet
type Bot struct {
	config         AccountConfig
	mu             sync.RWMutex
	status         engine.AccountStatus
	client         *FarmClient
	circuitBreaker *engine.CircuitBreaker
	socket         *network.SocketClient

	holderId       string
	leaseExpiresAt time.Time
	activeRunId    string
	lastBootstrap  *BootstrapResponse

	running   bool
	cancel    context.CancelFunc
	startTime time.Time

	// Anti-cheat & Circuit Breaker trip notification
	onTripNotice func(reason string)
}

// NewBot inicializa uma nova instância de bot para a conta
func NewBot(cfg AccountConfig) *Bot {
	st := engine.NewStatus(cfg.ID, cfg.Name)
	st.Status = "Configurado"

	cbCfg := engine.DefaultCircuitBreakerConfig()
	cbCfg.BaseCooldown = 15 * time.Minute
	cbCfg.MaxCooldown = 60 * time.Minute

	b := &Bot{
		config:   cfg,
		status:   st,
		holderId: newUUID(),
	}

	cbCfg.OnTrip = func(reason string) {
		log.Printf("[%s] 🚨 CIRCUIT BREAKER DISPARADO! Motivo: %s", b.config.ID, reason)
		b.updateStatus(func(s *engine.AccountStatus) {
			s.Status = "PAUSADO (Anti-Cheat)"
			s.Logs = append(s.Logs, fmt.Sprintf("⚠️ Proteção anti-ban ativada: %s. Cooldown de segurança em andamento.", reason))
		})
	}

	b.circuitBreaker = engine.NewCircuitBreaker(cfg.ID, cbCfg)
	b.client = NewFarmClient(cfg.Credentials.Cookie, cfg.Credentials.SessionToken)

	return b
}

func (b *Bot) ID() string {
	return b.config.ID
}

func (b *Bot) Name() string {
	return b.config.Name
}

// Init valida a sessão, adquire o lease e carrega o estado inicial do jogo
func (b *Bot) Init(ctx context.Context) error {
	b.updateStatus(func(s *engine.AccountStatus) {
		s.Status = "Validando sessão..."
	})

	if b.config.Credentials.Cookie == "" && b.config.Credentials.SessionToken == "" {
		err := fmt.Errorf("credenciais vazias. Insira o Cookie de sessão em accounts.json")
		b.addLog("Erro: Credenciais não configuradas no accounts.json")
		return err
	}

	// 1. Adquire ou renova lease da sessão de jogo no servidor
	lease, err := b.client.AcquireSessionLease(b.holderId, true)
	if err != nil {
		if strings.Contains(err.Error(), "401") {
			b.addLog("Sessão expirada ou cookie inválido (HTTP 401). Atualize o cookie.")
			return fmt.Errorf("sessão expirada: %w", err)
		}
		if strings.Contains(err.Error(), "account_restricted") {
			b.circuitBreaker.Trip("Conta com restrição detectada")
			return fmt.Errorf("conta restrita: %w", err)
		}
		log.Printf("[%s] Aviso ao reservar sessão: %v (tentando continuar)", b.ID(), err)
	} else if lease != nil {
		b.mu.Lock()
		b.leaseExpiresAt, _ = time.Parse(time.RFC3339, lease.ExpiresAt)
		b.mu.Unlock()
	}

	// 2. Carrega Bootstrap do jogo
	boot, err := b.client.GetBootstrap()
	if err != nil {
		b.addLog(fmt.Sprintf("Falha ao carregar bootstrap: %v", err))
		return err
	}

	b.mu.Lock()
	b.lastBootstrap = boot
	b.mu.Unlock()

	// 3. Atualiza dados do personagem no dashboard
	b.updateStatus(func(s *engine.AccountStatus) {
		s.Status = "Conectado"
		if boot.Account != nil {
			s.Name = boot.Account.DisplayName
			s.Gold = boot.Account.CommonCurrency
			s.Extra["gems"] = boot.Account.PremiumCurrency
			s.Extra["wheat"] = boot.Account.Wheat
			s.Extra["wood"] = boot.Account.Wood
			s.Extra["stone"] = boot.Account.Stone
			s.Extra["essence"] = boot.Account.HeroicEssence
			s.Extra["forgeFragments"] = boot.Account.ForgeFragments
			if boot.Account.Vip != nil && boot.Account.Vip.Active {
				s.Extra["vip"] = true
			}
		}

		if boot.ActiveLevel != nil {
			s.CurrentArea = fmt.Sprintf("Fase %d (%s)", boot.ActiveLevel.Number, boot.ActiveLevel.Name)
			s.Level = boot.ActiveLevel.Number
		}

		if boot.Composition != nil && len(boot.Composition.Slots) > 0 {
			primary := boot.Composition.Slots[0]
			s.Vocation = primary.Code
			s.HP = int(primary.HP)
			s.MaxHP = int(primary.HP)
			s.Mana = 100
			s.MaxMana = 100
		}
	})

	b.addLog(fmt.Sprintf("Fazenda conectada! Fase ativa: %s | Ouro: %d", b.status.CurrentArea, b.status.Gold))
	return nil
}

// Run executa o loop contínuo de automação autônoma do bot
func (b *Bot) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	b.mu.Lock()
	b.cancel = cancel
	b.running = true
	b.startTime = time.Now()
	b.mu.Unlock()

	defer func() {
		b.mu.Lock()
		b.running = false
		b.mu.Unlock()
		// Ao encerrar, libera o lease da sessão no servidor
		_ = b.client.ReleaseSessionLease(b.holderId)
	}()

	// Inicia conexão WebSocket para manter conexão ativa de stream
	b.initWebSocket()

	// Goroutine 1: Heartbeat da sessão de jogo a cada 8 segundos
	go b.sessionHeartbeatLoop(ctx)

	// Goroutine 2: Loop de tarefas de manutenção (baús, missões, passe, talentos, heróis)
	go b.maintenanceLoop(ctx)

	// Loop Principal: Execução da campanha / farm
	b.updateStatus(func(s *engine.AccountStatus) {
		s.Status = "hunting"
	})
	b.addLog("Automação total iniciada. Loop de farm e campanha em execução.")

	for {
		select {
		case <-ctx.Done():
			b.addLog("Bot parado pelo supervisor.")
			return ctx.Err()
		default:
			// Verifica disjuntor anti-ban
			if !b.circuitBreaker.AllowAction() {
				b.updateStatus(func(s *engine.AccountStatus) {
					s.Status = "PAUSADO (Anti-Cheat / Cooldown)"
				})
				time.Sleep(15 * time.Second)
				continue
			}

			if !b.config.Settings.AutoHunt {
				b.updateStatus(func(s *engine.AccountStatus) {
					s.Status = "idle (Farm pausado)"
				})
				time.Sleep(3 * time.Second)
				continue
			}

			// Executa um ciclo completo de fase / run
			err := b.performRunCycle(ctx)
			if err != nil {
				if isRejection, reason := engine.IsAntiCheatRejection(0, err.Error()); isRejection {
					b.circuitBreaker.Trip(reason)
				} else {
					log.Printf("[%s] Erro no ciclo de farm: %v", b.ID(), err)
					b.addLog(fmt.Sprintf("Aviso: %v. Aguardando 5s...", err))
				}
				time.Sleep(5 * time.Second)
				continue
			}

			// Intervalo humano seguro entre fases
			time.Sleep(time.Duration(1200+rand.Intn(1500)) * time.Millisecond)
		}
	}
}

// sessionHeartbeatLoop envia heartbeat contínuo para manter o lease da sessão ativo
func (b *Bot) sessionHeartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(8 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Se o disjuntor estiver aberto (em cooldown de segurança), aguarda
			if b.circuitBreaker.State() == engine.StateOpen {
				continue
			}

			lease, err := b.client.HeartbeatSessionLease(b.holderId)
			if err != nil {
				if strings.Contains(err.Error(), "409") {
					// Conflito de sessão detectado: tenta retomar controle com takeover
					b.addLog("⚠️ Conflito de sessão detectado (409). Tentando recuperar controle do lease...")
					newLease, acqErr := b.client.AcquireSessionLease(b.holderId, true)
					if acqErr == nil && newLease != nil {
						b.mu.Lock()
						b.leaseExpiresAt, _ = time.Parse(time.RFC3339, newLease.ExpiresAt)
						b.mu.Unlock()
						b.circuitBreaker.RecordSuccess()
						b.addLog("✅ Controle da sessão retomado com sucesso!")
						continue
					}
					// Se mesmo com força falhou e estava fechado, ativa proteção anti-ban
					if b.circuitBreaker.State() == engine.StateClosed {
						b.circuitBreaker.Trip("Conflito de sessão persistente (outra aba ou dispositivo conectado)")
					}
				}
			} else if lease != nil {
				b.mu.Lock()
				b.leaseExpiresAt, _ = time.Parse(time.RFC3339, lease.ExpiresAt)
				b.mu.Unlock()
				b.circuitBreaker.RecordSuccess()
			}
		}
	}
}

// maintenanceLoop executa tarefas periódicas de baús, missões, passe e talentos
func (b *Bot) maintenanceLoop(ctx context.Context) {
	// Aguarda 5 segundos após inicialização antes do primeiro ciclo
	select {
	case <-ctx.Done():
		return
	case <-time.After(5 * time.Second):
	}

	ticker := time.NewTicker(45 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !b.circuitBreaker.AllowAction() {
				continue
			}
			b.processMaintenance()
		}
	}
}

func (b *Bot) processMaintenance() {
	// 1. Abre Baús pendentes ou gratuitos
	if b.config.Settings.AutoChests {
		b.processChests()
	}

	// 2. Resgata Missões diárias, semanais e conquistas
	if b.config.Settings.AutoMissions {
		b.processMissions()
	}

	// 3. Resgata Passe de Batalha
	if b.config.Settings.AutoBattlePass {
		b.processBattlePass()
	}

	// 4. Compra Talentos da Fazenda
	if b.config.Settings.AutoTalents {
		b.processTalents()
	}

	// 5. Hero Ascension (subir estrelas de duplicatas)
	if b.config.Settings.AutoAscension {
		b.processHeroAscension()
	}

	// 6. Desmonta itens comuns repetidos
	if b.config.Settings.AutoDismantle {
		b.processDismantle()
	}

	// 7. Funde cristais de ascensão
	if b.config.Settings.AutoFuseCrystals {
		b.processCrystalFusion()
	}

	// 8. Otimização e preenchimento de vagas da equipe
	b.processTeamSlots()

	// 9. Auto-Equip (equipa os melhores itens por herói e slot)
	if b.config.Settings.AutoEquip {
		b.processAutoEquip()
	}

	// 10. Forja (aprimoramento seguro de equipamentos até +6)
	if b.config.Settings.AutoForge {
		b.processForgeEnhancement()
	}

	// 11. Poço Mítico (depósito e resgate de itens míticos)
	if b.config.Settings.AutoMythicWell {
		b.processMythicWell()
	}

	// 12. Clã / Guilda (missões de cooperação e resgate de baús)
	if b.config.Settings.AutoClan {
		b.processClan()
	}

	// 13. Prestígio (marcos e perks de tier)
	if b.config.Settings.AutoPrestige {
		b.processPrestige()
	}

	// 14. Desafio Infinito (Torre de Ondas)
	if b.config.Settings.AutoInfinite {
		b.processInfiniteChallenge()
	}

	// 15. Arena PvP (resgate de missões diárias/semanais e defesa)
	if b.config.Settings.AutoPvP {
		b.processPvP()
	}
}

// performRunCycle gerencia o início, execução rate-safe e conclusão de uma fase
func (b *Bot) performRunCycle(ctx context.Context) error {
	// 1. Atualiza bootstrap para obter fase ativa canônica
	boot, err := b.client.GetBootstrap()
	if err != nil {
		return err
	}
	b.mu.Lock()
	b.lastBootstrap = boot
	b.mu.Unlock()

	activeLvl := boot.ActiveLevel
	if activeLvl == nil {
		return fmt.Errorf("nenhuma fase ativa encontrada")
	}

	maxAllowed := b.getSafetyMaxLevel(boot)
	if activeLvl.Number > maxAllowed {
		for _, l := range boot.Levels {
			if l.Number == maxAllowed {
				b.addLog(fmt.Sprintf("🛡️ Fase atual (%d) acima do teto de segurança (%d). Recuando para farm seguro em %s...", activeLvl.Number, maxAllowed, l.Code))
				_ = b.client.SetActiveLevel(l.Code)
				break
			}
		}
	}

	b.updateStatus(func(s *engine.AccountStatus) {
		s.CurrentArea = fmt.Sprintf("Fase %d: %s", activeLvl.Number, activeLvl.Name)
		s.Level = activeLvl.Number
		s.Status = fmt.Sprintf("Jogando Fase %d", activeLvl.Number)
	})

	// 2. Inicia a run validada no servidor
	clientRunId := newUUID()
	runReq := RunStartRequest{
		ClientRunId:        clientRunId,
		HolderId:           b.holderId,
		PersistenceVersion: 2,
		ClientRevision:     1,
		ClientRefreshToken: "",
	}

	runResp, err := b.client.StartRun(runReq)
	if err != nil {
		if strings.Contains(err.Error(), "409") || strings.Contains(err.Error(), "lease") {
			b.addLog("⚠️ Conflito/expiração de lease antes de iniciar fase. Re-adquirindo sessão...")
			newLease, acqErr := b.client.AcquireSessionLease(b.holderId, true)
			if acqErr == nil && newLease != nil {
				b.mu.Lock()
				b.leaseExpiresAt, _ = time.Parse(time.RFC3339, newLease.ExpiresAt)
				b.mu.Unlock()
				b.circuitBreaker.RecordSuccess()
				runResp, err = b.client.StartRun(runReq)
			}
		}
		if err != nil {
			return fmt.Errorf("falha ao iniciar run: %w", err)
		}
	}

	runId := runResp.ID
	b.mu.Lock()
	b.activeRunId = runId
	b.mu.Unlock()

	targetGoal := runResp.ObjectiveAmount
	if targetGoal <= 0 {
		targetGoal = 300
	}

	rate := runResp.OnlineRatePerSecond
	if rate <= 0 {
		rate = 4.0
	}

	// Calcula a duração total exigida pela taxa canônica do servidor
	speedMult := b.config.Settings.SpeedMultiplier
	if speedMult <= 0.2 || speedMult > 10.0 {
		speedMult = 1.0
	}

	durationSec := float64(targetGoal) / (rate * speedMult)
	// Garante um piso de segurança mínimo de 8 segundos por fase
	if durationSec < 8.0 {
		durationSec = 8.0
	}

	b.addLog(fmt.Sprintf("▶ Fase %d iniciada (Meta: %d %s | Base: %.2f/s -> Real: %.2f/s [%.2fx] | Est: %.0fs)",
		activeLvl.Number, targetGoal, runResp.ObjectiveResource, rate, rate*speedMult, speedMult, durationSec))

	// Loop de simulação da fase com heartbeats a cada 6 segundos
	runStart := time.Now()
	heartbeatInterval := 6 * time.Second
	nextHeartbeat := time.Now().Add(heartbeatInterval)
	goblinDefeated := false
	hadSyncWait := false

	lastProgress := 0
	authProgress := 0
	structureSeq := 0
	syncAttempts := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			elapsed := time.Since(runStart)
			progressRatio := math.Min(1.0, elapsed.Seconds()/durationSec)
			currentProgress := int(float64(targetGoal) * progressRatio)

			// Derrota Loot Goblin após 40% da run se presente
			if runResp.LootGoblin && !goblinDefeated && progressRatio >= 0.4 {
				_ = b.client.DefeatLootGoblin(runId)
				goblinDefeated = true
				b.addLog("👾 Loot Goblin derrotado na partida!")
			}

			// Envia heartbeat se o intervalo chegou
			if time.Now().After(nextHeartbeat) && progressRatio < 1.0 {
				delta := currentProgress - lastProgress
				var batch []StructureEvent
				if delta > 0 {
					batch, structureSeq = b.generateStructureBatch(boot, runResp, structureSeq, delta, elapsed)
					for _, ev := range batch {
						lastProgress += ev.Amount
					}
				}
				hbEnv := b.buildEnvelope(b.holderId, "", currentProgress, elapsed, targetGoal, runResp, batch)
				hbResp, hbErr := b.client.HeartbeatRun(runId, hbEnv)
				if hbErr != nil {
					log.Printf("[%s] Heartbeat run aviso: %v", b.ID(), hbErr)
				} else if hbResp != nil {
					if hbResp.AuthoritativeProgress > 0 {
						authProgress = hbResp.AuthoritativeProgress
					}
					log.Printf("[%s] [HB] prog=%d/%d auth=%d/%d", b.ID(), currentProgress, targetGoal, hbResp.AuthoritativeProgress, hbResp.ObjectiveAmount)
				}
				nextHeartbeat = time.Now().Add(heartbeatInterval)
			}

			// Se a fase atingiu a duração necessária, conclui
			if progressRatio >= 1.0 {
				syncRetries := 0
				for authProgress < targetGoal && syncRetries < 5 {
					syncRetries++
					delta := targetGoal - authProgress
					var batch []StructureEvent
					batch, structureSeq = b.generateStructureBatch(boot, runResp, structureSeq, delta, elapsed)
					hbEnv := b.buildEnvelope(b.holderId, "", targetGoal, elapsed, targetGoal, runResp, batch)
					hbResp, hbErr := b.client.HeartbeatRun(runId, hbEnv)
					if hbErr == nil && hbResp != nil && hbResp.AuthoritativeProgress > 0 {
						authProgress = hbResp.AuthoritativeProgress
					}
					if authProgress >= targetGoal {
						break
					}
					time.Sleep(2 * time.Second)
				}
				time.Sleep(1500 * time.Millisecond)

				completeEnv := b.buildEnvelope(b.holderId, "settle", targetGoal, elapsed, targetGoal, runResp, nil)
				completeResp, compErr := b.client.CompleteRun(runId, completeEnv)
				if compErr != nil {
					return fmt.Errorf("conclusão da run recusada: %w", compErr)
				}
				if completeResp != nil {
					b.addLog(fmt.Sprintf("🔍 Complete: accepted=%v outcome=%s nextCode=%s nextId=%s auth=%d/%d gold=%d",
						completeResp.Accepted, completeResp.Outcome, completeResp.NextLevelCode, completeResp.NextLevelId,
						completeResp.AuthoritativeProgress, completeResp.ObjectiveAmount, completeResp.GoldReward))
				}

				// Se o servidor avisar que a simulação chegou antes do relógio canônico
				if completeResp != nil && !completeResp.Accepted {
					hadSyncWait = true
					syncAttempts++
					if syncAttempts > 3 {
						b.addLog("⚠️ Lease de simulação expirada após tentativa de settle. Reiniciando run limpa...")
						return nil
					}
					if completeResp.AuthoritativeProgress > 0 {
						lastProgress = completeResp.AuthoritativeProgress
					}
					waitMs := completeResp.RetryAfterMs
					if waitMs < 4000 {
						waitMs = 5000
					}
					b.addLog(fmt.Sprintf("⏳ Sincronizando com relógio do servidor (%d/%d)... aguardando %dms",
						completeResp.AuthoritativeProgress, completeResp.ObjectiveAmount, waitMs))

					missing := targetGoal - lastProgress
					if missing > 0 {
						var syncBatch []StructureEvent
						syncBatch, structureSeq = b.generateStructureBatch(boot, runResp, structureSeq, missing, time.Since(runStart))
						for _, ev := range syncBatch {
							lastProgress += ev.Amount
						}
						syncHbEnv := b.buildEnvelope(b.holderId, "", targetGoal, time.Since(runStart), targetGoal, runResp, syncBatch)
						hbResp, hbErr := b.client.HeartbeatRun(runId, syncHbEnv)
						if hbErr != nil {
							b.addLog("⚠️ Lease do heartbeat inativa. Reiniciando run limpa...")
							return nil
						}
						if hbResp != nil && hbResp.AuthoritativeProgress > 0 {
							lastProgress = hbResp.AuthoritativeProgress
						}
					}
					time.Sleep(time.Duration(waitMs) * time.Millisecond)
					continue
				}

				// Atualiza estatísticas do bot
				b.recordRunSuccess(activeLvl.Number, completeResp)

				// Auto-Tuner adaptativo de velocidade
				if b.config.Settings.AutoTuneSpeed {
					if hadSyncWait {
						actualElapsedSec := time.Since(runStart).Seconds()
						actualMaxMult := float64(targetGoal) / (rate * actualElapsedSec)
						optSpeed := math.Round(math.Max(1.10, actualMaxMult-0.05)*100) / 100
						b.config.Settings.SpeedMultiplier = optSpeed
						b.config.Settings.AutoTuneSpeed = false
						b.addLog(fmt.Sprintf("🎯 LIMITE DESCOBERTO! Servidor exigiu %.0fs reais -> Teto cravado em %.2fx (%.2f/s)!", actualElapsedSec, optSpeed, rate*optSpeed))
					} else if b.config.Settings.SpeedMultiplier < 10.0 {
						nextSpeed := math.Round((b.config.Settings.SpeedMultiplier+0.15)*100) / 100
						b.config.Settings.SpeedMultiplier = nextSpeed
						b.addLog(fmt.Sprintf("📈 Auto-Tuner: Partida aceita sem espera! Subindo para %.2fx na próxima run...", nextSpeed))
					}
				}

				// Coleta espólios acumulados da run
				_ = b.client.ClaimRunLoot(newUUID())

				// Se desbloqueou próxima fase e AutoAdvancePhase estiver ativo, avança
				if b.config.Settings.AutoAdvancePhase {
					maxAllowed := b.getSafetyMaxLevel(boot)
					nextNum := completeResp.NextLevelNumber
					if nextNum <= 0 && completeResp.NextLevelCode != "" {
						for _, l := range boot.Levels {
							if l.Code == completeResp.NextLevelCode {
								nextNum = l.Number
								break
							}
						}
					}

					// Trava de segurança ativa: se atingir o teto de segurança, não avança mais
					if (nextNum > 0 && nextNum > maxAllowed) || activeLvl.Number >= maxAllowed {
						b.addLog(fmt.Sprintf("🛑 Trava de Segurança Ativa: Fase %d atingida (limite de segurança: %d, %d fases antes do fim). Permanecendo em farm seguro contínuo!", activeLvl.Number, maxAllowed, b.config.Settings.SafetyMarginPhases))
						return nil
					}

					if completeResp.NextLevelCode != "" {
						b.addLog(fmt.Sprintf("🚀 Próxima fase desbloqueada: %s (Nv. %d)! Avançando...", completeResp.NextLevelCode, completeResp.NextLevelNumber))
						_ = b.client.SetActiveLevel(completeResp.NextLevelCode)
					} else {
						if newBoot, bootErr := b.client.GetBootstrap(); bootErr == nil && newBoot != nil && newBoot.Progress != nil {
							unlocked := newBoot.Progress.HighestUnlockedLevel
							if unlocked > maxAllowed {
								unlocked = maxAllowed
							}
							if unlocked > activeLvl.Number {
								for _, lvl := range newBoot.Levels {
									if lvl.Number == unlocked {
										b.addLog(fmt.Sprintf("🚀 Avançando para nível desbloqueado: %s (Nv. %d)!", lvl.Code, lvl.Number))
										_ = b.client.SetActiveLevel(lvl.Code)
										break
									}
								}
							}
						}
					}
				}

				return nil
			}

			time.Sleep(1000 * time.Millisecond)
		}
	}
}

// getSafetyMaxLevel calcula a fase máxima permitida respeitando a margem de segurança contra atenção/ban
func (b *Bot) getSafetyMaxLevel(boot *BootstrapResponse) int {
	margin := b.config.Settings.SafetyMarginPhases
	if margin <= 0 {
		margin = 6
	}

	maxLevel := b.config.Settings.MaxCampaignLevel
	if maxLevel <= 0 {
		maxGameLevel := 520
		if boot != nil && len(boot.Levels) > 0 {
			for _, l := range boot.Levels {
				if l.Number > maxGameLevel {
					maxGameLevel = l.Number
				}
			}
		}
		maxLevel = maxGameLevel - margin
	}

	if maxLevel < 1 {
		maxLevel = 1
	}
	return maxLevel
}

// generateStructureBatch gera eventos de produção dinâmicos baseados no progresso do período
func (b *Bot) generateStructureBatch(boot *BootstrapResponse, run *RunStartResponse, seqStart int, deltaResources int, elapsed time.Duration) ([]StructureEvent, int) {
	var events []StructureEvent
	if boot == nil || boot.ActiveLevel == nil || deltaResources <= 0 {
		return events, seqStart
	}

	levelVersionId := boot.ActiveLevel.LevelVersionId
	mapVersionId := boot.ActiveLevel.MapVersionId

	var heroId string
	if boot.Composition != nil && len(boot.Composition.Slots) > 0 {
		heroId = boot.Composition.Slots[0].CharacterId
	}

	type sInfo struct {
		id       string
		code     string
		resource string
	}
	byResource := make(map[string][]sInfo)
	for _, ent := range boot.ActiveLevel.Entities {
		if ent.Kind == "structure" {
			res := "wheat"
			if strings.Contains(ent.Type, "wood") || strings.Contains(ent.Type, "sawmill") || strings.Contains(ent.Type, "lumber") {
				res = "wood"
			} else if strings.Contains(ent.Type, "stone") || strings.Contains(ent.Type, "mine") || strings.Contains(ent.Type, "quarry") {
				res = "stone"
			}
			byResource[res] = append(byResource[res], sInfo{id: ent.ID, code: ent.Type, resource: res})
		}
	}

	if len(byResource) == 0 {
		return events, seqStart
	}

	seq := seqStart
	perRes := deltaResources / len(byResource)
	if perRes < 12 {
		perRes = 12
	}

	sIdx := 0
	for res, sList := range byResource {
		if len(sList) == 0 {
			continue
		}
		rem := perRes
		for rem > 0 {
			chunk := 12
			if rem < chunk {
				chunk = rem
			}
			rem -= chunk

			st := sList[sIdx%len(sList)]
			sIdx++
			seq++

			ev := StructureEvent{
				EventId:          fmt.Sprintf("%s%012x", run.ID[:24], seq),
				LevelVersionId:   levelVersionId,
				MapVersionId:     mapVersionId,
				Sequence:         seq,
				StructureId:      st.id,
				StructureCode:    st.code,
				StructureVersion: 1,
				HeroId:           heroId,
				EventType:        "production",
				OccurredAtMs:     elapsed.Milliseconds(),
				Resource:         res,
				Amount:           chunk,
				Quantity:         chunk,
			}
			events = append(events, ev)
		}
	}

	return events, seq
}

// buildEnvelope cria o pacote de telemetria canônico do jogo
func (b *Bot) buildEnvelope(holderId, outcome string, progress int, elapsed time.Duration, goal int, run *RunStartResponse, events []StructureEvent) RunEnvelope {
	ms := elapsed.Milliseconds()

	// Mapeia recursos e monstros reais da fase ativa
	collection := make(map[string]int)
	enemyDefeats := make(map[string]int)
	enemiesDefeated := 0

	b.mu.RLock()
	boot := b.lastBootstrap
	b.mu.RUnlock()

	if boot != nil && boot.ActiveLevel != nil {
		// 1. Mapeia e contabiliza os monstros reais presentes no mapa da fase
		for _, ent := range boot.ActiveLevel.Entities {
			if ent.Kind == "mob" || ent.Kind == "enemy" {
				mobType := ent.Type
				if mobType == "" {
					mobType = "mob_standard"
				}
				enemyDefeats[mobType]++
				enemiesDefeated++
			}
		}

		// 2. Distribui a coleta proporcionalmente entre os recursos exigidos (ex: trigo + madeira)
		obj := boot.ActiveLevel.Objective
		if obj != nil && len(obj.Resources) > 0 {
			totalReq := 0
			for _, reqQty := range obj.Resources {
				totalReq += reqQty
			}
			if totalReq <= 0 {
				totalReq = goal
			}
			for resName, reqQty := range obj.Resources {
				if progress >= goal {
					collection[resName] = reqQty
				} else {
					collection[resName] = int(float64(progress) * (float64(reqQty) / float64(totalReq)))
				}
			}
		}
	}

	if len(collection) == 0 {
		res := run.ObjectiveResource
		if res == "" {
			res = "wheat"
		}
		collection[res] = progress
	}

	if enemiesDefeated == 0 {
		enemyDefeats["mob_standard"] = 2
		enemiesDefeated = 2
	}

	telemetryOutcome := "playing"
	if outcome == "settle" {
		telemetryOutcome = "completed"
	}

	heroAliveKills := make(map[string]int)
	if boot != nil && boot.Composition != nil {
		for _, slot := range boot.Composition.Slots {
			if slot.CharacterId != "" {
				heroAliveKills[slot.CharacterId] = enemiesDefeated
			}
		}
	}

	telemetry := &RunTelemetry{
		SchemaVersion:      1,
		CombatIntegrity:    []any{},
		ExtensionSignature: false,
		Metrics: TelemetryMetrics{
			DamageDealt:    float64(progress * 15),
			DamageReceived: float64(progress * 2),
			Collection:     collection,
		},
		Visibility: VisibilityStats{
			ActiveMs: ms,
		},
		ClientRunId:        run.ClientRunId,
		Outcome:            telemetryOutcome,
		DurationMs:         ms,
		ResourcesCollected: progress,
		CoinsEarned:        int64(progress / 2),
		ArenaCleared:       true,
		EnemiesDefeated:    enemiesDefeated,
		EnemyDefeats:       enemyDefeats,
		HeroAliveKills:     heroAliveKills,
	}

	return RunEnvelope{
		HolderId:        holderId,
		Outcome:         outcome,
		Progress:        progress,
		Telemetry:       telemetry,
		StructureEvents: events,
	}
}

// recordRunSuccess processa os ganhos de EXP, Gold e recursos ao vencer a fase
func (b *Bot) recordRunSuccess(levelNum int, resp *RunCompleteResponse) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.status.Kills += 5
	b.status.Gold += resp.GoldReward
	if resp.GoldBalance > 0 {
		b.status.Gold = int64(resp.GoldBalance)
	}

	// Atualiza saldos de recursos no status
	for res, val := range resp.ResourceBalances {
		b.status.Extra[res] = val
	}

	// Calcula EXP/h com base no tempo decorrido
	totalSec := time.Since(b.startTime).Seconds()
	if totalSec > 0 {
		b.status.ExpPerHour = (float64(b.status.Kills) * 120.0 / totalSec) * 3600.0
	}
	b.status.DurationSec = int(totalSec)
	b.status.UpdatedAt = time.Now().UnixMilli()

	logMsg := fmt.Sprintf("✅ Fase %d Concluída! +%d Gold", levelNum, resp.GoldReward)
	if resp.RewardQuantity > 0 {
		logMsg += fmt.Sprintf(" | +%d %s", resp.RewardQuantity, resp.RewardResource)
	}
	b.addLogLocked(logMsg)
}

// ----------------- ROTINAS DE MANUTENÇÃO AUTOMÁTICA -----------------

// processChests abre baús gratuitos, créditos acumulados e baús de herói quando houver saldo
func (b *Bot) processChests() {
	// 1. Verifica e abre baú inicial do onboarding se pendente
	_ = b.client.OpenStarterChest()

	// 2. Consulta lista de baús
	chests, err := b.client.GetChests()
	if err != nil {
		return
	}

	for _, c := range chests {
		// Abre se tiver créditos acumulados ou custo zero
		if c.OpeningAvailable && (c.Credits > 0 || c.CostQuantity <= 0) {
			resp, openErr := b.client.OpenChest(c.Code)
			if openErr == nil && resp != nil {
				b.addLog(fmt.Sprintf("🎁 Baú '%s' aberto com sucesso!", c.Name))
				if len(resp.Characters) > 0 {
					for _, hero := range resp.Characters {
						b.addLog(fmt.Sprintf("🎉 NOVO HERÓI OBTIDO: %s (%s ★%d)!", hero.Name, hero.Rarity, hero.Stars))
					}
					b.processTeamSlots()
				}
			}
			time.Sleep(1 * time.Second)
			continue
		}

		// Se for Baú de Herói (hero-chest) e o bot tiver gemas suficientes
		if c.Code == "hero-chest" && c.OpeningAvailable && c.CostQuantity > 0 {
			b.mu.Lock()
			currentGems := int64(0)
			if g, ok := b.status.Extra["gems"].(int64); ok {
				currentGems = g
			} else if g, ok := b.status.Extra["gems"].(int); ok {
				currentGems = int64(g)
			} else if g, ok := b.status.Extra["gems"].(float64); ok {
				currentGems = int64(g)
			}
			b.mu.Unlock()

			if currentGems >= int64(c.CostQuantity) || c.GemBalance >= int64(c.CostQuantity) {
				b.addLog(fmt.Sprintf("💎 Abrindo Baú de Herói (%d Gemas)...", c.CostQuantity))
				resp, openErr := b.client.OpenChest(c.Code)
				if openErr == nil && resp != nil {
					for _, hero := range resp.Characters {
						b.addLog(fmt.Sprintf("🎉 NOVO HERÓI RECRUTADO: %s (%s ★%d)!", hero.Name, hero.Rarity, hero.Stars))
					}
					b.processTeamSlots()
				} else if openErr != nil {
					log.Printf("[%s] Erro ao abrir Baú de Herói: %v", b.ID(), openErr)
				}
				time.Sleep(1 * time.Second)
			}
		}
	}
}

// processMissions resgata diárias, semanais e conquistas
func (b *Bot) processMissions() {
	// 0. Missões de Iniciante (Beginner Missions)
	beginner, err := b.client.GetBeginnerMissions()
	if err == nil && beginner != nil && beginner.Enrolled {
		for _, chain := range beginner.Chains {
			if !chain.Unlocked {
				continue
			}
			for _, m := range chain.Missions {
				if m.Status == "completed" {
					choice := ""
					if m.Reward != nil && len(m.Reward.Choices) > 0 {
						choice = m.Reward.Choices[0]
					}
					claimErr := b.client.ClaimBeginnerMission(m.Code, choice)
					if claimErr == nil {
						b.addLog(fmt.Sprintf("🌟 Missão de Iniciante resgatada: %s", m.Code))
					}
					time.Sleep(600 * time.Millisecond)
				}
			}
		}
	}

	// 1. Diárias
	daily, err := b.client.GetDailyMissions()
	if err == nil && daily != nil {
		for _, m := range daily.Missions {
			if m.Completed && !m.Claimed {
				_ = b.client.ClaimDailyMission(m.Code)
				b.addLog(fmt.Sprintf("🎯 Missão diária resgatada: %s", m.Name))
				time.Sleep(600 * time.Millisecond)
			}
		}
	}

	// 2. Semanais
	weekly, err := b.client.GetWeeklyMissions()
	if err == nil && weekly != nil {
		for _, r := range weekly.Rewards {
			if r.Available && !r.Claimed {
				_ = b.client.ClaimWeeklyReward(r.ID)
				b.addLog("🏆 Recompensa semanal resgatada!")
				time.Sleep(600 * time.Millisecond)
			}
		}
	}

	// 3. Conquistas / Marcos
	milestones, err := b.client.GetMilestones()
	if err == nil && milestones != nil {
		for _, ms := range milestones.Milestones {
			if ms.Completed && !ms.Claimed {
				if claimErr := b.client.ClaimMilestone(ms.ID); claimErr == nil {
					b.addLog(fmt.Sprintf("⭐ Marco resgatado: %s", ms.Name))
	
				}

				time.Sleep(600 * time.Millisecond)
			}
		}
	}

	// 4. Bestiário
	bestiary, err := b.client.GetBestiary()
	if err == nil {
		for _, mob := range bestiary {
			if mob.Claimable {
				_ = b.client.ClaimBestiary(mob.Code)
				b.addLog(fmt.Sprintf("📖 Recompensa do Bestiário: %s", mob.Name))
				time.Sleep(600 * time.Millisecond)
			}
		}
	}
}

// processBattlePass resgata missões e níveis liberados do Passe
func (b *Bot) processBattlePass() {
	bp, err := b.client.GetBattlePass()
	if err != nil || bp == nil {
		return
	}

	// Resgata missões concluídas do passe em lote
	var missionClaims []any
	for _, m := range append(append(bp.Daily, bp.Weekly...), bp.SeasonMissions...) {
		if m.Completed && !m.Claimed {
			missionClaims = append(missionClaims, map[string]string{"missionId": m.ID})
		}
	}
	if len(missionClaims) > 0 {
		_ = b.client.ClaimBattlePassMissions(missionClaims)
		b.addLog(fmt.Sprintf("🎟️ %d missões do Passe de Batalha resgatadas!", len(missionClaims)))
		time.Sleep(1 * time.Second)
	}

	// Resgata níveis disponíveis da trilha gratuita
	var levelClaims []any
	for _, cell := range bp.Track {
		if cell.Available && !cell.Claimed && cell.Tier == "free" {
			levelClaims = append(levelClaims, map[string]any{"level": cell.Level, "tier": cell.Tier})
		}
	}
	if len(levelClaims) > 0 {
		_ = b.client.ClaimBattlePassLevels(levelClaims)
		b.addLog(fmt.Sprintf("🎖️ %d níveis do Passe de Batalha resgatados!", len(levelClaims)))
	}
}

// processTalents avalia e compra melhorias acessíveis na Árvore de Talentos
func (b *Bot) processTalents() {
	talents := []string{
		"wheat_harvest", "wood_extraction", "mining", "fortune",
		"hero_power", "hero_vitality", "hero_agility", "hero_collection",
		"team_slot_4", "team_slot_5", "hero_inventory", "item_inventory",
	}

	for _, code := range talents {
		prev, err := b.client.GetTalentPreview(code)
		if err != nil || prev == nil {
			continue
		}

		if prev.Unlocked && !prev.AtMaxLevel && prev.Affordable {
			res, buyErr := b.client.PurchaseTalent(code, newUUID())
			if buyErr == nil && res != nil {
				b.addLog(fmt.Sprintf("🌱 Talento evoluído: %s -> Nv. %d", prev.Name, res.Level))
				time.Sleep(800 * time.Millisecond)
			}
		}
	}
}

// processHeroAscension busca cópias idênticas e sobe as estrelas do herói
func (b *Bot) processHeroAscension() {
	chars, _, err := b.client.GetCharacters()
	if err != nil || len(chars) < 2 {
		return
	}

	// Agrupa heróis desbloqueados por (code + rarity + stars)
	type groupKey struct {
		code   string
		rarity string
		stars  int
	}
	groups := make(map[groupKey][]CharacterSummary)

	for _, c := range chars {
		if !c.Locked && c.Stars < 5 {
			k := groupKey{code: c.Code, rarity: c.Rarity, stars: c.Stars}
			groups[k] = append(groups[k], c)
		}
	}

	for _, list := range groups {
		if len(list) >= 2 {
			mainHero := list[0]
			dup := list[1]

			// A duplicata não pode estar na equipe ativa
			if dup.InTeam && !mainHero.InTeam {
				mainHero, dup = dup, mainHero
			}
			if dup.InTeam {
				continue
			}

			prev, prevErr := b.client.PreviewAscension(mainHero.ID, dup.ID, 0)
			if prevErr == nil && prev != nil && prev.Affordable && !prev.DuplicateLocked && !prev.DuplicateInTeam {
				res, confErr := b.client.ConfirmAscension(mainHero.ID, dup.ID, 0)
				if confErr == nil && res != nil {
					if res.Succeeded {
						b.addLog(fmt.Sprintf("⭐ ASCENSÃO! %s atingiu %d★!", mainHero.Name, res.TargetStars))
					} else {
						b.addLog(fmt.Sprintf("Ascensão de %s falhou (duplicata preservada).", mainHero.Name))
					}
					time.Sleep(1 * time.Second)
				}
			}
		}
	}
}

// processTeamSlots gerencia e preenche as vagas livres da equipe com as melhores classes e heróis
func (b *Bot) processTeamSlots() {
	chars, maxSlots, err := b.client.GetCharacters()
	if err != nil || len(chars) == 0 {
		return
	}

	if maxSlots <= 0 {
		maxSlots = 3
	}

	var activeTeam []string
	var availableCandidates []CharacterSummary
	inTeamMap := make(map[string]bool)

	for _, c := range chars {
		if c.InTeam || c.TeamSlot > 0 {
			activeTeam = append(activeTeam, c.ID)
			inTeamMap[c.ID] = true
		}
	}

	// Se a equipe já estiver cheia, não precisa alterar
	if len(activeTeam) >= maxSlots {
		return
	}

	for _, c := range chars {
		if !inTeamMap[c.ID] && !c.Locked {
			availableCandidates = append(availableCandidates, c)
		}
	}

	if len(availableCandidates) == 0 {
		return
	}

	// Prioridade estratégica:
	// 1. Ladino (farmer) -> +100% Gold por abate
	// 2. Arqueira (runner) -> Alta velocidade de abate
	// 3. Sacerdote (priest) -> Buff de dano cumulativo
	// 4. Mago (wizard) -> Dano em área
	// 5. Maior raridade e estrelas
	sort.SliceStable(availableCandidates, func(i, j int) bool {
		score := func(h CharacterSummary) int {
			s := 0
			switch h.Code {
			case "farmer":
				s += 2000
			case "runner":
				s += 1200
			case "priest":
				s += 1000
			case "wizard":
				s += 800
			case "woodcutter":
				s += 600
			case "knight":
				s += 500
			default:
				s += 100
			}
			s += h.Stars * 50
			s += h.Level * 10
			return s
		}
		return score(availableCandidates[i]) > score(availableCandidates[j])
	})

	needed := maxSlots - len(activeTeam)
	var newHeroes []string
	for i := 0; i < needed && i < len(availableCandidates); i++ {
		candidate := availableCandidates[i]
		activeTeam = append(activeTeam, candidate.ID)
		newHeroes = append(newHeroes, fmt.Sprintf("%s (%s)", candidate.Name, candidate.Code))
	}

	if len(newHeroes) > 0 {
		saveErr := b.client.SaveTeam(activeTeam)
		if saveErr != nil {
			log.Printf("[%s] Erro ao salvar equipe: %v", b.ID(), saveErr)
			return
		}
		b.addLog(fmt.Sprintf("👥 Equipe atualizada com sucesso (%d/%d heróis)! Adicionados: %s",
			len(activeTeam), maxSlots, strings.Join(newHeroes, ", ")))
	}
}

// processDismantle desmonta equipamentos comuns não equipados e repetidos
func (b *Bot) processDismantle() {
	items, err := b.client.GetItems()
	if err != nil {
		return
	}

	var toDismantle []string
	for _, it := range items {
		if it.Rarity == "common" && it.EquippedHeroId == "" && !it.Locked && it.RefineLevel == 0 {
			toDismantle = append(toDismantle, it.ID)
			if len(toDismantle) >= 20 {
				break
			}
		}
	}

	if len(toDismantle) > 0 {
		_ = b.client.DismantleItems(toDismantle)
		b.addLog(fmt.Sprintf("🔨 %d itens comuns desmontados em fragmentos de forja.", len(toDismantle)))
	}
}

// processCrystalFusion funde cristais de ascensão em tiers superiores
func (b *Bot) processCrystalFusion() {
	for _, tier := range []string{"common", "rare", "epic"} {
		_ = b.client.FuseCrystals(tier)
	}
}

// processAutoEquip equipa automaticamente os melhores itens disponíveis nos heróis da equipe ativa
func (b *Bot) processAutoEquip() {
	chars, _, err := b.client.GetCharacters()
	if err != nil || len(chars) == 0 {
		return
	}

	items, err := b.client.GetItems()
	if err != nil || len(items) == 0 {
		return
	}

	var activeHeroes []CharacterSummary
	for _, c := range chars {
		if c.InTeam || c.TeamSlot > 0 {
			activeHeroes = append(activeHeroes, c)
		}
	}
	if len(activeHeroes) == 0 {
		return
	}

	freeItems := make(map[string][]ItemSummary)
	for _, it := range items {
		if it.Locked {
			continue
		}
		if it.EquippedHeroId == "" && it.EquippedCharacterId == "" {
			freeItems[it.Slot] = append(freeItems[it.Slot], it)
		}
	}

	rarityScore := func(r string) int {
		switch strings.ToLower(r) {
		case "mythic":
			return 500
		case "legendary":
			return 300
		case "epic":
			return 150
		case "rare":
			return 50
		default:
			return 10
		}
	}

	calcScore := func(it ItemSummary) float64 {
		score := it.Power
		if score <= 0 {
			score = float64(rarityScore(it.Rarity) + (it.RefineLevel * 20) + (it.EnhancementLevel * 30))
		}
		return score
	}

	for _, slotItems := range freeItems {
		sort.SliceStable(slotItems, func(i, j int) bool {
			return calcScore(slotItems[i]) > calcScore(slotItems[j])
		})
	}

	slots := []string{"weapon", "armor", "helmet", "boots", "accessory", "ring", "tool"}
	for _, hero := range activeHeroes {
		for _, slot := range slots {
			candidates := freeItems[slot]
			if len(candidates) == 0 {
				continue
			}

			var bestItem *ItemSummary
			bestIdx := -1
			for idx, it := range candidates {
				if it.RequiredCharacterCode != "" && it.RequiredCharacterCode != hero.Code {
					continue
				}
				bestItem = &candidates[idx]
				bestIdx = idx
				break
			}

			if bestItem != nil {
				equipErr := b.client.EquipItem(hero.ID, slot, bestItem.ID)
				if equipErr == nil {
					b.addLog(fmt.Sprintf("⚔️ Equipado: %s colocou %s (+%d) no slot %s", hero.Name, bestItem.Name, bestItem.RefineLevel, slot))
					freeItems[slot] = append(candidates[:bestIdx], candidates[bestIdx+1:]...)
					time.Sleep(400 * time.Millisecond)
				}
			}
		}
	}
}

// processForgeEnhancement aprimora equipamentos na Forja até o nível seguro (+5/+6)
func (b *Bot) processForgeEnhancement() {
	items, err := b.client.GetItems()
	if err != nil || len(items) == 0 {
		return
	}

	enhancedCount := 0
	maxPerCycle := 2

	for _, it := range items {
		if enhancedCount >= maxPerCycle {
			break
		}
		isEquipped := it.EquippedHeroId != "" || it.EquippedCharacterId != ""
		if (!isEquipped && it.Rarity == "common") || it.EnhancementLevel >= 6 {
			continue
		}

		prev, err := b.client.GetEnhancementPreview(it.ID)
		if err != nil || prev == nil {
			continue
		}

		if prev.Affordable && prev.TargetLevel <= 6 && prev.SuccessRate >= 0.40 {
			res, enhErr := b.client.EnhanceItem(it.ID, false)
			if enhErr == nil && res != nil {
				enhancedCount++
				if res.Succeeded {
					b.addLog(fmt.Sprintf("✨ Forja: %s aprimorado para +%d (taxa: %.0f%%)!", it.Name, res.EnhancementLevel, prev.SuccessRate*100))
				} else {
					b.addLog(fmt.Sprintf("Forja: tentativa em %s (+%d) não subiu de nível.", it.Name, prev.CurrentLevel))
				}
				time.Sleep(800 * time.Millisecond)
			}
		}
	}
}

// processMythicWell gerencia o Poço Mítico (definição de alvo, depósito de itens e resgate)
func (b *Bot) processMythicWell() {
	well, err := b.client.GetMythicWell()
	if err != nil || well == nil {
		return
	}

	// 1. Resgate se disponível
	if well.Claimable || (well.Goal > 0 && well.Progress >= well.Goal) {
		res, claimErr := b.client.ClaimMythicWell()
		if claimErr == nil && res != nil {
			b.addLog(fmt.Sprintf("🏺 POÇO MÍTICO: Recompensa mítica resgatada com sucesso! (%s)", res.Item.Name))
			time.Sleep(800 * time.Millisecond)
		}
	}

	// 2. Alvo do poço
	if well.TargetDefinitionID == "" && len(well.EligibleItems) > 0 {
		target := well.EligibleItems[0]
		_ = b.client.SetMythicWellTarget(target.ID)
		b.addLog(fmt.Sprintf("🏺 Poço Mítico: Alvo definido para %s", target.Name))
		time.Sleep(500 * time.Millisecond)
	}

	// 3. Oferenda de itens comuns se não atingiu a meta
	if well.Goal > 0 && well.Progress < well.Goal {
		items, err := b.client.GetItems()
		if err == nil {
			var toDeposit []string
			for _, it := range items {
				if it.Rarity == "common" && it.EquippedHeroId == "" && it.EquippedCharacterId == "" && !it.Locked && it.RefineLevel == 0 {
					toDeposit = append(toDeposit, it.ID)
					if len(toDeposit) >= 5 {
						break
					}
				}
			}
			if len(toDeposit) > 0 {
				depErr := b.client.DepositMythicWell(toDeposit)
				if depErr == nil {
					b.addLog(fmt.Sprintf("🏺 Poço Mítico: %d itens comuns ofertados.", len(toDeposit)))
				}
			}
		}
	}
}

// processClan gerencia missões, doações e recompensas da Guilda
func (b *Bot) processClan() {
	clanResp, err := b.client.GetMyClan()
	if err != nil || clanResp == nil || clanResp.Clan == nil {
		return
	}

	clan := clanResp.Clan

	// 1. Resgata baús/recompensas de expedição
	_ = b.client.ClaimClanRewards(clan.ID)

	// 2. Contribui com missões do clã
	for _, m := range clanResp.Missions {
		if m.Completed || m.Progress >= m.Target {
			continue
		}

		donateQty := m.Target - m.Progress
		if donateQty > 100 {
			donateQty = 100
		}
		if donateQty <= 0 {
			continue
		}

		donErr := b.client.DonateClanMission(clan.ID, m.Code, donateQty)
		if donErr == nil {
			b.addLog(fmt.Sprintf("🏰 Clã: Doação de %d enviada na missão '%s'", donateQty, m.Code))
			time.Sleep(600 * time.Millisecond)
		}
	}
}

// processPrestige rastreia o prestígio e atualiza as métricas da conta
func (b *Bot) processPrestige() {
	prest, err := b.client.GetPrestige()
	if err != nil || prest == nil {
		return
	}

	b.mu.Lock()
	b.status.Extra["prestigePoints"] = prest.Points
	if prest.Tier != nil {
		b.status.Extra["prestigeLevel"] = prest.Tier.Level
		b.status.Extra["prestigeTierName"] = prest.Tier.Name
	}
	b.mu.Unlock()
}

// processInfiniteChallenge participa da Torre de Desafio Infinito se houver ingressos
func (b *Bot) processInfiniteChallenge() {
	infStatus, err := b.client.GetInfiniteStatus()
	if err != nil || infStatus == nil || !infStatus.Available {
		return
	}

	if infStatus.Tokens <= 0 {
		return
	}

	b.addLog(fmt.Sprintf("🌌 Desafio Infinito: Iniciando Torre (Tokens: %d)...", infStatus.Tokens))
	run, err := b.client.StartInfiniteRun(b.holderId)
	if err != nil || run == nil {
		return
	}

	targetWaves := 5
	for w := 1; w <= targetWaves; w++ {
		time.Sleep(1200 * time.Millisecond)
		_ = b.client.SendInfiniteEvent(run.ID, "wave_completed", w, w, 10)
	}

	endErr := b.client.EndInfiniteRun(run.ID, "cleared")
	if endErr == nil {
		b.addLog(fmt.Sprintf("🌌 Desafio Infinito: Torre concluída até a onda %d com sucesso!", targetWaves))
	}
}

// processPvP resgata missões da Arena PvP e alinha a defesa
func (b *Bot) processPvP() {
	hub, err := b.client.GetPvPHub()
	if err != nil || hub == nil {
		return
	}

	b.mu.Lock()
	b.status.Extra["pvpRankPoints"] = hub.RankPoints
	b.status.Extra["pvpTier"] = hub.Tier
	b.status.Extra["pvpCoins"] = hub.ArenaCoins
	b.mu.Unlock()

	// 1. Resgata missões diárias
	for _, m := range hub.DailyMissions {
		if m.Completed && !m.Claimed {
			claimErr := b.client.ClaimPvPMission("daily", m.Code)
			if claimErr == nil {
				b.addLog(fmt.Sprintf("⚔️ PvP: Missão diária resgatada: %s", m.Code))
				time.Sleep(500 * time.Millisecond)
			}
		}
	}

	// 2. Resgata missões semanais
	for _, m := range hub.WeeklyMissions {
		if m.Completed && !m.Claimed {
			claimErr := b.client.ClaimPvPMission("weekly", m.Code)
			if claimErr == nil {
				b.addLog(fmt.Sprintf("⚔️ PvP: Missão semanal resgatada: %s", m.Code))
				time.Sleep(500 * time.Millisecond)
			}
		}
	}
}

// initWebSocket estabelece conexão WebSocket passiva para keepalive e contagem de players
func (b *Bot) initWebSocket() {
	b.socket = b.client.CreateWebSocketClient(
		func(data []byte) {
			// Manipulador de mensagens WebSocket recebidas
			var streamMsg struct {
				Type   string `json:"type"`
				Online int    `json:"online,omitempty"`
			}
			if err := json.Unmarshal(data, &streamMsg); err == nil {
				if streamMsg.Type == "online" && streamMsg.Online > 0 {
					b.mu.Lock()
					b.status.Extra["onlinePlayers"] = streamMsg.Online
					b.mu.Unlock()
				}
			}
		},
		func(err error) {
			log.Printf("[%s] WebSocket desconectado: %v (auto-reconexão no client)", b.ID(), err)
		},
	)

	// Conecta em background sem travar
	go func() {
		_ = b.socket.Connect()
	}()
}

// ----------------- HELPERS & THREAD-SAFE STATUS -----------------

func (b *Bot) updateStatus(fn func(s *engine.AccountStatus)) {
	b.mu.Lock()
	defer b.mu.Unlock()
	fn(&b.status)
	b.status.UpdatedAt = time.Now().UnixMilli()
}

func (b *Bot) addLog(msg string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.addLogLocked(msg)
}

func (b *Bot) addLogLocked(msg string) {
	timestamp := time.Now().Format("15:04:05")
	entry := fmt.Sprintf("[%s] %s", timestamp, msg)
	b.status.Logs = append(b.status.Logs, entry)
	if len(b.status.Logs) > 15 {
		b.status.Logs = b.status.Logs[len(b.status.Logs)-15:]
	}
}

func (b *Bot) GetStatus() engine.AccountStatus {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.status
}

func (b *Bot) ExecuteAction(action string, payload map[string]any) (any, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	switch action {
	case "autoHunt":
		b.config.Settings.AutoHunt = !b.config.Settings.AutoHunt
		statusMsg := "pausado"
		if b.config.Settings.AutoHunt {
			statusMsg = "ativado"
		}
		b.addLogLocked(fmt.Sprintf("Comando: Auto-Hunt %s.", statusMsg))
		return map[string]any{"autoHunt": b.config.Settings.AutoHunt}, nil

	case "claimAll":
		go b.processMaintenance()
		b.addLogLocked("Comando: Resgate de baús, missões e passe acionado.")
		return "Resgates disparados com sucesso!", nil

	case "advanceLevel":
		if b.lastBootstrap != nil && b.lastBootstrap.Progress != nil {
			highest := b.lastBootstrap.Progress.HighestUnlockedLevel
			b.addLogLocked(fmt.Sprintf("Comando: Avançando para fase mais alta (%d).", highest))
			for _, lvl := range b.lastBootstrap.Levels {
				if lvl.Number == highest {
					_ = b.client.SetActiveLevel(lvl.Code)
					break
				}
			}
		}
		return "Avanço de fase requisitado.", nil

	case "emergencyStop":
		b.circuitBreaker.Trip("Parada de emergência acionada pelo usuário via Web Dashboard")
		b.config.Settings.AutoHunt = false
		b.status.Status = "PAUSADO (Manual)"
		b.addLogLocked("🚨 PARADA DE EMERGÊNCIA ACIONADA. Todas as ações foram suspensas.")
		return "Bot pausado com sucesso.", nil

	default:
		return nil, fmt.Errorf("ação desconhecida: %s", action)
	}
}

func (b *Bot) SupplyTicket(ticket, wsURL string) error {
	return nil
}

func (b *Bot) SupplyTurnstileToken(token string) error {
	return nil
}
