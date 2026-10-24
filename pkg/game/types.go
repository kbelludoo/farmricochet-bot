package game

// AccountConfig modelo de configuração carregado do accounts.json
type AccountConfig struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Enabled     bool            `json:"enabled"`
	Credentials GameCredentials `json:"credentials"`
	Settings    FarmBotSettings `json:"settings"`
}

// GameCredentials armazena os cookies e tokens para autenticação no Farm Ricochet
type GameCredentials struct {
	Cookie        string `json:"cookie"`        // Header Cookie completo (ex: connect.sid=... ou session=...)
	SessionToken  string `json:"sessionToken"`  // Token JWT / Google Credential alternativo se disponível
	Email         string `json:"email"`         // E-mail da conta Google para referência local
}

// FarmBotSettings configurações de automação completa para o Farm Ricochet
type FarmBotSettings struct {
	AutoHunt         bool   `json:"autoHunt"`         // Execução contínua de fases/runs da campanha
	TargetLevelCode  string `json:"targetLevelCode"`  // Código da fase alvo (vazio = avança continuamente)
	AutoAdvancePhase bool   `json:"autoAdvancePhase"` // Avança automaticamente para a próxima fase desbloqueada
	AutoChests       bool   `json:"autoChests"`       // Abre baús gratuitos e créditos VIP/diários acumulados
	AutoMissions     bool   `json:"autoMissions"`     // Resgata diárias, semanais, conquistas e bestiário
	AutoBattlePass   bool   `json:"autoBattlePass"`   // Resgata missões e níveis do Passe de Batalha
	AutoTalents      bool   `json:"autoTalents"`      // Compra melhorias na Árvore de Talentos automaticamente
	AutoAscension    bool   `json:"autoAscension"`    // Sobe estrelas de heróis duplicados compatíveis
	AutoTransmute    bool   `json:"autoTransmute"`    // Altar: repara e transmuta lotes de 10 heróis
	AutoDismantle    bool   `json:"autoDismantle"`    // Desmonta equipamentos comuns repetidos para forja
	AutoFuseCrystals bool   `json:"autoFuseCrystals"` // Funde cristais de ascensão inferiores
	AutoLootGoblin   bool   `json:"autoLootGoblin"`   // Derrota goblins de espólio quando surgem na partida
	AutoEquip        bool   `json:"autoEquip"`        // Equipa automaticamente os melhores itens nos heróis ativos
	AutoForge        bool   `json:"autoForge"`        // Aprimora itens equipados na Forja (+1 a +6)
	AutoMythicWell   bool   `json:"autoMythicWell"`   // Alimenta e resgata itens do Poço Mítico
	AutoClan         bool   `json:"autoClan"`         // Doações de missões e resgate de baús de Clã
	AutoPrestige     bool   `json:"autoPrestige"`     // Rastreamento e coleta de marcos de Prestígio
	AutoInfinite     bool   `json:"autoInfinite"`     // Execução da Torre de Desafio Infinito
	AutoPvP          bool   `json:"autoPvP"`          // Resgate de missões diárias/semanais da Arena PvP
	AutoTuneSpeed    bool   `json:"autoTuneSpeed"`    // Auto-tuner adaptativo para encontrar o teto máximo de velocidade
	SpeedMultiplier  float64 `json:"speedMultiplier"` // Fator de ajuste de tempo da fase (padrão: 1.0)
}

// DefaultSettings retorna configurações padrão recomendadas
func DefaultSettings() FarmBotSettings {
	return FarmBotSettings{
		AutoHunt:         true,
		AutoAdvancePhase: true,
		AutoChests:       true,
		AutoMissions:     true,
		AutoBattlePass:   true,
		AutoTalents:      true,
		AutoAscension:    true,
		AutoTransmute:    false, // Altar opcional para evitar queima acidental
		AutoDismantle:    true,
		AutoFuseCrystals: true,
		AutoLootGoblin:   true,
		AutoEquip:        true,
		AutoForge:        true,
		AutoMythicWell:   true,
		AutoClan:         true,
		AutoPrestige:     true,
		AutoInfinite:     true,
		AutoPvP:          true,
		AutoTuneSpeed:    true,
		SpeedMultiplier:  1.0,
	}
}

