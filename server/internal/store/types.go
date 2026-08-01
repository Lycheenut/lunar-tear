package store

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"lunar-tear/server/internal/model"
)

type SessionState struct {
	SessionKey string
	UserId     int64
	Uuid       string
	ExpireAt   time.Time
}

type UserState struct {
	UserId              int64
	Uuid                string
	PlayerId            int64
	OsType              int32
	PlatformType        int32
	UserRestrictionType int32
	RegisterDatetime    int64
	GameStartDatetime   int64
	LatestVersion       int64

	BirthYear            int32
	BirthMonth           int32
	BackupToken          string
	ChargeMoneyThisMonth int64
	FacebookId           int64

	Setting                 UserSettingState
	Status                  UserStatusState
	Gem                     UserGemState
	Profile                 UserProfileState
	Login                   UserLoginState
	LoginBonus              UserLoginBonusState
	Tutorials               map[int32]TutorialProgressState
	MainQuest               MainQuestState
	EventQuest              EventQuestState
	ExtraQuest              ExtraQuestState
	SideStoryQuests         map[int32]SideStoryQuestProgress
	SideStoryActiveProgress SideStoryActiveProgress
	QuestLimitContentStatus map[int32]QuestLimitContentStatus

	BigHuntProgress          BigHuntProgress
	BigHuntMaxScores         map[int32]BigHuntMaxScore
	BigHuntStatuses          map[int32]BigHuntStatus
	BigHuntScheduleMaxScores map[BigHuntScheduleScoreKey]BigHuntScheduleMaxScore
	BigHuntWeeklyMaxScores   map[BigHuntWeeklyScoreKey]BigHuntWeeklyMaxScore
	BigHuntWeeklyStatuses    map[int64]BigHuntWeeklyStatus
	BigHuntBattleBinary      []byte
	BigHuntBattleDetail      BigHuntBattleDetail
	BigHuntDeckNumber        int32

	Battle         BattleState
	Gifts          GiftState
	Gacha          GachaState
	Notifications  NotificationState
	Friends        map[int64]FriendState
	FriendRequests map[int64]int64

	Characters                 map[int32]CharacterState
	Costumes                   map[string]CostumeState
	Weapons                    map[string]WeaponState
	Companions                 map[string]CompanionState
	Thoughts                   map[string]ThoughtState
	DeckCharacters             map[string]DeckCharacterState
	Decks                      map[DeckKey]DeckState
	TripleDecks                map[DeckKey]TripleDeckState
	DeckLimitContentRestricted map[string]DeckLimitContentRestrictedState
	Quests                     map[int32]UserQuestState
	QuestMissions              map[QuestMissionKey]UserQuestMissionState
	Missions                   map[int32]UserMissionState
	QuestReplayFlowRewards     map[int32]QuestReplayFlowRewardState
	QuestSceneChoices          map[QuestSceneChoiceKey]QuestSceneChoiceState
	QuestSceneChoiceHistory    map[QuestSceneChoiceHistoryKey]QuestSceneChoiceState
	EventQuestDailyRewards     map[int32]EventQuestDailyRewardState
	MissionPassPoints          map[int32]MissionPassPointState
	MissionPassRewards         map[MissionPassRewardKey]MissionPassRewardState
	MissionPassRemaining       map[int32]MissionPassRemainingState
	WebviewPanelMissions       map[int32]WebviewPanelMissionState
	WeaponStories              map[int32]WeaponStoryState
	Gimmick                    GimmickState
	CageOrnamentRewards        map[int32]CageOrnamentRewardState
	CageOrnamentAccesses       map[int32]CageOrnamentAccessState
	TowerAccumulationRewards   map[int32]TowerAccumulationRewardState
	LabyrinthSeasons           map[int32]LabyrinthSeasonState
	LabyrinthStages            map[LabyrinthStageKey]LabyrinthStageState
	ConsumableItems            map[int32]int32
	Materials                  map[int32]int32
	Parts                      map[string]PartsState
	PartsGroupNotes            map[int32]PartsGroupNoteState
	PartsPresets               map[int32]PartsPresetState
	PartsPresetTags            map[int32]PartsPresetTagState
	PartsStatusSubs            map[PartsStatusSubKey]PartsStatusSubState
	ImportantItems             map[int32]int32
	CostumeActiveSkills        map[string]CostumeActiveSkillState
	WeaponSkills               map[string][]WeaponSkillState   // key: userWeaponUuid
	WeaponAbilities            map[string][]WeaponAbilityState // key: userWeaponUuid
	WeaponAwakens              map[string]WeaponAwakenState    // key: userWeaponUuid
	DeckTypeNotes              map[model.DeckType]DeckTypeNoteState
	WeaponNotes                map[int32]WeaponNoteState
	DeckSubWeapons             map[string][]string
	DeckParts                  map[string][]string
	NaviCutInPlayed            map[int32]bool
	ViewedMovies               map[int32]int64
	ContentsStories            map[int32]int64
	DrawnOmikuji               map[int32]int64
	PremiumItems               map[int32]int64
	DokanConfirmed             map[int32]bool
	PortalCageStatus           PortalCageStatusState
	GuerrillaFreeOpen          GuerrillaFreeOpenState
	ShopItems                  map[int32]UserShopItemState
	ShopReplaceable            UserShopReplaceableState
	ShopReplaceableLineup      map[int32]UserShopReplaceableLineupState

	Explore       ExploreState
	ExploreScores map[int32]ExploreScoreState

	CharacterBoards         map[int32]CharacterBoardState
	CharacterBoardAbilities map[CharacterBoardAbilityKey]CharacterBoardAbilityState
	CharacterBoardStatusUps map[CharacterBoardStatusUpKey]CharacterBoardStatusUpState

	CostumeAwakenStatusUps           map[CostumeAwakenStatusKey]CostumeAwakenStatusUpState
	CostumeLevelBonusReleaseStatuses map[int32]CostumeLevelBonusReleaseStatusState
	CostumeLotteryEffects            map[CostumeLotteryEffectKey]CostumeLotteryEffectState
	CostumeLotteryEffectAbilities    map[CostumeLotteryEffectKey]CostumeLotteryEffectAbilityState
	CostumeLotteryEffectStatusUps    map[CostumeLotteryEffectStatusKey]CostumeLotteryEffectStatusUpState
	CostumeLotteryEffectPending      map[string]CostumeLotteryEffectPendingState // key: userCostumeUuid
	AutoSaleSettings                 map[int32]AutoSaleSettingState
	CharacterRebirths                map[int32]CharacterRebirthState
	QuestAutoOrbit                   QuestAutoOrbitState
}

