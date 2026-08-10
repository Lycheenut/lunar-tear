package campaign

const millisecondsPerDay = int64(24 * 60 * 60 * 1000)

type beginnerRow struct {
	judgeStartMillis int64
	judgeEndMillis   int64
	grantDays        int32
	unlockQuestId    int32
}

type comebackRow struct {
	judgeStartMillis int64
	judgeEndMillis   int64
	judgeDays        int32
	grantDays        int32
	unlockQuestId    int32
}

type UserStatusContext struct {
	NowMillis                    int64
	RegisterDatetime             int64
	LastComebackLoginDatetime    int64
	IsCampaignUnlockQuestCleared func(int32) bool
}

func (c *Catalog) FilterForUser(ctx UserStatusContext) Filter {
	return Filter{NowMillis: ctx.NowMillis, UserStatus: c.userStatus(ctx)}
}

func (c *Catalog) IsComebackLogin(nowMillis, lastLoginMillis int64, isUnlockQuestCleared func(int32) bool) bool {
	if lastLoginMillis <= 0 || nowMillis <= lastLoginMillis {
		return false
	}
	for _, r := range c.comeback {
		if nowMillis < r.judgeStartMillis || nowMillis > r.judgeEndMillis || !questUnlocked(r.unlockQuestId, isUnlockQuestCleared) {
			continue
		}
		if nowMillis-lastLoginMillis >= int64(r.judgeDays)*millisecondsPerDay {
			return true
		}
	}
	return false
}

func (c *Catalog) userStatus(ctx UserStatusContext) TargetUserStatusType {
	for _, r := range c.beginner {
		if ctx.RegisterDatetime < r.judgeStartMillis || ctx.RegisterDatetime > r.judgeEndMillis {
			continue
		}
		if withinGrantTerm(ctx.RegisterDatetime, ctx.NowMillis, r.grantDays) && questUnlocked(r.unlockQuestId, ctx.IsCampaignUnlockQuestCleared) {
			return TargetUserStatusBeginner
		}
	}
	for _, r := range c.comeback {
		if ctx.LastComebackLoginDatetime < r.judgeStartMillis || ctx.LastComebackLoginDatetime > r.judgeEndMillis {
			continue
		}
		if withinGrantTerm(ctx.LastComebackLoginDatetime, ctx.NowMillis, r.grantDays) && questUnlocked(r.unlockQuestId, ctx.IsCampaignUnlockQuestCleared) {
			return TargetUserStatusComeback
		}
	}
	return TargetUserStatusAll
}

func withinGrantTerm(startMillis, nowMillis int64, grantDays int32) bool {
	return startMillis > 0 && grantDays > 0 && nowMillis >= startMillis && nowMillis < startMillis+int64(grantDays)*millisecondsPerDay
}

func questUnlocked(questId int32, isCleared func(int32) bool) bool {
	return questId == 0 || isCleared != nil && isCleared(questId)
}