// ----------------- SESSÃO & ADMISSÃO -----------------

type SessionLeaseRequest struct {
	HolderId string `json:"holderId"`
	Force    bool   `json:"force,omitempty"`
}

type SessionLeaseResponse struct {
	HolderId  string `json:"holderId"`
	ExpiresAt string `json:"expiresAt"`
}

type GameAdmissionResponse struct {
	MaxOnline int  `json:"maxOnline"`
	Queued    bool `json:"queued"`
	Position  int  `json:"position,omitempty"`
}

// ----------------- BOOTSTRAP DO JOGO -----------------

type BootstrapResponse struct {
	Account     *AccountData     `json:"account"`
	ActiveLevel *ActiveLevel     `json:"activeLevel"`
	Levels      []LevelInfo      `json:"levels"`
	Composition *CompositionData `json:"composition"`
	Progress    *ProgressData    `json:"progress"`
	Bestiary    []BestiaryEntry  `json:"bestiary,omitempty"`
}

type AccountData struct {
	ID                 string   `json:"id"`
	DisplayName        string   `json:"displayName"`
	Email              string   `json:"email"`
	CommonCurrency     int64    `json:"commonCurrency"`  // Gold
	PremiumCurrency    int64    `json:"premiumCurrency"` // Gemas
	Wheat              int64    `json:"wheat"`
	Wood               int64    `json:"wood"`
	Stone              int64    `json:"stone"`
	HeroicEssence      int64    `json:"heroicEssence"`
	ForgeFragments     int64    `json:"forgeFragments"`
	IsAdmin            bool     `json:"isAdmin"`
	Vip                *VipData `json:"vip,omitempty"`
}

type VipData struct {
	Active bool   `json:"active"`
	EndsAt string `json:"endsAt,omitempty"`
}

type ActiveLevel struct {
	ID               string       `json:"id"`
	Code             string       `json:"code"`
	Number           int          `json:"number"`
	Name             string       `json:"name"`
	LevelVersionId   string       `json:"levelVersionId"`
	MapId            string       `json:"mapId"`
	MapVersionId     string       `json:"mapVersionId"`
	ObjectiveType    string       `json:"objectiveType"`
	RecommendedPower int          `json:"recommendedPower"`
	Objective        *Objective   `json:"objective"`
	MapData          *MapData     `json:"mapData"`
	Entities         []EntityData `json:"entities"`
}

type Objective struct {
	Resource  string         `json:"resource"`
	Amount    int            `json:"amount"`
	Resources map[string]int `json:"resources,omitempty"`
}

type MapData struct {
	Cols   int            `json:"cols"`
	Rows   int            `json:"rows"`
	Spawns map[string]any `json:"spawns,omitempty"`
}