func (u *UserState) EnsureMaps() {
	if u.Friends == nil {
		u.Friends = make(map[int64]FriendState)
	}
	if u.FriendRequests == nil {
		u.FriendRequests = make(map[int64]int64)
	}
	if u.Tutorials == nil {
		u.Tutorials = make(map[int32]TutorialProgressState)
	}
	if u.Characters == nil {
		u.Characters = make(map[int32]CharacterState)
	}
	if u.Costumes == nil {
		u.Costumes = make(map[string]CostumeState)
	}
	if u.Weapons == nil {
		u.Weapons = make(map[string]WeaponState)
	}
	if u.Companions == nil {
		u.Companions = make(map[string]CompanionState)
	}
	if u.Thoughts == nil {
		u.Thoughts = make(map[string]ThoughtState)
	}
	if u.DeckCharacters == nil {
		u.DeckCharacters = make(map[string]DeckCharacterState)
	}
	if u.Decks == nil {
		u.Decks = make(map[DeckKey]DeckState)
	}
	if u.TripleDecks == nil {
		u.TripleDecks = make(map[DeckKey]TripleDeckState)
	}
	if u.DeckLimitContentRestricted == nil {
		u.DeckLimitContentRestricted = make(map[string]DeckLimitContentRestrictedState)
	}
	if u.DeckSubWeapons == nil {
		u.DeckSubWeapons = make(map[string][]string)
	}
	if u.DeckParts == nil {
		u.DeckParts = make(map[string][]string)
	}
	if u.Quests == nil {
		u.Quests = make(map[int32]UserQuestState)
	}
	if u.SideStoryQuests == nil {
		u.SideStoryQuests = make(map[int32]SideStoryQuestProgress)
	}
	if u.QuestLimitContentStatus == nil {
		u.QuestLimitContentStatus = make(map[int32]QuestLimitContentStatus)
	}
	if u.BigHuntMaxScores == nil {
		u.BigHuntMaxScores = make(map[int32]BigHuntMaxScore)
	}
	if u.BigHuntStatuses == nil {
		u.BigHuntStatuses = make(map[int32]BigHuntStatus)
	}
	if u.BigHuntScheduleMaxScores == nil {
		u.BigHuntScheduleMaxScores = make(map[BigHuntScheduleScoreKey]BigHuntScheduleMaxScore)
	}
	if u.BigHuntWeeklyMaxScores == nil {
		u.BigHuntWeeklyMaxScores = make(map[BigHuntWeeklyScoreKey]BigHuntWeeklyMaxScore)
	}
	if u.BigHuntWeeklyStatuses == nil {
		u.BigHuntWeeklyStatuses = make(map[int64]BigHuntWeeklyStatus)
	}
	if u.QuestMissions == nil {
		u.QuestMissions = make(map[QuestMissionKey]UserQuestMissionState)
	}
	if u.Missions == nil {
		u.Missions = make(map[int32]UserMissionState)
	}
	if u.QuestReplayFlowRewards == nil {
		u.QuestReplayFlowRewards = make(map[int32]QuestReplayFlowRewardState)
	}
	if u.QuestSceneChoices == nil {
		u.QuestSceneChoices = make(map[QuestSceneChoiceKey]QuestSceneChoiceState)
	}
	if u.QuestSceneChoiceHistory == nil {
		u.QuestSceneChoiceHistory = make(map[QuestSceneChoiceHistoryKey]QuestSceneChoiceState)
	}
	if u.EventQuestDailyRewards == nil {
		u.EventQuestDailyRewards = make(map[int32]EventQuestDailyRewardState)
	}
	if u.MissionPassPoints == nil {
		u.MissionPassPoints = make(map[int32]MissionPassPointState)
	}
	if u.MissionPassRewards == nil {
		u.MissionPassRewards = make(map[MissionPassRewardKey]MissionPassRewardState)
	}
	if u.MissionPassRemaining == nil {
		u.MissionPassRemaining = make(map[int32]MissionPassRemainingState)
	}
	if u.WebviewPanelMissions == nil {
		u.WebviewPanelMissions = make(map[int32]WebviewPanelMissionState)
	}
	if u.WeaponStories == nil {
		u.WeaponStories = make(map[int32]WeaponStoryState)
	}
	if u.CageOrnamentRewards == nil {
		u.CageOrnamentRewards = make(map[int32]CageOrnamentRewardState)
	}
	if u.CageOrnamentAccesses == nil {
		u.CageOrnamentAccesses = make(map[int32]CageOrnamentAccessState)
	}
	if u.TowerAccumulationRewards == nil {
		u.TowerAccumulationRewards = make(map[int32]TowerAccumulationRewardState)
	}
	if u.LabyrinthSeasons == nil {
		u.LabyrinthSeasons = make(map[int32]LabyrinthSeasonState)
	}
	if u.LabyrinthStages == nil {
		u.LabyrinthStages = make(map[LabyrinthStageKey]LabyrinthStageState)
	}
	if u.ConsumableItems == nil {
		u.ConsumableItems = make(map[int32]int32)
	}
	if u.Materials == nil {
		u.Materials = make(map[int32]int32)
	}
	if u.Parts == nil {
		u.Parts = make(map[string]PartsState)
	}
	if u.PartsGroupNotes == nil {
		u.PartsGroupNotes = make(map[int32]PartsGroupNoteState)
	}
	if u.PartsPresets == nil {
		u.PartsPresets = make(map[int32]PartsPresetState)
	}
	if u.PartsPresetTags == nil {
		u.PartsPresetTags = make(map[int32]PartsPresetTagState)
	}
	if u.PartsStatusSubs == nil {
		u.PartsStatusSubs = make(map[PartsStatusSubKey]PartsStatusSubState)
	}
	if u.ImportantItems == nil {
		u.ImportantItems = make(map[int32]int32)
	}
	if u.CostumeActiveSkills == nil {
		u.CostumeActiveSkills = make(map[string]CostumeActiveSkillState)
	}
	if u.WeaponSkills == nil {
		u.WeaponSkills = make(map[string][]WeaponSkillState)
	}
	if u.WeaponAbilities == nil {
		u.WeaponAbilities = make(map[string][]WeaponAbilityState)
	}
	if u.WeaponAwakens == nil {
		u.WeaponAwakens = make(map[string]WeaponAwakenState)
	}
	if u.DeckTypeNotes == nil {
		u.DeckTypeNotes = make(map[model.DeckType]DeckTypeNoteState)
	}
	if u.WeaponNotes == nil {
		u.WeaponNotes = make(map[int32]WeaponNoteState)
	}
	if u.NaviCutInPlayed == nil {
		u.NaviCutInPlayed = make(map[int32]bool)
	}
	if u.ViewedMovies == nil {
		u.ViewedMovies = make(map[int32]int64)
	}
	if u.ContentsStories == nil {
		u.ContentsStories = make(map[int32]int64)
	}
	if u.DrawnOmikuji == nil {
		u.DrawnOmikuji = make(map[int32]int64)
	}
	if u.PremiumItems == nil {
		u.PremiumItems = make(map[int32]int64)
	}
	if u.DokanConfirmed == nil {
		u.DokanConfirmed = make(map[int32]bool)
	}
	if u.ShopItems == nil {
		u.ShopItems = make(map[int32]UserShopItemState)
	}
	if u.ShopReplaceableLineup == nil {
		u.ShopReplaceableLineup = make(map[int32]UserShopReplaceableLineupState)
	}
	if u.ExploreScores == nil {
		u.ExploreScores = make(map[int32]ExploreScoreState)
	}
	if u.CharacterBoards == nil {
		u.CharacterBoards = make(map[int32]CharacterBoardState)
	}
	if u.CharacterBoardAbilities == nil {
		u.CharacterBoardAbilities = make(map[CharacterBoardAbilityKey]CharacterBoardAbilityState)
	}
	if u.CharacterBoardStatusUps == nil {
		u.CharacterBoardStatusUps = make(map[CharacterBoardStatusUpKey]CharacterBoardStatusUpState)
	}
	if u.CostumeAwakenStatusUps == nil {
		u.CostumeAwakenStatusUps = make(map[CostumeAwakenStatusKey]CostumeAwakenStatusUpState)
	}
	if u.CostumeLevelBonusReleaseStatuses == nil {
		u.CostumeLevelBonusReleaseStatuses = make(map[int32]CostumeLevelBonusReleaseStatusState)
	}
	if u.CostumeLotteryEffects == nil {
		u.CostumeLotteryEffects = make(map[CostumeLotteryEffectKey]CostumeLotteryEffectState)
	}
	if u.CostumeLotteryEffectAbilities == nil {
		u.CostumeLotteryEffectAbilities = make(map[CostumeLotteryEffectKey]CostumeLotteryEffectAbilityState)
	}
	if u.CostumeLotteryEffectStatusUps == nil {
		u.CostumeLotteryEffectStatusUps = make(map[CostumeLotteryEffectStatusKey]CostumeLotteryEffectStatusUpState)
	}
	if u.CostumeLotteryEffectPending == nil {
		u.CostumeLotteryEffectPending = make(map[string]CostumeLotteryEffectPendingState)
	}
	if u.AutoSaleSettings == nil {
		u.AutoSaleSettings = make(map[int32]AutoSaleSettingState)
	}
	if u.CharacterRebirths == nil {
		u.CharacterRebirths = make(map[int32]CharacterRebirthState)
	}
	if u.Gimmick.Progress == nil {
		u.Gimmick.Progress = make(map[GimmickKey]GimmickProgressState)
	}
	if u.Gimmick.OrnamentProgress == nil {
		u.Gimmick.OrnamentProgress = make(map[GimmickOrnamentKey]GimmickOrnamentProgressState)
	}
	if u.Gimmick.Sequences == nil {
		u.Gimmick.Sequences = make(map[GimmickSequenceKey]GimmickSequenceState)
	}
	if u.Gimmick.Unlocks == nil {
		u.Gimmick.Unlocks = make(map[GimmickKey]GimmickUnlockState)
	}
	if u.Gacha.BannerStates == nil {
		u.Gacha.BannerStates = make(map[int32]GachaBannerState)
	}
}

