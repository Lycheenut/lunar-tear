package service

var gachaWebText = map[string]map[string]string{
	"en": {
		"title": "Summon Details", "rates": "Drop Rates", "rarity": "Rarity", "reward": "Reward", "weapon": "Weapon",
		"characterWeapon": "With character", "weaponOnly": "Weapon only", "groups": "Rates by rarity", "items": "Available weapons",
		"slot": "Draw %d", "slots": "Draws %d–%d", "draws": "%d draw(s)", "ticket": "Ticket", "phase": "Summon option",
		"search": "Search rewards by name or ID", "refresh": "Refresh", "box": "Current Box", "remaining": "Remaining / total",
		"quantity": "Per draw", "probability": "Next draw", "unlimited": "Unlimited", "soldOut": "Exhausted", "pickup": "PICKUP",
		"updated": "Updated", "reset": "Next monthly reset", "gems": "Gems", "empty": "No matching rewards.",
		"premiumNote": "Each column shows the chance for one draw in those positions. Rates include the guarantee for the selected summon option. A character and its paired weapon are one result.",
		"boxNote":     "Rates apply to the next draw and change as limited rewards are taken. Remaining counts are the number of draws available, not the number of items received.",
		"chapterNote": "Rates reflect the current limited and unlimited reward pools. When a pool is exhausted, the remaining eligible pool receives 100%. Monthly limits reset at 00:00 UTC−8 on the first day of each month.",
		"rounding":    "Percentages are rounded to six decimal places; displayed totals may differ slightly from 100%.",
		"session":     "Your session has expired or is missing. Return to the game and reopen these details.",
		"invalid":     "This summon link is invalid. Please reopen it from the game.", "missing": "This summon is no longer available.",
		"unavailable": "Summon details are temporarily unavailable. Please try again.",
	},
	"ja": {
		"title": "ガチャ詳細", "rates": "提供割合", "rarity": "レアリティ", "reward": "報酬", "weapon": "武器",
		"characterWeapon": "キャラ付き武器", "weaponOnly": "武器のみ", "groups": "レアリティ別 提供割合", "items": "出現する武器",
		"slot": "%d枠目", "slots": "%d〜%d枠目", "draws": "%d回ガチャ", "ticket": "チケット", "phase": "ガチャの種類",
		"search": "報酬名・IDで検索", "refresh": "更新", "box": "現在のBox", "remaining": "残り / 総数",
		"quantity": "1回の獲得数", "probability": "次の1回の確率", "unlimited": "無制限", "soldOut": "獲得済み", "pickup": "PICKUP",
		"updated": "更新日時", "reset": "次回リセット", "gems": "ジェム", "empty": "該当する報酬はありません。",
		"premiumNote": "各列は該当する枠の1回あたりの確率です。選択したガチャの確定枠を反映しています。キャラと付属武器は1組の抽選結果です。",
		"boxNote":     "次の1回の抽選確率を表示しています。残りの報酬に応じて確率は変化します。残数は獲得できる回数で、アイテムの個数ではありません。",
		"chapterNote": "数量限定・無制限の各報酬の現在の提供割合を表示しています。一方の報酬がなくなると、抽選対象の残りの報酬が100%になります。月間の残数は毎月1日 00:00（UTC−8）にリセットされます。",
		"rounding":    "確率は小数点以下6桁に丸めて表示しているため、合計が100%にならない場合があります。",
		"session":     "セッションを確認できません。ゲームに戻り、詳細画面を開き直してください。",
		"invalid":     "ガチャの指定が正しくありません。ゲームから開き直してください。", "missing": "このガチャは現在利用できません。",
		"unavailable": "ガチャ詳細を取得できませんでした。時間をおいて再度お試しください。",
	},
}
