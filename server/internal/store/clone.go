package store

import "maps"

func CloneUserState(u UserState) UserState {
	out := u
	out.Friends = maps.Clone(u.Friends)
	out.FriendRequests = maps.Clone(u.FriendRequests)
	out.Tutorials = maps.Clone(u.Tutorials)
	out.Characters = maps.Clone(u.Characters)
	out.Costumes = maps.Clone(u.Costumes)
	out.Weapons = maps.Clone(u.Weapons)
	out.Companions = maps.Clone(u.Companions)
	out.Thoughts = maps.Clone(u.Thoughts)
	out.DeckCharacters = maps.Clone(u.DeckCharacters)
	out.DeckSubWeapons = cloneSliceMap(u.DeckSubWeapons)
	out.DeckParts = cloneSliceMap(u.DeckParts)
	out.Decks = maps.Clone(u.Decks)
	out.TripleDecks = maps.Clone(u.TripleDecks)
	out.Quests = maps.Clone(u.Quests)
	out.QuestMissions = maps.Clone(u.QuestMissions)
	out.WeaponStories = maps.Clone(u.WeaponStories)
	out.Missions = maps.Clone(u.Missions)
	out.Gimmick = GimmickState{
		Progress:         maps.Clone(u.Gimmick.Progress),
		OrnamentProgress: maps.Clone(u.Gimmick.OrnamentProgress),
		Sequences:        maps.Clone(u.Gimmick.Sequences),
		Unlocks:          maps.Clone(u.Gimmick.Unlocks),
	}
	out.CageOrnamentRewards = maps.Clone(u.CageOrnamentRewards)
	out.TowerAccumulationRewards = maps.Clone(u.TowerAccumulationRewards)
	out.LabyrinthSeasons = maps.Clone(u.LabyrinthSeasons)
	out.LabyrinthStages = maps.Clone(u.LabyrinthStages)
	out.ConsumableItems = maps.Clone(u.ConsumableItems)
	out.Materials = maps.Clone(u.Materials)
	out.Parts = maps.Clone(u.Parts)
	out.PartsGroupNotes = maps.Clone(u.PartsGroupNotes)
	out.PartsPresets = maps.Clone(u.PartsPresets)
	out.PartsPresetTags = maps.Clone(u.PartsPresetTags)
	out.PartsStatusSubs = maps.Clone(u.PartsStatusSubs)
	out.ImportantItems = maps.Clone(u.ImportantItems)
	out.CostumeActiveSkills = maps.Clone(u.CostumeActiveSkills)
	out.WeaponSkills = cloneSliceMap(u.WeaponSkills)
	out.WeaponAbilities = cloneSliceMap(u.WeaponAbilities)
	out.WeaponAwakens = maps.Clone(u.WeaponAwakens)
	out.DeckTypeNotes = maps.Clone(u.DeckTypeNotes)
	out.WeaponNotes = maps.Clone(u.WeaponNotes)
	out.NaviCutInPlayed = maps.Clone(u.NaviCutInPlayed)
	out.ViewedMovies = maps.Clone(u.ViewedMovies)
	out.ContentsStories = maps.Clone(u.ContentsStories)
	out.DrawnOmikuji = maps.Clone(u.DrawnOmikuji)
	out.PremiumItems = maps.Clone(u.PremiumItems)
	out.DokanConfirmed = maps.Clone(u.DokanConfirmed)
	out.ShopItems = maps.Clone(u.ShopItems)
	out.ShopReplaceableLineup = maps.Clone(u.ShopReplaceableLineup)
	out.Explore = u.Explore
	out.ExploreScores = maps.Clone(u.ExploreScores)
	out.Gacha = GachaState{
		RewardAvailable:        u.Gacha.RewardAvailable,
		TodaysCurrentDrawCount: u.Gacha.TodaysCurrentDrawCount,
		DailyMaxCount:          u.Gacha.DailyMaxCount,
		LastRewardDrawDate:     u.Gacha.LastRewardDrawDate,
		ConvertedGachaMedal: ConvertedGachaMedalState{
			ConvertedMedalPossession: append([]ConsumableItemState(nil), u.Gacha.ConvertedGachaMedal.ConvertedMedalPossession...),
			ObtainPossession:         cloneConsumableItemPtr(u.Gacha.ConvertedGachaMedal.ObtainPossession),
		},
		BannerStates: cloneBannerStates(u.Gacha.BannerStates),
	}
	out.Gifts = GiftState{
		NotReceived: cloneNotReceivedGifts(u.Gifts.NotReceived),
		Received:    cloneReceivedGifts(u.Gifts.Received),
	}
	out.Battle = u.Battle
	out.SideStoryQuests = maps.Clone(u.SideStoryQuests)
	out.QuestLimitContentStatus = maps.Clone(u.QuestLimitContentStatus)
	out.BigHuntMaxScores = maps.Clone(u.BigHuntMaxScores)
	out.BigHuntStatuses = maps.Clone(u.BigHuntStatuses)
	out.BigHuntScheduleMaxScores = maps.Clone(u.BigHuntScheduleMaxScores)
	out.BigHuntWeeklyMaxScores = maps.Clone(u.BigHuntWeeklyMaxScores)
	out.BigHuntWeeklyStatuses = maps.Clone(u.BigHuntWeeklyStatuses)
	out.BigHuntBattleBinary = append([]byte(nil), u.BigHuntBattleBinary...)
	out.CharacterBoards = maps.Clone(u.CharacterBoards)
	out.CharacterBoardAbilities = maps.Clone(u.CharacterBoardAbilities)
	out.CharacterBoardStatusUps = maps.Clone(u.CharacterBoardStatusUps)
	out.CostumeAwakenStatusUps = maps.Clone(u.CostumeAwakenStatusUps)
	out.CostumeLotteryEffects = maps.Clone(u.CostumeLotteryEffects)
	out.CostumeLotteryEffectPending = maps.Clone(u.CostumeLotteryEffectPending)
	out.AutoSaleSettings = maps.Clone(u.AutoSaleSettings)
	out.CharacterRebirths = maps.Clone(u.CharacterRebirths)
	out.QuestAutoOrbit.AccumulatedDrops = append([]AutoOrbitDropEntry(nil), u.QuestAutoOrbit.AccumulatedDrops...)
	return out
}