type ExploreState struct {
	IsUseExploreTicket bool
	PlayingExploreId   int32
	LatestPlayDatetime int64
	LatestVersion      int64
}

type ExploreScoreState struct {
	ExploreId              int32
	MaxScore               int32
	MaxScoreUpdateDatetime int64
	LatestVersion          int64
}

type GuerrillaFreeOpenState struct {
	StartDatetime    int64
	OpenMinutes      int32
	DailyOpenedCount int32
	LatestVersion    int64
}

type AutoOrbitDropEntry struct {
	PossessionType int32
	PossessionId   int32
	Count          int32
	IsAutoSale     bool
}

type QuestAutoOrbitState struct {
	QuestType             int32
	ChapterId             int32
	QuestId               int32
	MaxAutoOrbitCount     int32
	ClearedAutoOrbitCount int32
	LastClearDatetime     int64
	LatestVersion         int64
	AccumulatedDrops      []AutoOrbitDropEntry
}

func (s QuestAutoOrbitState) Equal(other QuestAutoOrbitState) bool {
	if s.QuestType != other.QuestType ||
		s.ChapterId != other.ChapterId ||
		s.QuestId != other.QuestId ||
		s.MaxAutoOrbitCount != other.MaxAutoOrbitCount ||
		s.ClearedAutoOrbitCount != other.ClearedAutoOrbitCount ||
		s.LastClearDatetime != other.LastClearDatetime ||
		s.LatestVersion != other.LatestVersion {
		return false
	}
	if len(s.AccumulatedDrops) != len(other.AccumulatedDrops) {
		return false
	}
	for i := range s.AccumulatedDrops {
		if s.AccumulatedDrops[i] != other.AccumulatedDrops[i] {
			return false
		}
	}
	return true
}

