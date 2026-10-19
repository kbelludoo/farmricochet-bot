# 🌾 Farm Ricochet — Bot Autônomo & Dashboard em Tempo Real (Go Native Engine)

Bot autônomo completo de altíssima performance desenvolvido em **Go (Golang)** para o jogo [Farm Ricochet](https://farmricochet.com/play). Projetado com arquitetura nativa em Goroutines, baixo consumo de memória (< 2 MB de RAM), compatibilidade com relógio canônico do servidor e proteção contra detecções via **Circuit Breaker Anti-Cheat**.

🌐 **Dashboard ao Vivo (GitHub Pages):** [https://kbelludoo.github.io/farmricochet-bot/](https://kbelludoo.github.io/farmricochet-bot/)  
⚡ **Central de Bots (VPS Hub):** [https://kbelludoo.github.io/bot-template-go/?v=force1](https://kbelludoo.github.io/bot-template-go/?v=force1)

---

## 🌟 Funcionalidades do Bot (Cobertura Total do Jogo)

1. 🚜 **Auto-Run da Campanha & Ritmo 1.25x:**
   - Inicia e avança fases da campanha automaticamente (`/api/v1/runs/start`).
   - Simulação canônica com geração de `structureEvents` e heartbeats legítimos a cada 6 segundos.
   - Sincronização inteligente com o relógio autoritativo do servidor (`accepted: false` com retry automático).
   - Coleta instantânea de espólios de vitória (`/api/v1/runs/loot/claim`).

2. 👥 **Gerenciamento e Preenchimento Automático de Equipe:**
   - Detecta vagas livres na equipe (até 3 vagas padrão, escalável até 5 nos Talentos).
   - Auto-equipa os melhores heróis disponíveis com priorização estratégica:
     - **Ladino (`farmer`):** Prioridade máxima (+100% Ouro por abate).
     - **Arqueira (`runner`):** DPS à distância e velocidade de abate.
     - **Sacerdote (`priest`):** Suporte, cura e buff cumulativo de dano permanente.
   - Salva a composição no servidor com `PUT /api/v1/compositions/active`.

3. 🎁 **Abertura Inteligente de Baús (`AutoChests`):**
   - Baú do Recruta do Onboarding (`/api/v1/onboarding/starter-chest/open`).
   - Baús gratuitos e créditos acumulados.
   - Baús de Herói (`hero-chest`) comprados automaticamente assim que houver gemas suficientes para preencher a equipe.

4. 🎯 **Resgate de Missões & Conquistas:**
   - Missões de Iniciante / Fazendeiro (`/api/v1/missions/beginner`).
   - Missões Diárias (`/api/v1/missions/daily`).
   - Missões Semanais (`/api/v1/missions/weekly`).
   - Marcos / Conquistas (`/api/v1/progression/milestones`).
   - Recompensas do Bestiário (`/api/v1/bestiary`).

5. 🎟️ **Passe de Batalha Automático:**
   - Resgate em lote de missões concluídas do Passe.
   - Reivindicação de níveis liberados da trilha gratuita.

6. 🌱 **Evolução da Árvore de Talentos:**
   - Compra automática de melhorias acessíveis (Colheita de Trigo, Extração de Madeira, Mineração, Fortuna, Atributos de Heróis e Vagas 4 e 5).

7. 🔨 **Forja, Desmanche & Fusão:**
   - Desmanche automático de equipamentos comuns repetidos (`/api/v1/items/dismantle`) para gerar fragmentos de forja.
   - Fusão de cristais de ascensão em tiers superiores (`common` -> `rare` -> `epic`).
   - Ascensão automática de heróis com cópias duplicadas (`ConfirmAscension`).

8. 👾 **Loot Goblin:**
   - Detecção e abate automático do Loot Goblin durante as partidas.

9. 🛡️ **Circuit Breaker Anti-Ban Integrado (`pkg/engine`):**
   - Detecção imediata de Rate Limits (429), WAF (403), conflitos de sessão (`outraAba`) e restrições.
   - Pausa e recuo para zona segura com recuperação em Half-Open após cooldown de 15 minutos.

---

## 📁 Estrutura de Arquivos

```text
farmbot go/
├── accounts.example.json   # Modelo de configuração (ignorado pelo Git)
├── accounts.json           # Suas credenciais reais (NUNCA SUBIR)
├── AGENTS.md               # Diretrizes globais de anti-ban e engenharia
├── main.go                 # Ponto de entrada universal
├── go.mod / go.sum         # Módulo Go nativo
│
├── pkg/
│   ├── game/               # 🎮 Lógica do Farm Ricochet
│   │   ├── types.go        # Structs completas da API, modelos e payloads
│   │   ├── client.go       # Cliente HTTP com lease de sessão e WebSocket
│   │   └── bot.go          # Implementação autônoma do bot
│   ├── engine/             # Circuit Breaker, supervisor e gerenciador
│   ├── network/            # WebSocket thread-safe
│   └── server/             # Web dashboard e endpoints /api/status
│
├── docs/                   # 🌐 GITHUB PAGES (DASHBOARD AO VIVO)
│   ├── index.html          # Dashboard Cyberpunk Farm Ricochet
│   └── status.json         # Telemetria em tempo real
│
└── scripts/
    └── bot.service         # Template systemd para VPS Linux
```

---

## 🚀 Como Executar

### Localmente (Windows / Linux):
```bash
go run .
```
O console exibirá o ticker a cada 4 segundos com memória RAM (< 2 MB), goroutines e status da fazenda, com dashboard em `http://localhost:3013`.

### Compilar para VPS Linux (Ubuntu x86_64):
```cmd
cmd /c "set GOOS=linux&& set GOARCH=amd64&& go build -ldflags=\"-s -w\" -o farmbot-linux ."
```

---

## 🔒 Segurança e Privacidade

- **Zero Credenciais no Git:** `accounts.json`, cookies de sessão, chaves `.key` e `.pem` são estritamente mantidos fora do controle de versão.
- **Telemetria 100% Anônima:** Dashboards públicos exibem exclusivamente identificadores genéricos (`Fazendeiro #1`), classe e métricas de desempenho.
