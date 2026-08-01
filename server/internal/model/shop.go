package model

const (
	ShopGroupTypeUnknown      int32 = 0
	ShopGroupTypePremiumShop  int32 = 1
	ShopGroupTypeItemShop     int32 = 3
	ShopGroupTypeExchangeShop int32 = 4
	ShopGroupTypeRecoveryShop int32 = 5
)

const (
	ShopItemAutoResetNone    int32 = 1
	ShopItemAutoResetWeekly  int32 = 2
	ShopItemAutoResetMonthly int32 = 3
)