type PortalCageStatusState struct {
	IsCurrentProgress     bool
	DropItemStartDatetime int64
	CurrentDropItemCount  int32
	LatestVersion         int64
}

type UserSettingState struct {
	IsNotifyPurchaseAlert bool
	LatestVersion         int64
}

type UserStatusState struct {
	Level                 int32
	Exp                   int32
	StaminaMilliValue     int32
	StaminaUpdateDatetime int64
	LatestVersion         int64
}

type UserGemState struct {
	PaidGem int32
	FreeGem int32
}

type UserProfileState struct {
	Name                            string
	NameUpdateDatetime              int64
	Message                         string
	MessageUpdateDatetime           int64
	FavoriteCostumeId               int32
	FavoriteCostumeIdUpdateDatetime int64
	CurrentPvpRank                  int32
	CurrentPvpGradeId               int32
	MaxPvpSeasonRank                int32
	LatestVersion                   int64
}

type UserLoginState struct {
	TotalLoginCount           int32
	ContinualLoginCount       int32
	MaxContinualLoginCount    int32
	LastLoginDatetime         int64
	LastComebackLoginDatetime int64
	LatestVersion             int64
}

type UserLoginBonusState struct {
	LoginBonusId                int32
	CurrentPageNumber           int32
	CurrentStampNumber          int32
	LatestRewardReceiveDatetime int64
	LatestVersion               int64
}

type CharacterState struct {
	CharacterId   int32
	Level         int32
	Exp           int32
	LatestVersion int64
}

type CostumeState struct {
	UserCostumeUuid                       string
	CostumeId                             int32
	LimitBreakCount                       int32
	Level                                 int32
	Exp                                   int32
	HeadupDisplayViewId                   int32
	AcquisitionDatetime                   int64
	AwakenCount                           int32
	CostumeLotteryEffectUnlockedSlotCount int32
	LatestVersion                         int64
}

type WeaponState struct {
	UserWeaponUuid      string
	WeaponId            int32
	Level               int32
	Exp                 int32
	LimitBreakCount     int32
	IsProtected         bool
	AcquisitionDatetime int64
	LatestVersion       int64
}

type CompanionState struct {
	UserCompanionUuid   string
	CompanionId         int32
	HeadupDisplayViewId int32
	Level               int32
	AcquisitionDatetime int64
	LatestVersion       int64
}

type ThoughtState struct {
	UserThoughtUuid     string
	ThoughtId           int32
	AcquisitionDatetime int64
	LatestVersion       int64
}

type DeckCharacterState struct {
	UserDeckCharacterUuid string
	UserCostumeUuid       string
	MainUserWeaponUuid    string
	UserCompanionUuid     string
	Power                 int32
	UserThoughtUuid       string
	DressupCostumeId      int32
	LatestVersion         int64
}

type DeckKey struct {
	DeckType       model.DeckType
	UserDeckNumber int32
}

func (k DeckKey) MarshalText() ([]byte, error) {
	return marshalKey(int64(k.DeckType), int64(k.UserDeckNumber)), nil
}

func (k *DeckKey) UnmarshalText(text []byte) error {
	v, err := unmarshalKey(text, "DeckKey", 2)
	if err != nil {
		return err
	}
	k.DeckType = model.DeckType(v[0])
	k.UserDeckNumber = int32(v[1])
	return nil
}

type DeckState struct {
	DeckType                model.DeckType
	UserDeckNumber          int32
	UserDeckCharacterUuid01 string
	UserDeckCharacterUuid02 string
	UserDeckCharacterUuid03 string
	Name                    string
	Power                   int32
	LatestVersion           int64
}

type DeckLimitContentRestrictedState struct {
	DeckRestrictedUuid  string
	EventQuestChapterId int32
	QuestId             int32
	PossessionType      int32
	TargetUuid          string
	LatestVersion       int64
}

type TripleDeckState struct {
	DeckType       model.DeckType
	UserDeckNumber int32
	Name           string
	DeckNumber01   int32
	DeckNumber02   int32
	DeckNumber03   int32
	LatestVersion  int64
}

type DeckCharacterInput struct {
	UserCostumeUuid    string
	MainUserWeaponUuid string
	SubWeaponUuids     []string
	PartsUuids         []string
	UserCompanionUuid  string
	UserThoughtUuid    string
	DressupCostumeId   int32
}

type CostumeActiveSkillState struct {
	UserCostumeUuid     string
	Level               int32
	AcquisitionDatetime int64
	LatestVersion       int64
}

type WeaponSkillState struct {
	UserWeaponUuid string
	SlotNumber     int32
	Level          int32
}

type WeaponAbilityState struct {
	UserWeaponUuid string
	SlotNumber     int32
	Level          int32
}

type WeaponAwakenState struct {
	UserWeaponUuid string
	LatestVersion  int64
}

type DeckTypeNoteState struct {
	DeckType      model.DeckType
	MaxDeckPower  int32
	LatestVersion int64
}

type TutorialProgressState struct {
	TutorialType  int32
	ProgressPhase int32
	ChoiceId      int32
	LatestVersion int64
}

type MainQuestState struct {
	CurrentQuestFlowType     int32
	CurrentMainQuestRouteId  int32
	CurrentQuestSceneId      int32
	HeadQuestSceneId         int32
	IsReachedLastQuestScene  bool
	ProgressQuestSceneId     int32
	ProgressHeadQuestSceneId int32
	ProgressQuestFlowType    int32
	MainQuestSeasonId        int32
	LatestVersion            int64

	SavedContext                  SavedQuestContext
	ReplayFlowCurrentQuestSceneId int32
	ReplayFlowHeadQuestSceneId    int32
}