func cloneBannerStates(m map[int32]GachaBannerState) map[int32]GachaBannerState {
	if m == nil {
		return nil
	}
	out := make(map[int32]GachaBannerState, len(m))
	for k, v := range m {
		bs := v
		bs.BoxDrewCounts = maps.Clone(v.BoxDrewCounts)
		out[k] = bs
	}
	return out
}

func cloneConsumableItemPtr(item *ConsumableItemState) *ConsumableItemState {
	if item == nil {
		return nil
	}
	out := *item
	return &out
}

func cloneNotReceivedGifts(gifts []NotReceivedGiftState) []NotReceivedGiftState {
	out := make([]NotReceivedGiftState, len(gifts))
	for i, gift := range gifts {
		out[i] = NotReceivedGiftState{
			GiftCommon: GiftCommonState{
				PossessionType:        gift.GiftCommon.PossessionType,
				PossessionId:          gift.GiftCommon.PossessionId,
				Count:                 gift.GiftCommon.Count,
				GrantDatetime:         gift.GiftCommon.GrantDatetime,
				DescriptionGiftTextId: gift.GiftCommon.DescriptionGiftTextId,
				EquipmentData:         append([]byte(nil), gift.GiftCommon.EquipmentData...),
			},
			ExpirationDatetime: gift.ExpirationDatetime,
			UserGiftUuid:       gift.UserGiftUuid,
		}
	}
	return out
}

func cloneReceivedGifts(gifts []ReceivedGiftState) []ReceivedGiftState {
	out := make([]ReceivedGiftState, len(gifts))
	for i, gift := range gifts {
		out[i] = ReceivedGiftState{
			GiftCommon: GiftCommonState{
				PossessionType:        gift.GiftCommon.PossessionType,
				PossessionId:          gift.GiftCommon.PossessionId,
				Count:                 gift.GiftCommon.Count,
				GrantDatetime:         gift.GiftCommon.GrantDatetime,
				DescriptionGiftTextId: gift.GiftCommon.DescriptionGiftTextId,
				EquipmentData:         append([]byte(nil), gift.GiftCommon.EquipmentData...),
			},
			ReceivedDatetime: gift.ReceivedDatetime,
		}
	}
	return out
}

func cloneSliceMap[T any](m map[string][]T) map[string][]T {
	if m == nil {
		return nil
	}
	out := make(map[string][]T, len(m))
	for k, v := range m {
		out[k] = append([]T(nil), v...)
	}
	return out
}
