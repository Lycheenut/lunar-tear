package store

import (
	"fmt"
	"log"

	"lunar-tear/server/internal/model"
)

const StaminaRecoveryDivisor int64 = 180

func SettleStamina(user *UserState, maxStaminaMillis int32, nowMillis int64) {
	stored := int64(user.Status.StaminaMilliValue)
	maxMilli := int64(maxStaminaMillis)
	if stored >= maxMilli {
		return
	}
	elapsed := nowMillis - user.Status.StaminaUpdateDatetime
	if elapsed <= 0 {
		return
	}
	regen := elapsed / StaminaRecoveryDivisor
	settled := min(stored+regen, maxMilli)
	user.Status.StaminaMilliValue = int32(settled)
	user.Status.StaminaUpdateDatetime = nowMillis
}

func ConsumeStamina(user *UserState, costUnits int32, maxStaminaMillis int32, nowMillis int64) error {
	if costUnits < 0 || int64(costUnits)*1000 > int64(^uint32(0)>>1) {
		return fmt.Errorf("invalid stamina cost %d", costUnits)
	}
	SettleStamina(user, maxStaminaMillis, nowMillis)
	costMillis := costUnits * 1000
	if costMillis > user.Status.StaminaMilliValue {
		return fmt.Errorf("insufficient stamina: cost=%d available=%d", costUnits, user.Status.StaminaMilliValue)
	}
	user.Status.StaminaMilliValue -= costMillis
	user.Status.StaminaUpdateDatetime = nowMillis
	log.Printf("[ConsumeStamina] cost=%d -> remaining=%d", costUnits, user.Status.StaminaMilliValue)
	return nil
}

func HasEnoughStamina(user *UserState, costUnits int32, maxStaminaMillis int32, nowMillis int64) bool {
	if costUnits < 0 {
		return false
	}
	costMillis := int64(costUnits) * 1000
	if costMillis > int64(^uint32(0)>>1) {
		return false
	}
	available := settledStaminaMilliValue(user, maxStaminaMillis, nowMillis)
	return int64(available) >= costMillis
}

func settledStaminaMilliValue(user *UserState, maxStaminaMillis int32, nowMillis int64) int32 {
	stored := int64(user.Status.StaminaMilliValue)
	if stored >= int64(maxStaminaMillis) {
		return user.Status.StaminaMilliValue
	}
	elapsed := nowMillis - user.Status.StaminaUpdateDatetime
	if elapsed <= 0 {
		return user.Status.StaminaMilliValue
	}
	return int32(min(stored+elapsed/StaminaRecoveryDivisor, int64(maxStaminaMillis)))
}

func RecoverStamina(user *UserState, millis int32, maxStaminaMillis int32, nowMillis int64) {
	SettleStamina(user, maxStaminaMillis, nowMillis)
	recovered := int64(user.Status.StaminaMilliValue) + int64(millis)
	if recovered > int64(^uint32(0)>>1) {
		recovered = int64(^uint32(0) >> 1)
	}
	user.Status.StaminaMilliValue = int32(recovered)
	user.Status.StaminaUpdateDatetime = nowMillis
	log.Printf("[RecoverStamina] +%d -> total=%d", millis, user.Status.StaminaMilliValue)
}

func ResolveStaminaEffectMillis(effectValueType, effectValue, maxStaminaMillis int32) int32 {
	var resolved int64
	switch effectValueType {
	case model.EffectValueFixed:
		resolved = int64(effectValue) * 1000
	case model.EffectValuePermil:
		resolved = int64(effectValue) * int64(maxStaminaMillis) / 1000
	default:
		return 0
	}
	if resolved <= 0 || resolved > int64(^uint32(0)>>1) {
		return 0
	}
	return int32(resolved)
}