// SavedQuestContext snapshots player state when entering a menu-replay (cleared
// quest started from the Main Quest List menu). On finish, every field is
// restored atomically so the player returns to the exact pre-replay state.
type SavedQuestContext struct {
	Active                  bool
	CurrentQuestSceneId     int32
	HeadQuestSceneId        int32
	CurrentMainQuestRouteId int32
	MainQuestSeasonId       int32
	IsReachedLastQuestScene bool
	PortalCageInProgress    bool
	CurrentQuestFlowType    int32
}

type EventQuestState struct {
	CurrentEventQuestChapterId int32
	CurrentQuestId             int32
	CurrentQuestSceneId        int32
	HeadQuestSceneId           int32
	LatestVersion              int64
}

type ExtraQuestState struct {
	CurrentQuestId      int32
	CurrentQuestSceneId int32
	HeadQuestSceneId    int32
	LatestVersion       int64
}

type SideStoryQuestProgress struct {
	HeadSideStoryQuestSceneId int32
	SideStoryQuestStateType   model.SideStoryQuestStateType
	LatestVersion             int64
}

type SideStoryActiveProgress struct {
	CurrentSideStoryQuestId      int32
	CurrentSideStoryQuestSceneId int32
	LatestVersion                int64
}

type QuestLimitContentStatus struct {
	LimitContentQuestStatusType int32
	EventQuestChapterId         int32
	LatestVersion               int64
}

type BigHuntProgress struct {
	CurrentBigHuntBossQuestId int32
	CurrentBigHuntQuestId     int32
	CurrentQuestSceneId       int32
	IsDryRun                  bool
	LatestVersion             int64
}

type BigHuntMaxScore struct {
	MaxScore               int64
	MaxScoreUpdateDatetime int64
	LatestVersion          int64
}

type BigHuntStatus struct {
	DailyChallengeCount               int32
	LatestChallengeDatetime           int64
	LastDailyRewardReceivedDayVersion int64
	LatestVersion                     int64
}

type BigHuntScheduleScoreKey struct {
	BigHuntScheduleId int32
	BigHuntBossId     int32
}

func (k BigHuntScheduleScoreKey) MarshalText() ([]byte, error) {
	return marshalKey(int64(k.BigHuntScheduleId), int64(k.BigHuntBossId)), nil
}

func (k *BigHuntScheduleScoreKey) UnmarshalText(text []byte) error {
	v, err := unmarshalKey(text, "BigHuntScheduleScoreKey", 2)
	if err != nil {
		return err
	}
	k.BigHuntScheduleId = int32(v[0])
	k.BigHuntBossId = int32(v[1])
	return nil
}

type BigHuntScheduleMaxScore struct {
	MaxScore               int64
	MaxScoreUpdateDatetime int64
	LatestVersion          int64
}

type BigHuntWeeklyScoreKey struct {
	BigHuntWeeklyVersion int64
	AttributeType        int32
}

func (k BigHuntWeeklyScoreKey) MarshalText() ([]byte, error) {
	return marshalKey(k.BigHuntWeeklyVersion, int64(k.AttributeType)), nil
}

func (k *BigHuntWeeklyScoreKey) UnmarshalText(text []byte) error {
	v, err := unmarshalKey(text, "BigHuntWeeklyScoreKey", 2)
	if err != nil {
		return err
	}
	k.BigHuntWeeklyVersion = v[0]
	k.AttributeType = int32(v[1])
	return nil
}

type BigHuntWeeklyMaxScore struct {
	MaxScore      int64
	LatestVersion int64
}

type BigHuntWeeklyStatus struct {
	IsReceivedWeeklyReward bool
	LatestVersion          int64
}

type BigHuntBattleDetail struct {
	DeckType             int32
	UserTripleDeckNumber int32
	BossKnockDownCount   int32
	MaxComboCount        int32
	TotalDamage          int64
	CostumeBattleInfo    []BigHuntCostumeBattleInfo
}

type BigHuntCostumeBattleInfo struct {
	WaveIndex              int32
	CostumeId              int32
	TotalDamage            int64
	HitCount               int32
	RandomDisplayValueType int32
	RandomDisplayValue     int64
}

type BattleState struct {
	IsActive              bool
	StartCount            int32
	FinishCount           int32
	LastStartedAt         int64
	LastFinishedAt        int64
	LastUserPartyCount    int32
	LastNpcPartyCount     int32
	LastBattleBinarySize  int32
	LastElapsedFrameCount int64
	MissionDetail         BattleMissionDetailState
}

type BattleMissionDetailState struct {
	IsValid                bool
	CharacterDeathCount    int32
	MaxDamage              int64
	CostumeSkillUseCount   int32
	WeaponSkillUseCount    int32
	CompanionSkillUseCount int32
	CriticalCount          int32
	ComboCount             int32
	ComboMaxDamage         int64
	TotalRecoverPoint      int64
	CostumeResults         [3]CostumeBattleResultState
	CostumeResultCount     int32
}

type CostumeBattleResultState struct {
	IsAlive     bool
	MaxHp       int64
	RemainingHp int64
}

type UserQuestState struct {
	QuestId             int32
	QuestStateType      model.UserQuestStateType
	IsBattleOnly        bool
	UserDeckNumber      int32
	LatestStartDatetime int64
	ClearCount          int32
	DailyClearCount     int32
	LastClearDatetime   int64
	ShortestClearFrames int32
	IsRewardGranted     bool
	LatestVersion       int64
}

type QuestMissionKey struct {
	QuestId        int32
	QuestMissionId int32
}

func (k QuestMissionKey) MarshalText() ([]byte, error) {
	return marshalKey(int64(k.QuestId), int64(k.QuestMissionId)), nil
}

func (k *QuestMissionKey) UnmarshalText(text []byte) error {
	v, err := unmarshalKey(text, "QuestMissionKey", 2)
	if err != nil {
		return err
	}
	k.QuestId = int32(v[0])
	k.QuestMissionId = int32(v[1])
	return nil
}

type UserQuestMissionState struct {
	QuestId             int32
	QuestMissionId      int32
	ProgressValue       int32
	IsClear             bool
	LatestClearDatetime int64
	LatestVersion       int64
}

