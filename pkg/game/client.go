package game

import (
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"bot-template-go/pkg/network"
)

func newUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

const (
	APIBaseURL = "https://api.farmricochet.com"
	OriginURL  = "https://farmricochet.com"
	PlayURL    = "https://farmricochet.com/play"
)

// FarmClient comunica-se com a API oficial do Farm Ricochet
type FarmClient struct {
	httpCli *network.HTTPClient
	cookie  string
}

// NewFarmClient instancia o cliente com headers modernos e cookies de sessão
func NewFarmClient(cookie, token string) *FarmClient {
	cli := network.NewHTTPClient(APIBaseURL, 20*time.Second)

	// Injeta headers essenciais de navegador
	cli.SetHeader("Origin", OriginURL)
	cli.SetHeader("Referer", PlayURL)
	cli.SetHeader("Accept", "application/json, text/plain, */*")
	cli.SetHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")

	if cookie != "" {
		cli.SetCookie(cookie)
	}

	return &FarmClient{
		httpCli: cli,
		cookie:  cookie,
	}
}

// ----------------- SESSÃO & ADMISSÃO -----------------

func (c *FarmClient) AcquireSessionLease(holderId string, force bool) (*SessionLeaseResponse, error) {
	req := SessionLeaseRequest{HolderId: holderId, Force: force}
	var resp SessionLeaseResponse
	err := c.httpCli.PostJSON("/api/v1/game-session", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *FarmClient) HeartbeatSessionLease(holderId string) (*SessionLeaseResponse, error) {
	req := SessionLeaseRequest{HolderId: holderId}
	var resp SessionLeaseResponse
	err := c.httpCli.PutJSON("/api/v1/game-session", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *FarmClient) ReleaseSessionLease(holderId string) error {
	endpoint := fmt.Sprintf("/api/v1/game-session/%s", url.PathEscape(holderId))
	return c.httpCli.DeleteJSON(endpoint, nil, nil)
}

func (c *FarmClient) CheckAdmission() (*GameAdmissionResponse, error) {
	var resp GameAdmissionResponse
	err := c.httpCli.GetJSON("/api/v1/game-admission", &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// ----------------- BOOTSTRAP & CONTA -----------------

func (c *FarmClient) GetBootstrap() (*BootstrapResponse, error) {
	var resp BootstrapResponse
	err := c.httpCli.GetJSON("/api/v1/bootstrap?mapImages=cdn", &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *FarmClient) GetAccount() (*AccountData, error) {
	var resp AccountData
	err := c.httpCli.GetJSON("/api/v1/account", &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *FarmClient) SetActiveLevel(code string) error {
	payload := map[string]string{"code": code}
	return c.httpCli.PutJSON("/api/v1/progress/active-level", payload, nil)
}

// ----------------- RUNS / CAMPANHA -----------------

func (c *FarmClient) StartRun(req RunStartRequest) (*RunStartResponse, error) {
	var resp RunStartResponse
	err := c.httpCli.PostJSON("/api/v1/runs/start", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *FarmClient) HeartbeatRun(runId string, envelope RunEnvelope) (*RunHeartbeatResponse, error) {
	endpoint := fmt.Sprintf("/api/v1/runs/%s/heartbeat", url.PathEscape(runId))
	var resp RunHeartbeatResponse
	status, body, err := c.httpCli.DoRequest(http.MethodPut, endpoint, envelope, &resp)
	if err != nil {
		return nil, err
	}
	log.Printf("[HEARTBEAT-RAW] HTTP %d: %s", status, string(body))
	return &resp, nil
}

func (c *FarmClient) CompleteRun(runId string, envelope RunEnvelope) (*RunCompleteResponse, error) {
	endpoint := fmt.Sprintf("/api/v1/runs/%s/complete", url.PathEscape(runId))
	var resp RunCompleteResponse
	status, body, err := c.httpCli.DoRequest(http.MethodPost, endpoint, envelope, &resp)
	if err != nil {
		return nil, err
	}
	log.Printf("[COMPLETE-RUN-RAW] HTTP %d: %s", status, string(body))
	if status == 429 {
		return nil, fmt.Errorf("rate_limit: pausa solicitada pelo servidor (HTTP 429)")
	}
	if status != 200 && status != 202 {
		return nil, fmt.Errorf("conclusão recusada (HTTP %d): %s", status, string(body))
	}
	return &resp, nil
}

func (c *FarmClient) DefeatLootGoblin(runId string) error {
	endpoint := fmt.Sprintf("/api/v1/runs/%s/loot-goblin/defeat", url.PathEscape(runId))
	return c.httpCli.PostJSON(endpoint, map[string]any{}, nil)
}

// ----------------- ESPÓLIOS (RUN LOOT) -----------------

func (c *FarmClient) GetRunLoot() (*RunLootResponse, error) {
	var resp RunLootResponse
	err := c.httpCli.GetJSON("/api/v1/run-loot", &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *FarmClient) ClaimRunLoot(idempotencyKey string) error {
	req := RunLootClaimRequest{IdempotencyKey: idempotencyKey}
	return c.httpCli.PostJSON("/api/v1/run-loot/claim", req, nil)
}

// ----------------- BAÚS (CHESTS) -----------------

func (c *FarmClient) GetChests() ([]ChestItem, error) {
	var resp ChestsResponse
	err := c.httpCli.GetJSON("/api/v1/chests", &resp)
	if err != nil {
		return nil, err
	}
	return resp.Chests, nil
}

func (c *FarmClient) OpenChest(code string) (*ChestOpenResponse, error) {
	endpoint := fmt.Sprintf("/api/v1/chests/%s/open", url.PathEscape(code))
	req := ChestOpenRequest{ClientRequestId: newUUID()}
	var resp ChestOpenResponse
	err := c.httpCli.PostJSON(endpoint, req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *FarmClient) OpenStarterChest() error {
	req := map[string]string{"clientRequestId": newUUID()}
	return c.httpCli.PostJSON("/api/v1/onboarding/starter-chest/open", req, nil)
}

// ----------------- MISSÕES & CONQUISTAS -----------------

func (c *FarmClient) GetDailyMissions() (*DailyMissionsResponse, error) {
	var resp DailyMissionsResponse
	err := c.httpCli.GetJSON("/api/v1/missions/daily", &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *FarmClient) ClaimDailyMission(code string) error {
	endpoint := fmt.Sprintf("/api/v1/missions/daily/%s/claim", url.PathEscape(code))
	return c.httpCli.PostJSON(endpoint, map[string]any{}, nil)
}

func (c *FarmClient) GetWeeklyMissions() (*WeeklyMissionsResponse, error) {
	var resp WeeklyMissionsResponse
	err := c.httpCli.GetJSON("/api/v1/missions/weekly", &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *FarmClient) DeliverWeeklyMission(missionId string, quantity int) error {
	endpoint := fmt.Sprintf("/api/v1/missions/weekly/%s/deliver", url.PathEscape(missionId))
	req := map[string]any{
		"clientRequestId": newUUID(),
		"quantity":        quantity,
	}
	return c.httpCli.PostJSON(endpoint, req, nil)
}

func (c *FarmClient) ClaimWeeklyReward(rewardId string) error {
	endpoint := fmt.Sprintf("/api/v1/missions/weekly/rewards/%s/claim", url.PathEscape(rewardId))
	req := map[string]any{"clientRequestId": newUUID()}
	return c.httpCli.PostJSON(endpoint, req, nil)
}

func (c *FarmClient) GetMilestones() (*MilestonesResponse, error) {
	var resp MilestonesResponse
	err := c.httpCli.GetJSON("/api/v1/progression/milestones", &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *FarmClient) ClaimMilestone(milestoneId string) error {
	endpoint := fmt.Sprintf("/api/v1/progression/milestones/%s/claim", url.PathEscape(milestoneId))
	req := map[string]any{"clientRequestId": newUUID()}
	return c.httpCli.PostJSON(endpoint, req, nil)
}

func (c *FarmClient) GetBeginnerMissions() (*BeginnerMissionsResponse, error) {
	var resp BeginnerMissionsResponse
	err := c.httpCli.GetJSON("/api/v1/missions/beginner", &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *FarmClient) ClaimBeginnerMission(code string, choice string) error {
	endpoint := fmt.Sprintf("/api/v1/missions/beginner/%s/claim", url.PathEscape(code))
	req := map[string]any{
		"clientRequestId": newUUID(),
		"choice":          choice,
	}
	return c.httpCli.PostJSON(endpoint, req, nil)
}

func (c *FarmClient) GetBestiary() ([]BestiaryEntry, error) {
	var list []BestiaryEntry
	err := c.httpCli.GetJSON("/api/v1/bestiary", &list)
	if err != nil {
		return nil, err
	}
	return list, nil
}

func (c *FarmClient) ClaimBestiary(creatureCode string) error {
	endpoint := fmt.Sprintf("/api/v1/bestiary/%s/claim", url.PathEscape(creatureCode))
	req := map[string]any{
		"clientRequestId": newUUID(),
		"stage":           3,
	}
	return c.httpCli.PostJSON(endpoint, req, nil)
}

// ----------------- PASSE DE BATALHA -----------------

func (c *FarmClient) GetBattlePass() (*BattlePassResponse, error) {
	var resp BattlePassResponse
	err := c.httpCli.GetJSON("/api/v1/battle-pass", &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *FarmClient) ClaimBattlePassMissions(claims []any) error {
	if len(claims) == 0 {
		return nil
	}
	req := map[string]any{"claims": claims}
	return c.httpCli.PostJSON("/api/v1/battle-pass/missions/claim-batch", req, nil)
}

func (c *FarmClient) ClaimBattlePassLevels(claims []any) error {
	if len(claims) == 0 {
		return nil
	}
	req := map[string]any{"claims": claims}
	return c.httpCli.PostJSON("/api/v1/battle-pass/levels/claim-batch", req, nil)
}

// ----------------- TALENTOS -----------------

func (c *FarmClient) GetTalentPreview(code string) (*TalentPreview, error) {
	endpoint := fmt.Sprintf("/api/v1/farm-talents/%s/preview", url.PathEscape(code))
	var resp TalentPreview
	err := c.httpCli.GetJSON(endpoint, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *FarmClient) PurchaseTalent(code string, key string) (*TalentPurchaseResponse, error) {
	endpoint := fmt.Sprintf("/api/v1/farm-talents/%s/purchase", url.PathEscape(code))
	req := TalentPurchaseRequest{IdempotencyKey: key}
	var resp TalentPurchaseResponse
	err := c.httpCli.PostJSON(endpoint, req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// ----------------- PERSONAGENS & HERÓIS -----------------

func (c *FarmClient) GetCharacters() ([]CharacterSummary, int, error) {
	var resp CharactersResponse
	err := c.httpCli.GetJSON("/api/v1/characters", &resp)
	if err != nil {
		return nil, 0, err
	}
	return resp.Characters, resp.MaxTeamSlots, nil
}

func (c *FarmClient) SaveTeam(characterIds []string) error {
	payload := map[string]any{"characterIds": characterIds}
	return c.httpCli.PutJSON("/api/v1/compositions/active", payload, nil)
}

func (c *FarmClient) PreviewAscension(mainId, duplicateId string, extraEssence int) (*AscensionPreviewResponse, error) {
	req := AscensionPreviewRequest{
		MainCharacterId:      mainId,
		DuplicateCharacterId: duplicateId,
		ExtraEssence:         extraEssence,
	}
	var resp AscensionPreviewResponse
	err := c.httpCli.PostJSON("/api/v1/ascensions/preview", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *FarmClient) ConfirmAscension(mainId, duplicateId string, extraEssence int) (*AscensionConfirmResponse, error) {
	req := AscensionConfirmRequest{
		ClientRequestId:      newUUID(),
		MainCharacterId:      mainId,
		DuplicateCharacterId: duplicateId,
		ExtraEssence:         extraEssence,
	}
	var resp AscensionConfirmResponse
	err := c.httpCli.PostJSON("/api/v1/ascensions/confirm", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *FarmClient) GetHeroBurn() (*HeroBurnResponse, error) {
	var resp HeroBurnResponse
	err := c.httpCli.GetJSON("/api/v1/hero-burn", &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *FarmClient) ExecuteHeroBurn(characterIds []string, boosts map[string]int) error {
	req := map[string]any{
		"clientRequestId": newUUID(),
		"characterIds":    characterIds,
		"boosts":          boosts,
	}
	return c.httpCli.PostJSON("/api/v1/hero-burn", req, nil)
}

func (c *FarmClient) RepairHeroBurnAltar(level int) error {
	req := map[string]any{
		"clientRequestId": newUUID(),
		"level":           level,
	}
	return c.httpCli.PostJSON("/api/v1/hero-burn/repair", req, nil)
}

func (c *FarmClient) SacrificeHeroesLegacy(characterIds []string) error {
	req := map[string]any{
		"clientRequestId": newUUID(),
		"characterIds":    characterIds,
	}
	return c.httpCli.PostJSON("/api/v1/characters/legacy", req, nil)
}

// ----------------- ITENS & INVENTÁRIO -----------------

func (c *FarmClient) GetItems() ([]ItemSummary, error) {
	var resp ItemsResponse
	err := c.httpCli.GetJSON("/api/v1/items", &resp)
	if err != nil {
		return nil, err
	}
	return resp.Items, nil
}

func (c *FarmClient) DismantleItems(itemIds []string) error {
	req := map[string]any{
		"clientRequestId": newUUID(),
		"itemIds":         itemIds,
	}
	return c.httpCli.PostJSON("/api/v1/items/dismantle", req, nil)
}

func (c *FarmClient) FuseCrystals(tier string) error {
	req := map[string]any{
		"clientRequestId": newUUID(),
		"tier":            tier,
	}
	return c.httpCli.PostJSON("/api/v1/items/crystals/fuse", req, nil)
}

func (c *FarmClient) EquipItem(characterId, slot, itemId string) error {
	endpoint := fmt.Sprintf("/api/v1/characters/%s/equipment/%s", url.PathEscape(characterId), url.PathEscape(slot))
	req := map[string]string{"itemId": itemId}
	return c.httpCli.PutJSON(endpoint, req, nil)
}

func (c *FarmClient) UnequipItem(characterId, slot string) error {
	endpoint := fmt.Sprintf("/api/v1/characters/%s/equipment/%s", url.PathEscape(characterId), url.PathEscape(slot))
	return c.httpCli.DeleteJSON(endpoint, nil, nil)
}

func (c *FarmClient) GetEnhancementPreview(itemId string) (*EnhancementPreviewResponse, error) {
	endpoint := fmt.Sprintf("/api/v1/items/%s/enhancement-preview", url.PathEscape(itemId))
	var resp EnhancementPreviewResponse
	err := c.httpCli.GetJSON(endpoint, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *FarmClient) EnhanceItem(itemId string, useForgeMiracle bool) (*EnhanceItemResponse, error) {
	endpoint := fmt.Sprintf("/api/v1/items/%s/enhance", url.PathEscape(itemId))
	req := map[string]any{
		"clientRequestId": newUUID(),
		"useForgeMiracle": useForgeMiracle,
	}
	var resp EnhanceItemResponse
	err := c.httpCli.PostJSON(endpoint, req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *FarmClient) ReaffixItem(itemId string) (*ReaffixItemResponse, error) {
	endpoint := fmt.Sprintf("/api/v1/items/%s/reaffix", url.PathEscape(itemId))
	req := map[string]any{"clientRequestId": newUUID()}
	var resp ReaffixItemResponse
	err := c.httpCli.PostJSON(endpoint, req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *FarmClient) GetMythicWell() (*MythicWellResponse, error) {
	var resp MythicWellResponse
	err := c.httpCli.GetJSON("/api/v1/items/mythic-well", &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *FarmClient) SetMythicWellTarget(itemDefinitionId string) error {
	req := map[string]any{
		"clientRequestId":  newUUID(),
		"itemDefinitionId": itemDefinitionId,
	}
	return c.httpCli.PutJSON("/api/v1/items/mythic-well/target", req, nil)
}

func (c *FarmClient) DepositMythicWell(itemIds []string) error {
	req := map[string]any{
		"clientRequestId": newUUID(),
		"itemIds":         itemIds,
	}
	return c.httpCli.PostJSON("/api/v1/items/mythic-well/deposit", req, nil)
}

func (c *FarmClient) ClaimMythicWell() (*MythicWellClaimResponse, error) {
	req := map[string]any{"clientRequestId": newUUID()}
	var resp MythicWellClaimResponse
	err := c.httpCli.PostJSON("/api/v1/items/mythic-well/claim", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// ----------------- CLÃ (GUILD) -----------------

func (c *FarmClient) GetMyClan() (*ClanDetailResponse, error) {
	var resp ClanDetailResponse
	err := c.httpCli.GetJSON("/api/v1/clans/me", &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *FarmClient) DonateClanMission(clanId, missionCode string, quantity int) error {
	endpoint := fmt.Sprintf("/api/v1/clans/%s/missions/%s/donate", url.PathEscape(clanId), url.PathEscape(missionCode))
	req := map[string]any{
		"clientRequestId": newUUID(),
		"quantity":        quantity,
	}
	return c.httpCli.PostJSON(endpoint, req, nil)
}

func (c *FarmClient) AllocateClanArsenal(clanId, itemId string) error {
	endpoint := fmt.Sprintf("/api/v1/clans/%s/arsenal/allocate", url.PathEscape(clanId))
	req := map[string]any{
		"clientRequestId": newUUID(),
		"itemId":          itemId,
	}
	return c.httpCli.PostJSON(endpoint, req, nil)
}

func (c *FarmClient) ClaimClanRewards(clanId string) error {
	endpoint := fmt.Sprintf("/api/v1/clans/%s/rewards/claim", url.PathEscape(clanId))
	req := map[string]any{"clientRequestId": newUUID()}
	return c.httpCli.PostJSON(endpoint, req, nil)
}

// ----------------- PRESTÍGIO -----------------

func (c *FarmClient) GetPrestige() (*PrestigeResponse, error) {
	var resp PrestigeResponse
	err := c.httpCli.GetJSON("/api/v1/prestige/me", &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// ----------------- DESAFIO INFINITO (TORRE) -----------------

func (c *FarmClient) GetInfiniteStatus() (*InfiniteStatusResponse, error) {
	var resp InfiniteStatusResponse
	err := c.httpCli.GetJSON("/api/v1/infinite", &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *FarmClient) StartInfiniteRun(holderId string) (*InfiniteRunResponse, error) {
	req := map[string]string{
		"clientRunId": newUUID(),
		"holderId":    holderId,
	}
	var resp InfiniteRunResponse
	err := c.httpCli.PostJSON("/api/v1/infinite/runs", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *FarmClient) SendInfiniteEvent(runId, eventType string, wave, sequence, qty int) error {
	endpoint := fmt.Sprintf("/api/v1/infinite/runs/%s/events", url.PathEscape(runId))
	req := map[string]any{
		"eventId":          newUUID(),
		"type":             eventType,
		"wave":             wave,
		"sequence":         sequence,
		"quantity":         qty,
		"activeDurationMS": 0,
		"payload": map[string]string{
			"source": "phaser-combat",
		},
	}
	return c.httpCli.PostJSON(endpoint, req, nil)
}

func (c *FarmClient) EndInfiniteRun(runId, reason string) error {
	endpoint := fmt.Sprintf("/api/v1/infinite/runs/%s/end", url.PathEscape(runId))
	req := map[string]string{"reason": reason}
	return c.httpCli.PostJSON(endpoint, req, nil)
}

// ----------------- ARENA PVP -----------------

func (c *FarmClient) GetPvPHub() (*PvPHubResponse, error) {
	var resp PvPHubResponse
	err := c.httpCli.GetJSON("/api/v1/pvp/hub", &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *FarmClient) ClaimPvPMission(cadence, code string) error {
	req := map[string]string{
		"cadence":   cadence,
		"code":      code,
		"requestId": newUUID(),
	}
	return c.httpCli.PostJSON("/api/v1/pvp/missions/claim", req, nil)
}

func (c *FarmClient) SavePvPFormation(characterIds []string) error {
	req := map[string]any{"characterIds": characterIds}
	return c.httpCli.PostJSON("/api/v1/pvp/formation", req, nil)
}

// ----------------- WEBSOCKET STREAM -----------------

func (c *FarmClient) CreateWebSocketClient(onMessage network.MessageHandler, onDisconnect network.DisconnectHandler) *network.SocketClient {
	wsURL := "wss://api.farmricochet.com/api/v1/game/stream"
	headers := make(http.Header)
	headers.Set("Origin", OriginURL)
	headers.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
	if c.cookie != "" {
		headers.Set("Cookie", c.cookie)
	}
	return network.NewSocketClient(wsURL, headers, onMessage, onDisconnect)
}
