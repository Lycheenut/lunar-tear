package campaign

const millisecondsPerDay = int64(24 * 60 * 60 * 1000)

type beginnerRow struct {
	campaignId       int32
	judgeStartMillis int64
	judgeEndMillis   int64
	grantDays        int32
	unlockQuestId    int32
}

type comebackRow struct {
	campaignId       int32
	judgeStartMillis int64
	judgeEndMillis   int64
	judgeDays        int32
	grantDays        int32
	unlockQuestId    int32
	gradeGroupId     int32
}

type BeginnerEnrollment struct {
	CampaignId               int32
	CampaignRegisterDatetime int64
}

type ComebackEnrollment struct {
	CampaignId       int32
	ComebackDatetime int64
	GradeGroupId     int32
}

type UserStatusContext struct {
	NowMillis                        int64
	RegisterDatetime                 int64
	LastComebackLoginDatetime        int64
	BeginnerCampaignId               int32
	BeginnerCampaignRegisterDatetime int64
	ComebackCampaignId               int32
	ComebackDatetime                 int64
	IsCampaignUnlockQuestCleared     func(int32) bool
}

func (c *Catalog) FilterForUser(ctx UserStatusContext) Filter {
	return Filter{NowMillis: ctx.NowMillis, UserStatus: c.userStatus(ctx)}
}

func (c *Catalog) IsComebackLogin(nowMillis, lastLoginMillis int64, isUnlockQuestCleared func(int32) bool) bool {
	_, ok := c.ComebackEnrollmentForLogin(nowMillis, lastLoginMillis, isUnlockQuestCleared)
	return ok
}

func (c *Catalog) BeginnerEnrollmentForRegistration(registerMillis int64) (BeginnerEnrollment, bool) {
	var selected *beginnerRow
	for i := range c.beginner {
		r := &c.beginner[i]
		if registerMillis < r.judgeStartMillis || registerMillis > r.judgeEndMillis {
			continue
		}
		if selected == nil || r.judgeStartMillis > selected.judgeStartMillis ||
			r.judgeStartMillis == selected.judgeStartMillis && r.campaignId > selected.campaignId {
			selected = r
		}
	}
	if selected == nil {
		return BeginnerEnrollment{}, false
	}
	return BeginnerEnrollment{CampaignId: selected.campaignId, CampaignRegisterDatetime: registerMillis}, true
}

func (c *Catalog) ComebackEnrollmentForLogin(nowMillis, lastLoginMillis int64, isUnlockQuestCleared func(int32) bool) (ComebackEnrollment, bool) {
	if lastLoginMillis <= 0 || nowMillis <= lastLoginMillis {
		return ComebackEnrollment{}, false
	}
	var selected *comebackRow
	for i := range c.comeback {
		r := &c.comeback[i]
		if nowMillis < r.judgeStartMillis || nowMillis > r.judgeEndMillis || !questUnlocked(r.unlockQuestId, isUnlockQuestCleared) {
			continue
		}
		if nowMillis-lastLoginMillis < int64(r.judgeDays)*millisecondsPerDay {
			continue
		}
		if selected == nil || r.judgeDays > selected.judgeDays ||
			r.judgeDays == selected.judgeDays && r.judgeStartMillis > selected.judgeStartMillis ||
			r.judgeDays == selected.judgeDays && r.judgeStartMillis == selected.judgeStartMillis && r.campaignId > selected.campaignId {
			selected = r
		}
	}
	if selected == nil {
		return ComebackEnrollment{}, false
	}
	return ComebackEnrollment{
		CampaignId:       selected.campaignId,
		ComebackDatetime: nowMillis,
		GradeGroupId:     selected.gradeGroupId,
	}, true
}