type UserMissionState struct {
	MissionId                 int32
	StartDatetime             int64
	ProgressValue             int32
	MissionProgressStatusType int32
	ClearDatetime             int64
	LatestVersion             int64
}

type QuestReplayFlowRewardState struct {
	QuestReplayFlowRewardGroupId int32
	RewardReceiveDatetime        int64
	LatestVersion                int64
}

type QuestSceneChoiceKey struct {
	QuestSceneId  int32
	QuestFlowType int32
}

func (k QuestSceneChoiceKey) MarshalText() ([]byte, error) {
	return marshalKey(int64(k.QuestSceneId), int64(k.QuestFlowType)), nil
}

func (k *QuestSceneChoiceKey) UnmarshalText(text []byte) error {
	v, err := unmarshalKey(text, "QuestSceneChoiceKey", 2)
	if err != nil {
		return err
	}
	k.QuestSceneId, k.QuestFlowType = int32(v[0]), int32(v[1])
	return nil
}

type QuestSceneChoiceState struct {
	QuestSceneId   int32
	QuestFlowType  int32
	ChoiceNumber   int32
	ChoiceDatetime int64
	LatestVersion  int64
}

type QuestSceneChoiceHistoryKey struct {
	QuestSceneId  int32
	QuestFlowType int32
	ChoiceNumber  int32
}

func (k QuestSceneChoiceHistoryKey) MarshalText() ([]byte, error) {
	return marshalKey(int64(k.QuestSceneId), int64(k.QuestFlowType), int64(k.ChoiceNumber)), nil
}

func (k *QuestSceneChoiceHistoryKey) UnmarshalText(text []byte) error {
	v, err := unmarshalKey(text, "QuestSceneChoiceHistoryKey", 3)
	if err != nil {
		return err
	}
	k.QuestSceneId, k.QuestFlowType, k.ChoiceNumber = int32(v[0]), int32(v[1]), int32(v[2])
	return nil
}

type EventQuestDailyRewardState struct {
	EventQuestDailyGroupId int32
	RewardReceiveDatetime  int64
	LatestVersion          int64
}

type MissionPassPointState struct {
	MissionPassId int32
	Point         int32
	LatestVersion int64
}

type MissionPassRewardKey struct {
	MissionPassId int32
	Level         int32
	IsPremium     bool
}

func (k MissionPassRewardKey) MarshalText() ([]byte, error) {
	premium := int64(0)
	if k.IsPremium {
		premium = 1
	}
	return marshalKey(int64(k.MissionPassId), int64(k.Level), premium), nil
}

func (k *MissionPassRewardKey) UnmarshalText(text []byte) error {
	v, err := unmarshalKey(text, "MissionPassRewardKey", 3)
	if err != nil {
		return err
	}
	k.MissionPassId, k.Level, k.IsPremium = int32(v[0]), int32(v[1]), v[2] != 0
	return nil
}

type MissionPassRewardState struct {
	MissionPassId         int32
	Level                 int32
	IsPremium             bool
	RewardReceiveDatetime int64
	LatestVersion         int64
}

type MissionPassRemainingState struct {
	MissionPassId         int32
	RewardReceived        bool
	RewardReceiveDatetime int64
	LatestVersion         int64
}

type WebviewPanelMissionState struct {
	WebviewPanelMissionPageId int32
	RewardReceiveDatetime     int64
	LatestVersion             int64
}

type WeaponStoryState struct {
	WeaponId              int32
	ReleasedMaxStoryIndex int32
	LatestVersion         int64
}

type WeaponNoteState struct {
	WeaponId                 int32
	MaxLevel                 int32
	MaxLimitBreakCount       int32
	FirstAcquisitionDatetime int64
	LatestVersion            int64
}

type GimmickSequenceKey struct {
	GimmickSequenceScheduleId int32
	GimmickSequenceId         int32
}

func (k GimmickSequenceKey) MarshalText() ([]byte, error) {
	return marshalKey(int64(k.GimmickSequenceScheduleId), int64(k.GimmickSequenceId)), nil
}

func (k *GimmickSequenceKey) UnmarshalText(text []byte) error {
	v, err := unmarshalKey(text, "GimmickSequenceKey", 2)
	if err != nil {
		return err
	}
	k.GimmickSequenceScheduleId = int32(v[0])
	k.GimmickSequenceId = int32(v[1])
	return nil
}

type GimmickKey struct {
	GimmickSequenceScheduleId int32
	GimmickSequenceId         int32
	GimmickId                 int32
}

func (k GimmickKey) MarshalText() ([]byte, error) {
	return marshalKey(int64(k.GimmickSequenceScheduleId), int64(k.GimmickSequenceId), int64(k.GimmickId)), nil
}

func (k *GimmickKey) UnmarshalText(text []byte) error {
	v, err := unmarshalKey(text, "GimmickKey", 3)
	if err != nil {
		return err
	}
	k.GimmickSequenceScheduleId = int32(v[0])
	k.GimmickSequenceId = int32(v[1])
	k.GimmickId = int32(v[2])
	return nil
}

type GimmickOrnamentKey struct {
	GimmickSequenceScheduleId int32
	GimmickSequenceId         int32
	GimmickId                 int32
	GimmickOrnamentIndex      int32
}

func (k GimmickOrnamentKey) MarshalText() ([]byte, error) {
	return marshalKey(int64(k.GimmickSequenceScheduleId), int64(k.GimmickSequenceId), int64(k.GimmickId), int64(k.GimmickOrnamentIndex)), nil
}

func (k *GimmickOrnamentKey) UnmarshalText(text []byte) error {
	v, err := unmarshalKey(text, "GimmickOrnamentKey", 4)
	if err != nil {
		return err
	}
	k.GimmickSequenceScheduleId = int32(v[0])
	k.GimmickSequenceId = int32(v[1])
	k.GimmickId = int32(v[2])
	k.GimmickOrnamentIndex = int32(v[3])
	return nil
}

type GimmickState struct {
	Progress         map[GimmickKey]GimmickProgressState
	OrnamentProgress map[GimmickOrnamentKey]GimmickOrnamentProgressState
	Sequences        map[GimmickSequenceKey]GimmickSequenceState
	Unlocks          map[GimmickKey]GimmickUnlockState
}

