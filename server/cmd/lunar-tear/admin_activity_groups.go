package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"sync"

	"lunar-tear/server/internal/activitygroup"
	"lunar-tear/server/internal/gacha"
	"lunar-tear/server/internal/masterdataadmin"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
)

func initializeActivityGroups(binPath, configPath, gachaConfigPath string, holder *runtime.Holder) error {
	snapshot := holder.Get()
	if snapshot.ActivityConfig != nil && snapshot.ActivityConfig.Version == activitygroup.ConfigVersion && snapshot.GachaConfig.EventSchedules != nil {
		return nil
	}
	config, err := masterdataadmin.GenerateActivityGroups(binPath, snapshot.ActivityConfig, snapshot.GachaConfig, snapshot.GachaEntries)
	if err != nil {
		return err
	}
	var gachaConfig *gacha.Config
	if snapshot.GachaConfig.EventSchedules == nil {
		copy := *snapshot.GachaConfig
		copy.EventSchedules = make(map[int32]gacha.EventSchedule)
		for _, entry := range snapshot.GachaEntries {
			if entry.GachaLabelType == model.GachaLabelEvent {
				copy.EventSchedules[entry.GachaId] = gacha.EventSchedule{StartDatetime: entry.StartDatetime, EndDatetime: entry.EndDatetime}
			}
		}
		gachaConfig = &copy
	}
	return installActivityConfig(binPath, configPath, gachaConfigPath, holder, config, gachaConfig, nil, snapshot.ActivityConfigHash, snapshot.GachaConfigHash, snapshot.MasterDataHash)
}

func installActivityConfig(binPath, configPath, gachaConfigPath string, holder *runtime.Holder, config *activitygroup.Config, gachaConfig *gacha.Config, masterRaw []byte, expectedConfig, expectedGacha, expectedMaster string) error {
	masterCandidate := ""
	if len(masterRaw) > 0 {
		var err error
		masterCandidate, err = writeCandidate(binPath, masterRaw)
		if err != nil {
			return err
		}
		defer os.Remove(masterCandidate)
	}
	gachaCandidate := ""
	if gachaConfig != nil && !reflect.DeepEqual(gachaConfig, holder.Get().GachaConfig) {
		gachaConfig.SourceMasterDataHash = expectedMaster
		if len(masterRaw) > 0 {
			gachaConfig.SourceMasterDataHash = gacha.ContentHash(masterRaw)
		}
		raw, _, err := gacha.EncodeConfig(gachaConfig)
		if err != nil {
			return err
		}
		gachaCandidate, err = writeConfigCandidate(gachaConfigPath, raw)
		if err != nil {
			return err
		}
		defer os.Remove(gachaCandidate)
	}
	raw, err := activitygroup.EncodeConfig(config)
	if err != nil {
		return err
	}
	candidate, err := writeConfigCandidate(configPath, raw)
	if err != nil {
		return err
	}
	defer os.Remove(candidate)
	return holder.InstallActivityConfig(candidate, gachaCandidate, masterCandidate, expectedConfig, expectedGacha, expectedMaster)
}

type activityGroupRequest struct {
	ExpectedContentHash     string                `json:"expectedContentHash"`
	ExpectedGachaConfigHash string                `json:"expectedGachaConfigHash"`
	ExpectedMasterDataHash  string                `json:"expectedMasterDataHash"`
	Config                  *activitygroup.Config `json:"config,omitempty"`
	GroupID                 string                `json:"groupId,omitempty"`
	StartDatetime           int64                 `json:"startDatetime,omitempty"`
	EndDatetime             int64                 `json:"endDatetime,omitempty"`
}

func registerActivityGroupRoutes(mux *http.ServeMux, authorized func(*http.Request) bool, updateMu *sync.Mutex, binPath, configPath, gachaConfigPath string, holder *runtime.Holder) {
	const base = "/api/admin/activity-groups"
	handler := func(w http.ResponseWriter, r *http.Request) {
		if !authorized(r) {
			writeAdminError(w, http.StatusUnauthorized, "unauthorized")
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		if r.Method != http.MethodPost && !(r.Method == http.MethodGet && r.URL.Path == base) {
			w.Header().Set("Allow", "GET, POST")
			writeAdminError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		updateMu.Lock()
		defer updateMu.Unlock()
		fail := func(err error) {
			status := http.StatusBadRequest
			if errors.Is(err, runtime.ErrActivityConfigConflict) || errors.Is(err, runtime.ErrGachaConfigConflict) || errors.Is(err, runtime.ErrMasterDataConflict) || errors.Is(err, masterdataadmin.ErrVersionConflict) {
				status = http.StatusConflict
			}
			writeAdminError(w, status, err.Error())
		}
		snapshot := holder.Get()
		catalog, err := masterdataadmin.LoadActivityGroups(binPath, snapshot.ActivityConfig, snapshot.GachaConfig, snapshot.GachaEntries)
		if err != nil {
			fail(err)
			return
		}
		if "sha256:"+catalog.Version != snapshot.MasterDataHash {
			fail(runtime.ErrMasterDataConflict)
			return
		}
		if r.Method == http.MethodGet {
			writeAdminJSON(w, http.StatusOK, map[string]interface{}{"catalog": catalog, "contentHash": snapshot.ActivityConfigHash, "gachaConfigHash": snapshot.GachaConfigHash, "masterDataHash": snapshot.MasterDataHash})
			return
		}
		var request activityGroupRequest
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<20))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&request); err != nil {
			fail(fmt.Errorf("请求格式无效: %w", err))
			return
		}
		if request.ExpectedContentHash == "" || request.ExpectedContentHash != snapshot.ActivityConfigHash {
			fail(runtime.ErrActivityConfigConflict)
			return
		}
		if request.ExpectedGachaConfigHash == "" || request.ExpectedGachaConfigHash != snapshot.GachaConfigHash {
			fail(runtime.ErrGachaConfigConflict)
			return
		}
		if request.ExpectedMasterDataHash == "" || request.ExpectedMasterDataHash != snapshot.MasterDataHash {
			fail(runtime.ErrMasterDataConflict)
			return
		}
		config := *snapshot.ActivityConfig
		var gachaConfig *gacha.Config
		var masterRaw []byte
		if r.URL.Path == base {
			if err := masterdataadmin.ValidateActivityGroups(request.Config, catalog); err != nil {
				fail(err)
				return
			}
			config.Units = request.Config.Units
			config.Groups = request.Config.Groups
		} else {
			if request.Config != nil {
				fail(fmt.Errorf("请先保存活动组配置，再修改整体时间"))
				return
			}
			var preview []masterdataadmin.ActivityScheduleChange
			masterRaw, gachaConfig, preview, err = masterdataadmin.BuildActivitySchedule(binPath, &config, snapshot.GachaConfig, catalog, request.GroupID, request.StartDatetime, request.EndDatetime)
			if err != nil {
				fail(err)
				return
			}
			if r.URL.Path == base+"/schedule/preview" {
				writeAdminJSON(w, http.StatusOK, map[string]interface{}{"changes": preview})
				return
			}
		}
		if err := installActivityConfig(binPath, configPath, gachaConfigPath, holder, &config, gachaConfig, masterRaw, request.ExpectedContentHash, request.ExpectedGachaConfigHash, request.ExpectedMasterDataHash); err != nil {
			fail(err)
			return
		}
		writeAdminJSON(w, http.StatusOK, map[string]bool{"ok": true})
	}
	for _, route := range []string{base, base + "/schedule", base + "/schedule/preview"} {
		mux.HandleFunc(route, handler)
	}
}
