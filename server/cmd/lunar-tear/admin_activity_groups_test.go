package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"lunar-tear/server/internal/activitygroup"
	"lunar-tear/server/internal/gacha"
	"lunar-tear/server/internal/masterdataadmin"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
)

func TestActivityGroupRoutesPersistPreviewAndPublish(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e"))
	if errors.Is(err, os.ErrNotExist) {
		t.Skip("repository master-data asset is not installed")
	}
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	binPath, configPath := filepath.Join(directory, "master.bin.e"), filepath.Join(directory, "gacha.json")
	activityPath := filepath.Join(directory, "config", "activity-groups.json")
	if err := os.WriteFile(binPath, raw, 0600); err != nil {
		t.Fatal(err)
	}
	initialConfig := gacha.DefaultConfig()
	initialConfig.Banners[588] = gacha.BannerConfig{BannerAssetName: "limited_588", StartDatetime: gacha.DefaultBannerStartDatetime, EndDatetime: gacha.DefaultBannerEndDatetime}
	originalConfig, _, err := gacha.EncodeConfig(initialConfig)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, originalConfig, 0600); err != nil {
		t.Fatal(err)
	}
	holder, err := runtime.NewHolderWithConfigs(binPath, configPath, "", activityPath)
	if err != nil {
		t.Fatal(err)
	}
	originalHash := holder.Get().MasterDataHash
	originalEntries := holder.Get().GachaEntries
	if err := initializeActivityGroups(binPath, activityPath, configPath, holder); err != nil {
		t.Fatal(err)
	}
	if holder.Get().MasterDataHash != originalHash {
		t.Fatal("initialization rescheduled master data")
	}
	initialHash := holder.Get().ActivityConfigHash
	initialGachaHash := holder.Get().GachaConfigHash
	if len(holder.Get().GachaConfig.EventSchedules) == 0 {
		t.Fatal("initial Event Gacha schedules missing from Gacha config")
	}
	for _, entry := range originalEntries {
		if entry.GachaLabelType == model.GachaLabelEvent {
			schedule := holder.Get().GachaConfig.EventSchedules[entry.GachaId]
			if schedule.StartDatetime != entry.StartDatetime || schedule.EndDatetime != entry.EndDatetime {
				t.Fatal("initialization changed Event Gacha time")
			}
		}
	}
	originalConfig, err = os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := initializeActivityGroups(binPath, activityPath, configPath, holder); err != nil {
		t.Fatal(err)
	}
	if holder.Get().ActivityConfigHash != initialHash || holder.Get().GachaConfigHash != initialGachaHash {
		t.Fatal("initialization was not idempotent")
	}
	mux := http.NewServeMux()
	registerActivityGroupRoutes(mux, func(r *http.Request) bool { return r.Header.Get("Authorization") == "Bearer test" }, &sync.Mutex{}, binPath, activityPath, configPath, holder)
	request := func(method, route, token string, body interface{}) *httptest.ResponseRecorder {
		var encoded []byte
		if body != nil {
			encoded, _ = json.Marshal(body)
		}
		req := httptest.NewRequest(method, route, bytes.NewReader(encoded))
		req.Header.Set("Authorization", token)
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, req)
		return response
	}
	const base = "/api/admin/activity-groups"
	if response := request("GET", base, "", nil); response.Code != 401 {
		t.Fatal("missing authentication accepted")
	}
	response := request("GET", base, "Bearer test", nil)
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	var loaded struct {
		Catalog masterdataadmin.ActivityGroupCatalog `json:"catalog"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &loaded); err != nil {
		t.Fatal(err)
	}
	if len(loaded.Catalog.Config.Groups) == 0 {
		t.Fatal("initial groups missing from API")
	}
	schedule := activityGroupRequest{ExpectedContentHash: initialHash, ExpectedGachaConfigHash: holder.Get().GachaConfigHash, ExpectedMasterDataHash: originalHash, GroupID: "chapter:508", StartDatetime: 1800000000000, EndDatetime: 1800100000000}
	response = request("POST", base+"/schedule/preview", "Bearer test", schedule)
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	if holder.Get().ActivityConfigHash != initialHash || holder.Get().MasterDataHash != originalHash {
		t.Fatal("preview published changes")
	}
	bad := schedule
	bad.EndDatetime = bad.StartDatetime - 1
	if response = request("POST", base+"/schedule", "Bearer test", bad); response.Code != 400 {
		t.Fatal("invalid date accepted")
	}
	bad = schedule
	bad.ExpectedContentHash = "stale"
	if response = request("POST", base+"/schedule", "Bearer test", bad); response.Code != 409 {
		t.Fatal("stale config accepted")
	}
	bad = schedule
	bad.ExpectedGachaConfigHash = "stale"
	if response = request("POST", base+"/schedule", "Bearer test", bad); response.Code != 409 {
		t.Fatal("stale Gacha config accepted")
	}
	bad = schedule
	bad.ExpectedMasterDataHash = "stale"
	if response = request("POST", base+"/schedule", "Bearer test", bad); response.Code != 409 {
		t.Fatal("stale master data accepted")
	}
	if response = request("POST", base+"/schedule", "Bearer test", schedule); response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	if holder.Get().MasterDataHash == originalHash {
		t.Fatal("schedule did not publish master data")
	}
	if response = request("POST", base+"/schedule", "Bearer test", schedule); response.Code != 409 {
		t.Fatal("repeated stale update accepted")
	}

	// An individual chapter edit must not change the now-persisted Event Gacha.
	snapshot := holder.Get()
	var eventID, chapterID int32
	var eventEnd int64
	for _, entry := range snapshot.GachaEntries {
		if entry.GachaLabelType == model.GachaLabelEvent {
			eventID, chapterID, eventEnd = entry.GachaId, entry.RelatedEventQuestChapterId, entry.EndDatetime
			break
		}
	}
	if eventID == 0 {
		t.Fatal("no Event Gacha fixture")
	}
	catalog, err := masterdataadmin.LoadTable(binPath, "m_event_quest_chapter")
	if err != nil {
		t.Fatal(err)
	}
	rowIndex := -1
	for _, row := range catalog.Tables[0].Rows {
		if row.Values["EventQuestChapterId"] == jsonNumber(chapterID) {
			rowIndex = row.Index
		}
	}
	if rowIndex < 0 {
		t.Fatal("Event Gacha chapter missing")
	}
	candidate, _, err := masterdataadmin.BuildUpdate(binPath, masterdataadmin.UpdateRequest{ExpectedVersion: catalog.Version, Changes: []masterdataadmin.Change{{Table: "m_event_quest_chapter", Row: rowIndex, Field: "EndDatetime", Value: eventEnd + 1000}}})
	if err != nil {
		t.Fatal(err)
	}
	path, err := writeCandidate(binPath, candidate)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)
	if err := holder.InstallAndReload(path); err != nil {
		t.Fatal(err)
	}
	for _, entry := range holder.Get().GachaEntries {
		if entry.GachaId == eventID && entry.EndDatetime != eventEnd {
			t.Fatal("Event Gacha time still follows chapter edits")
		}
	}

	currentConfig, err := os.ReadFile(configPath)
	if err != nil || !bytes.Equal(currentConfig, originalConfig) {
		t.Fatalf("chapter edits rewrote Gacha config: %v", err)
	}
	snapshot = holder.Get()
	eventSchedule := activityGroupRequest{ExpectedContentHash: snapshot.ActivityConfigHash, ExpectedGachaConfigHash: snapshot.GachaConfigHash, ExpectedMasterDataHash: snapshot.MasterDataHash, GroupID: "chapter:" + jsonNumber(chapterID), StartDatetime: 1801000000000, EndDatetime: 1802000000000}
	if response = request("POST", base+"/schedule", "Bearer test", eventSchedule); response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	if holder.Get().GachaConfig.EventSchedules[eventID].EndDatetime != eventSchedule.EndDatetime+48*60*60*1000 {
		t.Fatal("Event Gacha schedule was not saved in Gacha config with 48 hour expiry")
	}
	if holder.Get().ActivityConfigHash != snapshot.ActivityConfigHash {
		t.Fatal("scheduling changed activity membership config")
	}
	snapshot = holder.Get()
	premiumSchedule := activityGroupRequest{ExpectedContentHash: snapshot.ActivityConfigHash, ExpectedGachaConfigHash: snapshot.GachaConfigHash, ExpectedMasterDataHash: snapshot.MasterDataHash, GroupID: "premium:588", StartDatetime: 1800000000000, EndDatetime: 1800100000000}
	if response = request("POST", base+"/schedule", "Bearer test", premiumSchedule); response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	if holder.Get().GachaConfig.Banners[588].EndDatetime != premiumSchedule.EndDatetime {
		t.Fatal("Premium Gacha schedule was not updated")
	}
	currentConfig, err = os.ReadFile(configPath)
	if err != nil || bytes.Contains(currentConfig, []byte(`"activityGroups"`)) || !bytes.Contains(currentConfig, []byte(`"eventSchedules"`)) {
		t.Fatalf("Gacha config does not separate schedules from activity membership: %v", err)
	}
	snapshot = holder.Get()
	empty := &activitygroup.Config{Version: activitygroup.ConfigVersion, Units: []activitygroup.ActivityUnit{}, Groups: []activitygroup.ActivityGroup{}}
	response = request("POST", base, "Bearer test", activityGroupRequest{ExpectedContentHash: snapshot.ActivityConfigHash, ExpectedGachaConfigHash: snapshot.GachaConfigHash, ExpectedMasterDataHash: snapshot.MasterDataHash, Config: empty})
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	if err := holder.Reload(); err != nil {
		t.Fatal(err)
	}
	if err := initializeActivityGroups(binPath, activityPath, configPath, holder); err != nil {
		t.Fatal(err)
	}
	if len(holder.Get().ActivityConfig.Groups) != 0 {
		t.Fatal("saved empty configuration was regenerated after restart")
	}
	persisted, _, err := activitygroup.ReadConfig(activityPath)
	if err != nil || persisted == nil || len(persisted.Groups) != 0 {
		t.Fatalf("independent activity config was not saved: %v", err)
	}
	afterMembership, err := os.ReadFile(configPath)
	if err != nil || !bytes.Equal(afterMembership, currentConfig) {
		t.Fatalf("membership save rewrote Gacha config: %v", err)
	}
	activityRaw, err := os.ReadFile(activityPath)
	if err != nil || bytes.Contains(activityRaw, []byte(`"eventSchedules"`)) {
		t.Fatalf("activity config still stores Event Gacha schedules: %v", err)
	}
	withoutGroups, err := runtime.NewHolderWithGachaConfig(binPath, configPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range withoutGroups.Get().GachaEntries {
		if entry.GachaId == eventID && (entry.StartDatetime != eventSchedule.StartDatetime || entry.EndDatetime != eventSchedule.EndDatetime+48*60*60*1000) {
			t.Fatal("Event Gacha schedule depends on loading the activity config")
		}
	}
}

func jsonNumber(value int32) string { raw, _ := json.Marshal(value); return string(raw) }