// ComebackEnrollmentForRecordedLogin backfills records created before per-user
// comeback campaign persistence existed. The old logic selected the first
// eligible baseline campaign, so prefer the least restrictive overlapping row.
func (c *Catalog) ComebackEnrollmentForRecordedLogin(comebackMillis int64) (ComebackEnrollment, bool) {
	var selected *comebackRow
	for i := range c.comeback {
		r := &c.comeback[i]
		if comebackMillis < r.judgeStartMillis || comebackMillis > r.judgeEndMillis {
			continue
		}
		if selected == nil || r.judgeDays < selected.judgeDays ||
			r.judgeDays == selected.judgeDays && r.judgeStartMillis > selected.judgeStartMillis ||
			r.judgeDays == selected.judgeDays && r.judgeStartMillis == selected.judgeStartMillis && r.campaignId > selected.campaignId {
			selected = r
		}
	}
	if selected == nil {
		return ComebackEnrollment{}, false
	}
	return ComebackEnrollment{
		CampaignId:       selected.campaignId,
		ComebackDatetime: comebackMillis,
		GradeGroupId:     selected.gradeGroupId,
	}, true
}

func (c *Catalog) userStatus(ctx UserStatusContext) TargetUserStatusType {
	if ctx.BeginnerCampaignId != 0 && c.IsBeginnerEnrollmentActive(
		ctx.BeginnerCampaignId, ctx.BeginnerCampaignRegisterDatetime, ctx.NowMillis, ctx.IsCampaignUnlockQuestCleared,
	) {
		return TargetUserStatusBeginner
	}
	for _, r := range c.beginner {
		if ctx.RegisterDatetime < r.judgeStartMillis || ctx.RegisterDatetime > r.judgeEndMillis {
			continue
		}
		if withinGrantTerm(ctx.RegisterDatetime, ctx.NowMillis, r.grantDays) && questUnlocked(r.unlockQuestId, ctx.IsCampaignUnlockQuestCleared) {
			return TargetUserStatusBeginner
		}
	}
	if ctx.ComebackCampaignId != 0 && c.IsComebackEnrollmentActive(
		ctx.ComebackCampaignId, ctx.ComebackDatetime, ctx.NowMillis, ctx.IsCampaignUnlockQuestCleared,
	) {
		return TargetUserStatusComeback
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

func (c *Catalog) IsBeginnerEnrollmentActive(campaignId int32, registerMillis, nowMillis int64, isUnlockQuestCleared func(int32) bool) bool {
	for _, r := range c.beginner {
		if r.campaignId == campaignId {
			return withinGrantTerm(registerMillis, nowMillis, r.grantDays) && questUnlocked(r.unlockQuestId, isUnlockQuestCleared)
		}
	}
	return false
}

func (c *Catalog) IsComebackEnrollmentActive(campaignId int32, comebackMillis, nowMillis int64, isUnlockQuestCleared func(int32) bool) bool {
	for _, r := range c.comeback {
		if r.campaignId == campaignId {
			return withinGrantTerm(comebackMillis, nowMillis, r.grantDays) && questUnlocked(r.unlockQuestId, isUnlockQuestCleared)
		}
	}
	return false
}

func (c *Catalog) IsComebackGradeGroupActive(campaignId int32, comebackMillis, nowMillis int64, gradeGroupId int32, isUnlockQuestCleared func(int32) bool) bool {
	for _, r := range c.comeback {
		if r.campaignId == campaignId {
			return r.gradeGroupId == gradeGroupId && withinGrantTerm(comebackMillis, nowMillis, r.grantDays) && questUnlocked(r.unlockQuestId, isUnlockQuestCleared)
		}
	}
	return false
}

func withinGrantTerm(startMillis, nowMillis int64, grantDays int32) bool {
	return startMillis > 0 && grantDays > 0 && nowMillis >= startMillis && nowMillis < startMillis+int64(grantDays)*millisecondsPerDay
}

func questUnlocked(questId int32, isCleared func(int32) bool) bool {
	return questId == 0 || isCleared != nil && isCleared(questId)
}