type EntityData struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`      // "hero", "structure", "enemy"
	Type      string `json:"type"`      // ex: "wheat_crop", "wood_tree", "stone_rock", "runner"
	Cell      int    `json:"cell"`
	CharacterId string `json:"characterId,omitempty"`
}

type LevelInfo struct {
	ID               string `json:"id"`
	Code             string `json:"code"`
	Number           int    `json:"number"`
	Name             string `json:"name"`
	Unlocked         bool   `json:"unlocked"`
	Completed        bool   `json:"completed"`
	RecommendedPower int    `json:"recommendedPower"`
}

type CompositionData struct {
	Slots []CompositionSlot `json:"slots"`
}

type CompositionSlot struct {
	CharacterId        string         `json:"characterId"`
	Code               string         `json:"code"` // "knight", "runner", "priest", etc.
	Level              int            `json:"level"`
	Experience         int64          `json:"experience"`
	ExperienceRequired int64          `json:"experienceRequired"`
	HP                 float64        `json:"hp"`
	Speed              float64        `json:"speed"`
	Damage             float64        `json:"damage"`
	Collection         float64        `json:"collection"`
	Rarity             string         `json:"rarity"`
	RarityName         string         `json:"rarityName"`
	RarityColor        string         `json:"rarityColor"`
	Stars              int            `json:"stars"`
	Combat             map[string]any `json:"combat,omitempty"`
}

type ProgressData struct {
	HighestUnlockedLevel int    `json:"highestUnlockedLevel"`
	ActiveLevelCode      string `json:"activeLevelCode"`
}

type BestiaryEntry struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	Stage     int    `json:"stage"`
	Claimable bool   `json:"claimable"`
}

// ----------------- RUNS / CAMPANHA / FARM -----------------

type RunStartRequest struct {
	ClientRunId        string `json:"clientRunId"`
	HolderId           string `json:"holderId"`
	PersistenceVersion int    `json:"persistenceVersion"` // 2
	ClientRevision     int    `json:"clientRevision"`     // 1
	ClientRefreshToken string `json:"clientRefreshToken"` // ""
}

type RunStartResponse struct {
	ID                    string            `json:"id"`
	ClientRunId           string            `json:"clientRunId"`
	Seed                  int64             `json:"seed"`
	LevelId               string            `json:"levelId"`
	LevelVersionId        string            `json:"levelVersionId"`
	ObjectiveResource     string            `json:"objectiveResource"`
	ObjectiveAmount       int               `json:"objectiveAmount"`
	ObjectiveResources    map[string]int    `json:"objectiveResources"`
	DefeatAllEnemies      bool              `json:"defeatAllEnemies"`
	OnlineRatePerSecond   float64           `json:"onlineRatePerSecond"`
	AuthoritativeProgress int               `json:"authoritativeProgress"`
	ServerTime            string            `json:"serverTime"`
	StartedAt             string            `json:"startedAt"`
	LootGoblin            bool              `json:"lootGoblin"`
	OfflineReward         *RunOfflineReward `json:"offlineReward,omitempty"`
}

type RunOfflineReward struct {
	Resource string `json:"resource"`
	Quantity int    `json:"quantity"`
}

type RunEnvelope struct {
	HolderId        string           `json:"holderId"`
	Outcome         string           `json:"outcome,omitempty"` // "settle", "forfeit", "abandoned"
	Progress        int              `json:"progress"`
	Telemetry       *RunTelemetry    `json:"telemetry"`
	StructureEvents []StructureEvent `json:"structureEvents"`
}

type RunTelemetry struct {
	SchemaVersion      int             `json:"schemaVersion"` // 1
	CombatIntegrity    []any           `json:"combatIntegrity"`
	ExtensionSignature bool            `json:"extensionSignature"` // false
	Metrics            TelemetryMetrics `json:"metrics"`
	Visibility         VisibilityStats `json:"visibility"`
	ClientRunId        string          `json:"clientRunId"`
	Outcome            string          `json:"outcome"`
	DurationMs         int64           `json:"durationMs"`
	ResourcesCollected int             `json:"resourcesCollected"`
	CoinsEarned        int64           `json:"coinsEarned"`
	ArenaCleared       bool            `json:"arenaCleared"`
	EnemiesDefeated    int             `json:"enemiesDefeated"`
	EnemyDefeats       map[string]int  `json:"enemyDefeats"`
	HeroAliveKills     map[string]int  `json:"heroAliveKills"`
	StructureEvents    []StructureEvent `json:"structureEvents,omitempty"`
}

type TelemetryMetrics struct {
	DamageDealt    float64        `json:"damageDealt"`
	DamageReceived float64        `json:"damageReceived"`
	UsefulHealing  float64        `json:"usefulHealing"`
	Overhealing    float64        `json:"overhealing"`
	Shield         float64        `json:"shield"`
	Mitigation     float64        `json:"mitigation"`
	Collection     map[string]int `json:"collection"`
}

type VisibilityStats struct {
	ActiveMs           int64 `json:"activeMs"`
	BackgroundMs       int64 `json:"backgroundMs"`
	PictureInPictureMs int64 `json:"pictureInPictureMs"`
	OfflineMs          int64 `json:"offlineMs"`
}

type StructureEvent struct {
	EventId          string `json:"eventId"`
	LevelVersionId   string `json:"levelVersionId"`
	MapVersionId     string `json:"mapVersionId"`
	Sequence         int    `json:"sequence,omitempty"`
	StructureId      string `json:"structureId"`
	StructureCode    string `json:"structureCode"`
	StructureVersion int    `json:"structureVersion"`
	HeroId           string `json:"heroId"`
	EventType        string `json:"eventType"` // "production"
	OccurredAtMs     int64  `json:"occurredAtMs"`
	Resource         string `json:"resource"`
	Amount           int    `json:"amount"`
	Quantity         int    `json:"quantity,omitempty"`
}

type RunHeartbeatResponse struct {
	LeaseExpiresAt        string `json:"leaseExpiresAt"`
	Progress              int    `json:"progress"`
	AuthoritativeProgress int    `json:"authoritativeProgress"`
	ObjectiveAmount       int    `json:"objectiveAmount"`
}

type RunCompleteResponse struct {
	Accepted              bool                 `json:"accepted"`
	Outcome               string               `json:"outcome"` // "completed"
	AuthoritativeProgress int                  `json:"authoritativeProgress"`
	ObjectiveAmount       int                  `json:"objectiveAmount"`
	GoldBalance           float64              `json:"goldBalance"`
	GoldReward            int64                `json:"goldReward"`
	RewardResource        string               `json:"rewardResource"`
	RewardQuantity        int                  `json:"rewardQuantity"`
	ResourceRewards       map[string]int       `json:"resourceRewards"`
	ResourceBalances      map[string]int64     `json:"resourceBalances"`
	HeroExperience        []HeroExperienceGain `json:"heroExperience"`
	NextLevelId           string               `json:"nextLevelId"`
	NextLevelCode         string               `json:"nextLevelCode"`
	NextLevelNumber       int                  `json:"nextLevelNumber"`
	RetryAfterMs          int64                `json:"retryAfterMs,omitempty"`
}

type HeroExperienceGain struct {
	HeroId        string         `json:"heroId"`
	Level         int            `json:"level"`
	PreviousLevel int            `json:"previousLevel"`
	TotalXp       int64          `json:"totalXp"`
	EnemyXp       int64          `json:"enemyXp"`
	CompletionXp  int64          `json:"completionXp"`
	StatGains     map[string]any `json:"statGains,omitempty"`
}

// ----------------- ESPÓLIOS (RUN LOOT) -----------------

type RunLootResponse struct {
	Items     []RunLootItem  `json:"items"`
	Resources map[string]int `json:"resources"`
}

type RunLootItem struct {
	ID       string `json:"id"`
	Code     string `json:"code"`
	Name     string `json:"name"`
	Rarity   string `json:"rarity"`
	Quantity int    `json:"quantity"`
}

type RunLootClaimRequest struct {
	IdempotencyKey string `json:"idempotencyKey"`
}

// ----------------- BAÚS (CHESTS) -----------------

type ChestsResponse struct {
	Chests []ChestItem `json:"chests"`
}

type ChestItem struct {
	Code             string        `json:"code"` // "hero-chest", "item-chest", "equipment-chest"
	Name             string        `json:"name"`
	CostQuantity     int           `json:"costQuantity"`
	RewardType       string        `json:"rewardType"` // "character", "item"
	OpeningAvailable bool          `json:"openingAvailable"`
	Credits          int           `json:"credits"`
	GemBalance       int64         `json:"gemBalance"`
	Chances          []ChestChance `json:"chances"`
}

type ChestChance struct {
	Rarity      string  `json:"rarity"`
	RarityName  string  `json:"rarityName"`
	Color       string  `json:"color"`
	Probability float64 `json:"probability"`
}

type ChestOpenRequest struct {
	ClientRequestId string `json:"clientRequestId"`
}

type ChestOpenResponse struct {
	Characters []CharacterSummary `json:"characters,omitempty"`
	Items      []ItemSummary      `json:"items,omitempty"`
}

type StarterChestResponse struct {
	Available bool `json:"available"`
	Opened    bool `json:"opened"`
}

// ----------------- MISSÕES & CONQUISTAS -----------------

type DailyMissionsResponse struct {
	PeriodEnd string         `json:"periodEnd"`
	Missions  []DailyMission `json:"missions"`
}

type DailyMission struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	Metric    string `json:"metric"`
	Progress  int    `json:"progress"`
	Target    int    `json:"target"`
	Claimed   bool   `json:"claimed"`
	Completed bool   `json:"completed"`
}

type WeeklyMissionsResponse struct {
	ResetsAt string          `json:"resetsAt"`
	Missions []WeeklyMission `json:"missions"`
	Rewards  []WeeklyReward  `json:"rewards"`
}

type WeeklyMission struct {
	ID        string `json:"id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	Progress  int    `json:"progress"`
	Target    int    `json:"target"`
	Completed bool   `json:"completed"`
}