type GimmickProgressState struct {
	Key              GimmickKey
	IsGimmickCleared bool
	StartDatetime    int64
	LatestVersion    int64
}

type GimmickOrnamentProgressState struct {
	Key              GimmickOrnamentKey
	ProgressValueBit int32
	BaseDatetime     int64
	LatestVersion    int64
}

type GimmickSequenceState struct {
	Key                      GimmickSequenceKey
	IsGimmickSequenceCleared bool
	ClearDatetime            int64
	LatestVersion            int64
}

type GimmickUnlockState struct {
	Key           GimmickKey
	IsUnlocked    bool
	LatestVersion int64
}

type CageOrnamentRewardState struct {
	CageOrnamentId      int32
	AcquisitionDatetime int64
	LatestVersion       int64
}

type CageOrnamentAccessState struct {
	CageOrnamentId       int32
	FirstAccessDatetime  int64
	LatestAccessDatetime int64
	LatestVersion        int64
}

type TowerAccumulationRewardState struct {
	EventQuestChapterId                       int32
	LatestRewardReceiveQuestMissionClearCount int32
	LatestVersion                             int64
}

type LabyrinthSeasonState struct {
	EventQuestChapterId                  int32
	LastJoinSeasonNumber                 int32
	LastSeasonRewardReceivedSeasonNumber int32
	LatestVersion                        int64
}

// LabyrinthStageKey is the composite key for UserState.LabyrinthStages.
type LabyrinthStageKey struct {
	EventQuestChapterId int32
	StageOrder          int32
}

type LabyrinthStageState struct {
	EventQuestChapterId                         int32
	StageOrder                                  int32
	IsReceivedStageClearReward                  bool
	AccumulationRewardReceivedQuestMissionCount int32
	LatestVersion                               int64
}

type PartsState struct {
	UserPartsUuid       string
	PartsId             int32
	Level               int32
	PartsStatusMainId   int32
	IsProtected         bool
	AcquisitionDatetime int64
	LatestVersion       int64
}

type PartsGroupNoteState struct {
	PartsGroupId             int32
	FirstAcquisitionDatetime int64
	LatestVersion            int64
}

type PartsPresetState struct {
	UserPartsPresetNumber    int32
	UserPartsUuid01          string
	UserPartsUuid02          string
	UserPartsUuid03          string
	Name                     string
	UserPartsPresetTagNumber int32
	LatestVersion            int64
}

type PartsPresetTagState struct {
	UserPartsPresetTagNumber int32
	Name                     string
	LatestVersion            int64
}

type PartsStatusSubKey struct {
	UserPartsUuid string
	StatusIndex   int32
}

type PartsStatusSubState struct {
	UserPartsUuid           string
	StatusIndex             int32
	PartsStatusSubLotteryId int32
	Level                   int32
	StatusKindType          int32
	StatusCalculationType   int32
	StatusChangeValue       int32
	LatestVersion           int64
}

type NotificationState struct {
	GiftNotReceiveCount       int32
	FriendRequestReceiveCount int32
	IsExistUnreadInformation  bool
}

type FriendState struct {
	IsFriend                bool
	CheerSentDatetime       int64
	CheerReceivedDatetime   int64
	StaminaReceivedDatetime int64
}

type GiftState struct {
	NotReceived []NotReceivedGiftState
	Received    []ReceivedGiftState
}

type GiftCommonState struct {
	PossessionType        int32
	PossessionId          int32
	Count                 int32
	GrantDatetime         int64
	DescriptionGiftTextId int32
	EquipmentData         []byte
}

type NotReceivedGiftState struct {
	GiftCommon         GiftCommonState
	ExpirationDatetime int64
	UserGiftUuid       string
}

type ReceivedGiftState struct {
	GiftCommon       GiftCommonState
	ReceivedDatetime int64
}

type GachaState struct {
	RewardAvailable        bool
	TodaysCurrentDrawCount int32
	DailyMaxCount          int32
	LastRewardDrawDate     int64
	ConvertedGachaMedal    ConvertedGachaMedalState
	BannerStates           map[int32]GachaBannerState
}

type ConvertedGachaMedalState struct {
	ConvertedMedalPossession []ConsumableItemState
	ObtainPossession         *ConsumableItemState
}

type ConsumableItemState struct {
	ConsumableItemId int32
	Count            int32
}

type UserShopItemState struct {
	ShopItemId                       int32
	BoughtCount                      int32
	LatestBoughtCountChangedDatetime int64
	LatestVersion                    int64
}

type UserShopReplaceableState struct {
	LineupUpdateCount          int32
	LatestLineupUpdateDatetime int64
	LatestVersion              int64
}

type UserShopReplaceableLineupState struct {
	SlotNumber    int32
	ShopItemId    int32
	LatestVersion int64
}

type GachaPricePhaseEntry struct {
	PhaseId        int32
	PriceType      int32
	PriceId        int32
	Price          int32
	RegularPrice   int32
	DrawCount      int32
	FixedRarityMin int32
	FixedCount     int32
	LimitExecCount int32
	StepNumber     int32
	Bonuses        []GachaBonusEntry
}

type GachaBonusEntry struct {
	PossessionType int32
	PossessionId   int32
	Count          int32
}

type GachaPromotionItem struct {
	PossessionType      int32
	PossessionId        int32
	IsTarget            bool
	BonusPossessionType int32
	BonusPossessionId   int32
}

type GachaBannerState struct {
	GachaId       int32
	MedalCount    int32
	StepNumber    int32
	LoopCount     int32
	DrawCount     int32
	BoxDrewCounts map[int32]int32
	BoxNumber     int32
}

