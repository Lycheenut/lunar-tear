package userdata

import (
	"sort"

	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/utils"
)

func init() {
	register("IUserWebviewPanelMission", func(user store.UserState) string {
		records := make([]map[string]any, 0, len(user.WebviewPanelMissions))
		ids := make([]int, 0, len(user.WebviewPanelMissions))
		for id := range user.WebviewPanelMissions {
			ids = append(ids, int(id))
		}
		sort.Ints(ids)
		for _, id := range ids {
			state := user.WebviewPanelMissions[int32(id)]
			records = append(records, map[string]any{
				"userId":                    user.UserId,
				"webviewPanelMissionPageId": state.WebviewPanelMissionPageId,
				"rewardReceiveDatetime":     state.RewardReceiveDatetime,
				"latestVersion":             state.LatestVersion,
			})
		}
		s, _ := utils.EncodeJSONMaps(records...)
		return s
	})
}