type WeeklyReward struct {
	ID        string `json:"id"`
	Threshold int    `json:"threshold"`
	Claimed   bool   `json:"claimed"`
	Available bool   `json:"available"`
}

type MilestonesResponse struct {
	Milestones []MilestoneItem `json:"milestones"`
}

type MilestoneItem struct {
	ID        string `json:"id"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	Progress  int    `json:"progress"`
	Target    int    `json:"target"`
	Claimed   bool   `json:"claimed"`
	Completed bool   `json:"completed"`
}

type BeginnerMissionsResponse struct {
	Enrolled       bool                   `json:"enrolled"`
	Completed      bool                   `json:"completed"`
	ClaimableCount int                    `json:"claimableCount"`
	Chains         []BeginnerMissionChain `json:"chains"`
}

type BeginnerMissionChain struct {
	Code      string            `json:"code"`
	Unlocked  bool              `json:"unlocked"`
	Completed bool              `json:"completed"`
	Missions  []BeginnerMission `json:"missions"`
}

type BeginnerMission struct {
	Code     string                 `json:"code"`
	Status   string                 `json:"status"` // "completed", "claimed", "active", "locked"
	Progress int                    `json:"progress"`
	Target   int                    `json:"target"`
	Reward   *BeginnerMissionReward `json:"reward,omitempty"`
}

type BeginnerMissionReward struct {
	Choices []string `json:"choices,omitempty"`
}

// ----------------- PASSE DE BATALHA -----------------

type BattlePassResponse struct {
	Season         BattlePassSeason  `json:"season"`
	Progress       BattlePassProgress`json:"progress"`
	Track          []BattlePassCell  `json:"track"`
	Daily          []BattlePassTask  `json:"daily"`
	Weekly         []BattlePassTask  `json:"weekly"`
	SeasonMissions []BattlePassTask  `json:"seasonMissions"`
}

type BattlePassSeason struct {
	SeasonNumber int    `json:"seasonNumber"`
	MaxLevel     int    `json:"maxLevel"`
	EndsAt       string `json:"endsAt"`
}

type BattlePassProgress struct {
	Level       int   `json:"level"`
	CurrentXp   int64 `json:"currentXp"`
	NextLevelXp int64 `json:"nextLevelXp"`
}

type BattlePassCell struct {
	Level     int    `json:"level"`
	Tier      string `json:"tier"` // "free", "premium"
	Claimed   bool   `json:"claimed"`
	Available bool   `json:"available"`
}

type BattlePassTask struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Progress  int    `json:"progress"`
	Target    int    `json:"target"`
	Claimed   bool   `json:"claimed"`
	Completed bool   `json:"completed"`
}

type BattlePassClaimBatchRequest struct {
	Claims []any `json:"claims"`
}

// ----------------- TALENTOS -----------------

type TalentPreview struct {
	Code                   string         `json:"code"`
	Name                   string         `json:"name"`
	EffectCode             string         `json:"effectCode"`
	CurrentLevel           int            `json:"currentLevel"`
	MaxLevel               int            `json:"maxLevel"`
	CurrentEffect          float64        `json:"currentEffect"`
	NextEffect             float64        `json:"nextEffect"`
	EffectPerLevel         float64        `json:"effectPerLevel"`
	Unlocked               bool           `json:"unlocked"`
	AtMaxLevel             bool           `json:"atMaxLevel"`
	Affordable             bool           `json:"affordable"`
	RequiredCampaignLevel  int            `json:"requiredCampaignLevel"`
	HighestUnlockedLevel   int            `json:"highestUnlockedLevel"`
	Balances               map[string]int `json:"balances"`
	Costs                  map[string]int `json:"costs"`
	Prerequisites          map[string]int `json:"prerequisites"`
}

type TalentPurchaseRequest struct {
	IdempotencyKey string `json:"idempotencyKey"`
}

type TalentPurchaseResponse struct {
	Code  string `json:"code"`
	Level int    `json:"level"`
}

// ----------------- PERSONAGENS & HERÓIS -----------------

type CharactersResponse struct {
	Characters   []CharacterSummary `json:"characters"`
	MaxTeamSlots int                `json:"maxTeamSlots"`
}

type CharacterSummary struct {
	ID          string         `json:"id"`
	Code        string         `json:"code"`
	Name        string         `json:"name"`
	Rarity      string         `json:"rarity"`
	RarityName  string         `json:"rarityName"`
	RarityColor string         `json:"rarityColor"`
	Stars       int            `json:"stars"`
	Level       int            `json:"level"`
	Locked      bool           `json:"locked"`
	InTeam      bool           `json:"inTeam"`
	TeamSlot    int            `json:"teamSlot,omitempty"`
	NonTradable bool           `json:"nonTradable"`
	Stats       CharacterStats `json:"stats"`
}

type CharacterStats struct {
	HP         float64 `json:"hp"`
	Damage     float64 `json:"damage"`
	Speed      float64 `json:"speed"`
	Collection float64 `json:"collection"`
}

type AscensionPreviewRequest struct {
	MainCharacterId      string `json:"mainCharacterId"`
	DuplicateCharacterId string `json:"duplicateCharacterId"`
	ExtraEssence         int    `json:"extraEssence"`
}

type AscensionPreviewResponse struct {
	MainCharacterId      string  `json:"mainCharacterId"`
	DuplicateCharacterId string  `json:"duplicateCharacterId"`
	TargetStars          int     `json:"targetStars"`
	ChanceBasisPoints    int     `json:"chanceBasisPoints"`
	BaseChanceBasisPoints int    `json:"baseChanceBasisPoints"`
	FailureCount         int     `json:"failureCount"`
	FailureBonusBasisPoints int  `json:"failureBonusBasisPoints"`
	GoldCost             int64   `json:"goldCost"`
	BaseEssenceCost      int     `json:"baseEssenceCost"`
	ExtraEssenceCost     int     `json:"extraEssenceCost"`
	MaxExtraEssence      int     `json:"maxExtraEssence"`
	Affordable           bool    `json:"affordable"`
	DuplicateInTeam      bool    `json:"duplicateInTeam"`
	DuplicateLocked      bool    `json:"duplicateLocked"`
}

type AscensionConfirmRequest struct {
	ClientRequestId      string `json:"clientRequestId"`
	MainCharacterId      string `json:"mainCharacterId"`
	DuplicateCharacterId string `json:"duplicateCharacterId"`
	ExtraEssence         int    `json:"extraEssence"`
}

type AscensionConfirmResponse struct {
	Succeeded       bool   `json:"succeeded"`
	MainCharacterId string `json:"mainCharacterId"`
	TargetStars     int    `json:"targetStars"`
}

type HeroBurnResponse struct {
	Level     int            `json:"level"`
	BatchSize int            `json:"batchSize"` // 10
	Balances  map[string]int `json:"balances"`
	Tiers     []HeroBurnTier `json:"tiers"`
	Heroes    []HeroBurnHero `json:"heroes"`
}

type HeroBurnTier struct {
	FromRarity string `json:"fromRarity"`
	ToRarity   string `json:"toRarity"`
	Unlocked   bool   `json:"unlocked"`
	Charges    int    `json:"charges"`
	MaxCharges int    `json:"maxCharges"`
}

type HeroBurnHero struct {
	CharacterId string `json:"characterId"`
	RarityCode  string `json:"rarityCode"`
	Eligible    bool   `json:"eligible"`
}

// ----------------- ITENS & INVENTÁRIO -----------------

type ItemsResponse struct {
	Items []ItemSummary `json:"items"`
}

type ItemSummary struct {
	ID                    string             `json:"id"`
	Code                  string             `json:"code"`
	Name                  string             `json:"name"`
	Rarity                string             `json:"rarity"`
	Slot                  string             `json:"slot"`
	EquippedHeroId        string             `json:"equippedHeroId,omitempty"`
	EquippedCharacterId   string             `json:"equippedCharacterId,omitempty"`
	RequiredCharacterCode string             `json:"requiredCharacterCode,omitempty"`
	RefineLevel           int                `json:"refineLevel"`
	EnhancementLevel      int                `json:"enhancementLevel,omitempty"`
	Power                 float64            `json:"power,omitempty"`
	Stats                 map[string]float64 `json:"stats"`
	Locked                bool               `json:"locked"`
}

// ----------------- FORJA, POÇO MÍTICO, CLÃ, PRESTÍGIO, INFINITO & PVP -----------------

type EnhancementPreviewResponse struct {
	ItemID           string         `json:"itemId"`
	CurrentLevel     int            `json:"currentLevel"`
	TargetLevel      int            `json:"targetLevel"`
	SuccessRate      float64        `json:"successRate"`
	CostFragments    int            `json:"costFragments"`
	CostGold         int64          `json:"costGold"`
	Affordable       bool           `json:"affordable"`
	CurrentStats     map[string]any `json:"currentStats"`
	TargetStats      map[string]any `json:"targetStats"`
}

type EnhanceItemResponse struct {
	Succeeded        bool `json:"succeeded"`
	EnhancementLevel int  `json:"enhancementLevel"`
}

type ReaffixItemResponse struct {
	Succeeded bool           `json:"succeeded"`
	Stats     map[string]any `json:"stats"`
}

type MythicWellResponse struct {
	Progress           int           `json:"progress"`
	Goal               int           `json:"goal"`
	Claimable          bool          `json:"claimable"`
	TargetDefinitionID string        `json:"targetDefinitionId"`
	EligibleItems      []ItemSummary `json:"eligibleItems"`
}

type MythicWellClaimResponse struct {
	Succeeded bool        `json:"succeeded"`
	Item      ItemSummary `json:"item"`
}

type ClanDetailResponse struct {
	Clan     *ClanInfo         `json:"clan"`
	Missions []ClanMission     `json:"missions"`
	Members  []ClanMember      `json:"members"`
	Arsenal  []ClanArsenalItem `json:"items"`
}

type ClanInfo struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Tag      string `json:"tag"`
	Power    int64  `json:"power"`
	Capacity int    `json:"capacity"`
	Role     string `json:"role"`
}

type ClanMission struct {
	Code      string `json:"code"`
	Progress  int    `json:"progress"`
	Target    int    `json:"target"`
	Completed bool   `json:"completed"`
	Reward    int    `json:"reward"`
}

type ClanMember struct {
	AccountID string `json:"accountId"`
	Name      string `json:"name"`
	Role      string `json:"role"`
}

type ClanArsenalItem struct {
	AccountID string      `json:"accountId"`
	Item      ItemSummary `json:"item"`
}

type PrestigeResponse struct {
	Points    int               `json:"points"`
	MaxPoints int               `json:"maxPoints"`
	Tier      *PrestigeTier     `json:"tier"`
	NextTier  *PrestigeTier     `json:"nextTier"`
	Missions  []PrestigeMission `json:"missions"`
}

type PrestigeTier struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	Level     int    `json:"level"`
	MinPoints int    `json:"minPoints"`
}

type PrestigeMission struct {
	Code      string `json:"code"`
	Type      string `json:"type"`
	Progress  int    `json:"progress"`
	Target    int    `json:"target"`
	Points    int    `json:"points"`
	Completed bool   `json:"completed"`
}

type InfiniteStatusResponse struct {
	Available   bool   `json:"available"`
	HighestWave int    `json:"highestWave"`
	CurrentWave int    `json:"currentWave"`
	Tokens      int    `json:"tokens"`
	NextResetAt string `json:"nextResetAt"`
}

type InfiniteRunResponse struct {
	ID                string `json:"id"`
	CurrentWave       int    `json:"currentWave"`
	NextEventSequence int    `json:"nextEventSequence"`
}

type PvPHubResponse struct {
	RankPoints     int          `json:"rankPoints"`
	Tier           string       `json:"tier"`
	ArenaCoins     int          `json:"arenaCoins"`
	DailyMissions  []PvPMission `json:"daily"`
	WeeklyMissions []PvPMission `json:"weekly"`
}

type PvPMission struct {
	Code      string `json:"code"`
	Progress  int    `json:"progress"`
	Target    int    `json:"target"`
	Completed bool   `json:"completed"`
	Claimed   bool   `json:"claimed"`
	Reward    int    `json:"reward"`
}

type GenericRequest struct {
	ClientRequestId string `json:"clientRequestId"`
}