type GachaCatalogEntry struct {
	GachaId                    int32
	GachaLabelType             int32
	GachaModeType              int32
	GachaAutoResetType         int32
	GachaAutoResetPeriod       int32
	NextAutoResetDatetime      int64
	IsUserGachaUnlock          bool
	StartDatetime              int64
	EndDatetime                int64
	RelatedMainQuestChapterId  int32
	RelatedEventQuestChapterId int32
	PromotionMovieAssetId      int32
	GachaMedalId               int32
	MedalConsumableItemId      int32
	GachaDecorationType        int32
	SortOrder                  int32
	IsInactive                 bool
	UnlockConditions           []GachaUnlockConditionEntry
	InformationId              int32
	BannerAssetName            string
	GroupId                    int32
	CeilingCount               int32
	PricePhases                []GachaPricePhaseEntry
	BoxItems                   []GachaBoxItemEntry
	PromotionItems             []GachaPromotionItem
	DescriptionTextId          int32
	MaxStepNumber              int32
}

type GachaBoxItemEntry struct {
	PossessionType int32
	PossessionId   int32
	RarityType     int32
	Count          int32
	MaxCount       int32
}

type GachaUnlockConditionEntry struct {
	GachaUnlockConditionType int32
	ConditionValue           int32
}

type CharacterBoardState struct {
	CharacterBoardId int32
	PanelReleaseBit1 int32
	PanelReleaseBit2 int32
	PanelReleaseBit3 int32
	PanelReleaseBit4 int32
	LatestVersion    int64
}

type CharacterBoardAbilityKey struct {
	CharacterId int32
	AbilityId   int32
}

func (k CharacterBoardAbilityKey) MarshalText() ([]byte, error) {
	return marshalKey(int64(k.CharacterId), int64(k.AbilityId)), nil
}

func (k *CharacterBoardAbilityKey) UnmarshalText(text []byte) error {
	v, err := unmarshalKey(text, "CharacterBoardAbilityKey", 2)
	if err != nil {
		return err
	}
	k.CharacterId = int32(v[0])
	k.AbilityId = int32(v[1])
	return nil
}

type CharacterBoardAbilityState struct {
	CharacterId   int32
	AbilityId     int32
	Level         int32
	LatestVersion int64
}

type CharacterBoardStatusUpKey struct {
	CharacterId           int32
	StatusCalculationType int32
}

func (k CharacterBoardStatusUpKey) MarshalText() ([]byte, error) {
	return marshalKey(int64(k.CharacterId), int64(k.StatusCalculationType)), nil
}

func (k *CharacterBoardStatusUpKey) UnmarshalText(text []byte) error {
	v, err := unmarshalKey(text, "CharacterBoardStatusUpKey", 2)
	if err != nil {
		return err
	}
	k.CharacterId = int32(v[0])
	k.StatusCalculationType = int32(v[1])
	return nil
}

type CharacterBoardStatusUpState struct {
	CharacterId           int32
	StatusCalculationType int32
	Hp                    int32
	Attack                int32
	Vitality              int32
	Agility               int32
	CriticalRatio         int32
	CriticalAttack        int32
	LatestVersion         int64
}

type CostumeAwakenStatusKey struct {
	UserCostumeUuid       string
	StatusCalculationType model.StatusCalculationType
}

func (k CostumeAwakenStatusKey) MarshalText() ([]byte, error) {
	return fmt.Appendf(nil, "%s:%d", k.UserCostumeUuid, k.StatusCalculationType), nil
}

func (k *CostumeAwakenStatusKey) UnmarshalText(text []byte) error {
	s := string(text)
	idx := strings.LastIndex(s, ":")
	if idx < 0 {
		return fmt.Errorf("invalid CostumeAwakenStatusKey: %s", text)
	}
	k.UserCostumeUuid = s[:idx]
	v, err := strconv.ParseInt(s[idx+1:], 10, 32)
	if err != nil {
		return err
	}
	k.StatusCalculationType = model.StatusCalculationType(v)
	return nil
}

type CostumeAwakenStatusUpState struct {
	UserCostumeUuid       string
	StatusCalculationType model.StatusCalculationType
	Hp                    int32
	Attack                int32
	Vitality              int32
	Agility               int32
	CriticalRatio         int32
	CriticalAttack        int32
	LatestVersion         int64
}

type CostumeLevelBonusReleaseStatusState struct {
	CostumeId              int32
	LastReleasedBonusLevel int32
	ConfirmedBonusLevel    int32
	LatestVersion          int64
}

type CostumeLotteryEffectAbilityState struct {
	UserCostumeUuid string
	SlotNumber      int32
	AbilityId       int32
	AbilityLevel    int32
	LatestVersion   int64
}

type CostumeLotteryEffectStatusKey struct {
	UserCostumeUuid       string
	StatusCalculationType model.StatusCalculationType
}

type CostumeLotteryEffectStatusUpState struct {
	UserCostumeUuid       string
	StatusCalculationType model.StatusCalculationType
	Hp                    int32
	Attack                int32
	Vitality              int32
	Agility               int32
	CriticalRatio         int32
	CriticalAttack        int32
	LatestVersion         int64
}

type AutoSaleSettingState struct {
	PossessionAutoSaleItemType  int32
	PossessionAutoSaleItemValue string
}

type CharacterRebirthState struct {
	CharacterId   int32
	RebirthCount  int32
	LatestVersion int64
}

type CostumeLotteryEffectKey struct {
	UserCostumeUuid string
	SlotNumber      int32
}

func (k CostumeLotteryEffectKey) MarshalText() ([]byte, error) {
	return fmt.Appendf(nil, "%s:%d", k.UserCostumeUuid, k.SlotNumber), nil
}

func (k *CostumeLotteryEffectKey) UnmarshalText(text []byte) error {
	s := string(text)
	idx := strings.LastIndex(s, ":")
	if idx < 0 {
		return fmt.Errorf("invalid CostumeLotteryEffectKey: %s", text)
	}
	k.UserCostumeUuid = s[:idx]
	v, err := strconv.ParseInt(s[idx+1:], 10, 32)
	if err != nil {
		return err
	}
	k.SlotNumber = int32(v)
	return nil
}

type CostumeLotteryEffectState struct {
	UserCostumeUuid string
	SlotNumber      int32
	OddsNumber      int32
	LatestVersion   int64
}

type CostumeLotteryEffectPendingState struct {
	UserCostumeUuid string
	SlotNumber      int32
	OddsNumber      int32
	LatestVersion   int64
}
