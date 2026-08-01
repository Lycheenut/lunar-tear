package userdata

import (
	"sync"

	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/utils"
)

var labyrinthCatalog = sync.OnceValue(masterdata.LoadLabyrinthCatalog)

func init() {
	register("IUserEventQuestLabyrinthSeason", func(user store.UserState) string {
		chapters := labyrinthCatalog().ChaptersByOrder
		records := make([]map[string]any, 0, len(chapters))
		for _, ch := range chapters {
			if st, ok := user.LabyrinthSeasons[ch.EventQuestChapterId]; ok {
				lastJoinSeasonNumber := st.LastJoinSeasonNumber
				if lastJoinSeasonNumber == 0 {
					lastJoinSeasonNumber = ch.LatestSeasonNumber
				}
				records = append(records, map[string]any{
					"userId":                               user.UserId,
					"eventQuestChapterId":                  st.EventQuestChapterId,
					"lastJoinSeasonNumber":                 lastJoinSeasonNumber,
					"lastSeasonRewardReceivedSeasonNumber": st.LastSeasonRewardReceivedSeasonNumber,
					"latestVersion":                        st.LatestVersion,
				})
				continue
			}
			records = append(records, map[string]any{
				"userId":                               user.UserId,
				"eventQuestChapterId":                  ch.EventQuestChapterId,
				"lastJoinSeasonNumber":                 ch.LatestSeasonNumber,
				"lastSeasonRewardReceivedSeasonNumber": 0,
				"latestVersion":                        user.GameStartDatetime,
			})
		}
		s, _ := utils.EncodeJSONMaps(records...)
		return s
	})

	register("IUserEventQuestLabyrinthStage", func(user store.UserState) string {
		records := make([]map[string]any, 0)
		for _, ch := range labyrinthCatalog().ChaptersByOrder {
			for _, stageOrder := range ch.StageOrders {
				key := store.LabyrinthStageKey{
					EventQuestChapterId: ch.EventQuestChapterId,
					StageOrder:          stageOrder,
				}
				if st, ok := user.LabyrinthStages[key]; ok {
					records = append(records, map[string]any{
						"userId":                     user.UserId,
						"eventQuestChapterId":        st.EventQuestChapterId,
						"stageOrder":                 st.StageOrder,
						"isReceivedStageClearReward": st.IsReceivedStageClearReward,
						"accumulationRewardReceivedQuestMissionCount": st.AccumulationRewardReceivedQuestMissionCount,
						"latestVersion": st.LatestVersion,
					})
					continue
				}
				records = append(records, map[string]any{
					"userId":                     user.UserId,
					"eventQuestChapterId":        ch.EventQuestChapterId,
					"stageOrder":                 stageOrder,
					"isReceivedStageClearReward": false,
					"accumulationRewardReceivedQuestMissionCount": 0,
					"latestVersion": user.GameStartDatetime,
				})
			}
		}
		s, _ := utils.EncodeJSONMaps(records...)
		return s
	})
}
