package service

import (
	"bytes"
	"encoding/json"
	"testing"

	"google.golang.org/protobuf/proto"

	pb "lunar-tear/server/gen/proto"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/questflow"
	"lunar-tear/server/internal/store"
)

func TestQuestPartsEquipmentSurvivesResponsesAndAutoOrbit(t *testing.T) {
	drops := []questflow.RewardGrant{
		{PossessionType: model.PossessionTypeParts, PossessionId: 16, Count: 1, EquipmentData: []byte{8, 1, 16, 24}},
		{PossessionType: model.PossessionTypeParts, PossessionId: 17, Count: 1, EquipmentData: []byte{8, 1, 16, 4, 26, 12, 8, 1, 16, 8, 24, 1, 32, 7, 40, 1, 48, 50}},
		{PossessionType: model.PossessionTypeParts, PossessionId: 18, Count: 1, IsAutoSale: true},
	}
	check := func(rewards []*pb.QuestReward) {
		t.Helper()
		if len(rewards) != len(drops) {
			t.Fatalf("rewards=%d, want %d", len(rewards), len(drops))
		}
		for i, reward := range rewards {
			wire, err := proto.Marshal(reward)
			if err != nil {
				t.Fatal(err)
			}
			var decoded pb.QuestReward
			if err := proto.Unmarshal(wire, &decoded); err != nil {
				t.Fatal(err)
			}
			if decoded.PossessionId != drops[i].PossessionId || decoded.Count != 1 || decoded.IsAutoSale != drops[i].IsAutoSale || !bytes.Equal(decoded.EquipmentData, drops[i].EquipmentData) {
				t.Fatalf("reward %d lost its rolled rank, sale result or equipment: %v", i, &decoded)
			}
		}
	}
	check(toProtoRewards(drops))
	user := store.SeedUserState(1, "parts", 1, model.ClientPlatform{})
	startAutoOrbit(user, model.QuestTypeEvent, 1, 10, 2, 1000)
	if _, ended := finishAutoOrbit(user, true, false, false, model.QuestTypeEvent, 1, 10, 1001, drops[:1]); ended {
		t.Fatal("auto orbit ended before the second run")
	}
	// The accumulator is persisted as JSON between runs, including after a
	// server restart; the final response must retain each item's own snapshot.
	saved, err := json.Marshal(user.QuestAutoOrbit.AccumulatedDrops)
	if err != nil {
		t.Fatal(err)
	}
	user.QuestAutoOrbit.AccumulatedDrops = nil
	if err := json.Unmarshal(saved, &user.QuestAutoOrbit.AccumulatedDrops); err != nil {
		t.Fatal(err)
	}
	endedDrops, ended := finishAutoOrbit(user, true, false, false, model.QuestTypeEvent, 1, 10, 1002, drops[1:])
	if !ended {
		t.Fatal("auto orbit did not end after the second run")
	}
	check(autoOrbitDropsToProto(endedDrops))
	check(autoOrbitDropsToProto(consumeAutoOrbitRewards(user)))
}
