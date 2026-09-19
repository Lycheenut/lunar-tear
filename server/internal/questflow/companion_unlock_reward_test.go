package questflow

import (
	"reflect"
	"strconv"
	"testing"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/store"
)

func TestCompanionTutorialGrantsOnlySelectedCompanion(t *testing.T) {
	for _, receivedGift := range []bool{false, true} {
		for choice, selected := range map[int32]int32{1: 2, 2: 1, 3: 7, 4: 10} {
			t.Run(strconv.FormatBool(receivedGift)+"/"+strconv.Itoa(int(choice)), func(t *testing.T) {
				user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
				handler := &QuestHandler{Granter: &store.PossessionGranter{}}
				if receivedGift {
					store.GrantCompanionUnlockReward(user, handler.Granter, 100)
				}
				before := make(map[string]store.CompanionState)
				for key, companion := range user.Companions {
					before[key] = companion
				}
				grants := handler.ApplyTutorialReward(user, model.TutorialTypeCompanion, choice, 1000)
				want := []RewardGrant{{PossessionType: model.PossessionTypeCompanion, PossessionId: selected, Count: 1}}
				if !reflect.DeepEqual(grants, want) || len(user.Companions) != len(before)+1 {
					t.Fatalf("grants=%v companions=%d", grants, len(user.Companions))
				}
				for key, companion := range user.Companions {
					if original, exists := before[key]; exists {
						if companion != original {
							t.Fatalf("tutorial changed a gifted companion: %+v", companion)
						}
					} else if companion.CompanionId != selected || companion.Level != 1 {
						t.Fatalf("unexpected selected companion: %+v", companion)
					}
				}
			})
		}
	}
}

func TestCompanionTutorialRejectsInvalidChoices(t *testing.T) {
	handler := &QuestHandler{Granter: &store.PossessionGranter{}}
	user := store.SeedUserState(1, "test", 1, model.ClientPlatform{})
	for _, choice := range []int32{0, -1, 5} {
		if grants := handler.ApplyTutorialReward(user, model.TutorialTypeCompanion, choice, 1000); len(grants) != 0 || len(user.Companions) != 0 {
			t.Fatal("invalid choice granted companions")
		}
	}
}
