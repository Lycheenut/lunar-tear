package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"lunar-tear/server/internal/assettext"
	"lunar-tear/server/internal/database"
	"lunar-tear/server/internal/gacha"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/store/sqlite"
	"lunar-tear/server/migrations"
)

func newGachaWebTestServer(t *testing.T) (*GachaWebHandler, *sqlite.SQLiteStore, store.SessionState, *runtime.Catalogs) {
	t.Helper()
	db, err := database.Open(filepath.Join(t.TempDir(), "game.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	if err := migrations.Up(context.Background(), db); err != nil {
		t.Fatal(err)
	}
	repo := sqlite.New(db, nil)
	if _, err := repo.CreateUser("gacha-web-test", model.ClientPlatform{}); err != nil {
		t.Fatal(err)
	}
	session, err := repo.CreateSession("gacha-web-test", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	cat := &runtime.Catalogs{
		GachaEntries: []store.GachaCatalogEntry{
			{GachaId: 100, GachaLabelType: model.GachaLabelPremium, GachaModeType: model.GachaModeBasic, IsUserGachaUnlock: true, BannerAssetName: "<script>banner</script>", PricePhases: []store.GachaPricePhaseEntry{{PhaseId: 1, DrawCount: 1}, {PhaseId: 2, DrawCount: 10, FixedCount: 1, FixedRarityMin: 30}}},
			{GachaId: 201, GachaLabelType: model.GachaLabelEvent, GachaModeType: model.GachaModeBox, IsUserGachaUnlock: true, PricePhases: []store.GachaPricePhaseEntry{{PhaseId: 3, DrawCount: 1}}, BoxItems: []store.GachaBoxItemEntry{
				{CounterId: 10, PossessionType: 5, PossessionId: 1, Count: 5, MaxCount: 3},
				{CounterId: 20, PossessionType: 5, PossessionId: 2, Count: 1, MaxCount: 1},
			}},
		},
		GachaHandler: &gacha.GachaHandler{Granter: &store.PossessionGranter{}, Premium: &gacha.PremiumCatalog{Banners: map[int32]*gacha.PremiumBannerPool{100: {Groups: []gacha.PremiumGroup{
			{Star: 3, Rarity: 30, Weight: 2000, GrantType: gacha.GrantWeaponOnly, NonPickup: []gacha.PoolItem{{WeaponId: 100002, RarityType: 30}}},
			{Star: 2, Rarity: 20, Weight: 8000, GrantType: gacha.GrantWeaponOnly, NonPickup: []gacha.PoolItem{{WeaponId: 100001, RarityType: 20}}},
		}}}}},
		Weapon: &masterdata.WeaponCatalog{Weapons: map[int32]masterdata.EntityMWeapon{100001: {WeaponType: 1, AssetVariationId: 60}, 100002: {WeaponType: 1, AssetVariationId: 60}}},
	}
	web := &GachaWebHandler{users: repo, sessions: repo, catalogs: func() *runtime.Catalogs { return cat }, names: assettext.Index{
		"en": {"weapon.name.wp001060.1": "Test weapon"},
		"ja": {"weapon.name.wp001060.1": "テスト武器"},
	}}
	return web, repo, session, cat
}

func requestGachaPage(web http.Handler, path string) *httptest.ResponseRecorder {
	recorder := httptest.NewRecorder()
	web.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
	return recorder
}

func TestGachaWebPremiumRatesAndRequestValidation(t *testing.T) {
	web, repo, session, cat := newGachaWebTestServer(t)
	auth := "&sessionKey=" + url.QueryEscape(session.SessionKey)
	for _, path := range []string{"/gacha-rate", "/web/en/gacha-rate", "/web/ja/gacha-details"} {
		response := requestGachaPage(web, path+"?gachaId=100&tab=Rate"+auth)
		body := response.Body.String()
		if response.Code != 200 || !strings.Contains(body, "80.000000%") || !strings.Contains(body, "100.000000%") || !strings.Contains(body, "0.000000%") {
			t.Fatalf("response %d: %s", response.Code, body)
		}
		if strings.Contains(body, "<script>banner</script>") || !strings.Contains(body, "&lt;script&gt;") {
			t.Fatal("title was not escaped")
		}
		language := "en"
		if strings.Contains(path, "/ja/") {
			language = "ja"
		}
		if !strings.Contains(body, web.names[language]["weapon.name.wp001060.1"]) {
			t.Fatal("weapon names missing")
		}
		if response.Header().Get("Cache-Control") != "private, no-store" || response.Header().Get("Referrer-Policy") != "no-referrer" {
			t.Fatal("private WebView caching headers missing")
		}
	}
	// A new catalog snapshot must be used by the next request.
	cat.GachaHandler.Premium.Banners[100].Groups[0].Weight = 1000
	cat.GachaHandler.Premium.Banners[100].Groups[1].Weight = 9000
	if body := requestGachaPage(web, "/gacha-rate?gachaId=100"+auth).Body.String(); !strings.Contains(body, "90.000000%") {
		t.Fatal("stale premium rates")
	}
	for _, tc := range []struct {
		query  string
		status int
	}{
		{"?gachaId=100", 401}, {"?gachaId=100&sessionKey=wrong", 401},
		{"?gachaId=0" + auth, 400}, {"?gachaId=abc" + auth, 400}, {"?gachaId=2147483648" + auth, 400},
		{"?gachaId=100&gachaId=201" + auth, 400}, {"?gachaId=100&nowPlayingId=201" + auth, 400},
		{"?gachaId=999" + auth, 404}, {"?gachaId=100&gachaPricePhaseId=999" + auth, 400},
	} {
		if got := requestGachaPage(web, "/web/en/gacha-rate"+tc.query).Code; got != tc.status {
			t.Fatalf("query %s: status %d, want %d", strings.ReplaceAll(tc.query, auth, "&sessionKey=REDACTED"), got, tc.status)
		}
	}
	expired, err := repo.CreateSession("gacha-web-test", -time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if got := requestGachaPage(web, "/gacha-rate?gachaId=100&sessionKey="+url.QueryEscape(expired.SessionKey)).Code; got != 401 {
		t.Fatalf("expired session status = %d", got)
	}
	recorder := httptest.NewRecorder()
	web.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/gacha-rate?gachaId=100"+auth, nil))
	if recorder.Code != 405 {
		t.Fatalf("POST status = %d", recorder.Code)
	}
}

func TestGachaWebBoxUsesCurrentIDAndLiveDrawState(t *testing.T) {
	web, repo, session, cat := newGachaWebTestServer(t)
	path := "/web/en/gacha/box?gachaId=100&gachaId=201&nowPlayingId=201&userId=999999&sessionKey=" + url.QueryEscape(session.SessionKey)
	body := requestGachaPage(web, path).Body.String()
	if !strings.Contains(body, "75.000000%") || !strings.Contains(body, "3 / 3") || !strings.Contains(body, "×5") {
		t.Fatal("incorrect initial box quantities/rates")
	}
	var drawErr error
	user, err := repo.UpdateUser(session.UserId, func(user *store.UserState) {
		_, drawErr = cat.GachaHandler.HandleDraw(user, cat.GachaEntries[1], 3, 1)
	})
	if err != nil || drawErr != nil {
		t.Fatalf("draw: %v, %v", err, drawErr)
	}
	body = requestGachaPage(web, path).Body.String()
	for _, odds := range gacha.BoxOdds(cat.GachaEntries[1], user.Gacha.BannerStates[201], gametime.NowMillis()) {
		if !strings.Contains(body, fmt.Sprintf("%.6f%%", odds.Rate*100)) || !strings.Contains(body, fmt.Sprintf("%d / %d", odds.Remaining, odds.MaxCount)) {
			t.Fatal("page did not reflect actual draw")
		}
	}
	_, err = repo.UpdateUser(session.UserId, func(user *store.UserState) { drawErr = cat.GachaHandler.HandleResetBox(user, cat.GachaEntries[1]) })
	if err != nil || drawErr != nil {
		t.Fatalf("reset: %v, %v", err, drawErr)
	}
	body = requestGachaPage(web, path).Body.String()
	if !strings.Contains(body, "<span>2</span>") || !strings.Contains(body, "75.000000%") {
		t.Fatal("reset box was not refreshed")
	}
	// Chapter Gacha displays unlimited rewards and applies monthly reset without a draw.
	cat.GachaEntries[1].GachaLabelType = model.GachaLabelChapter
	cat.GachaEntries[1].BoxItems[1].MaxCount = 0
	cat.GachaEntries[1].BoxItems[1].Weight = 1
	_, err = repo.UpdateUser(session.UserId, func(user *store.UserState) {
		user.Gacha.BannerStates[201] = store.GachaBannerState{BoxDrewCounts: map[int32]int32{10: 3, model.ChapterGachaMonthCounterId: 202001}}
	})
	if err != nil {
		t.Fatal(err)
	}
	body = requestGachaPage(web, path).Body.String()
	if !strings.Contains(body, "80.000000%") || !strings.Contains(body, "3 / 3") || !strings.Contains(body, ">∞</td>") {
		t.Fatal("chapter monthly reset/unlimited rates missing")
	}
}

func TestGachaWebCDNProxyPreservesClientQuery(t *testing.T) {
	var received string
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received = r.URL.RequestURI()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte("gacha-page"))
	}))
	defer backend.Close()
	cdn := NewOctoHTTPServer("", t.TempDir())
	if err := cdn.SetGachaWebBackend(backend.URL); err != nil {
		t.Fatal(err)
	}
	path := "/web/ja/gacha/box?gachaId=1&gachaId=2&nowPlayingId=2&sessionKey=test&serverAddress=http%3A%2F%2Finvalid"
	response := requestGachaPage(cdn.Handler(), path)
	if received != path || response.Code != 200 || response.Body.String() != "gacha-page" {
		t.Fatalf("proxy failed: %d %s", response.Code, received)
	}
	if err := cdn.SetGachaWebBackend("file:///tmp/gacha"); err == nil {
		t.Fatal("invalid backend accepted")
	}
}

func TestGachaWebNamesCoverCurrentPools(t *testing.T) {
	cat := newGachaResponseTestHolder(t).Get()
	names, err := assettext.Load(filepath.Join("..", "..", "assets", "release", "20240404193219.bin.e"))
	if err != nil {
		t.Fatal(err)
	}
	web := &GachaWebHandler{names: names}
	for _, language := range []string{"en", "ja"} {
		check := func(possessionType, id int32) {
			t.Helper()
			if name := web.gachaPossessionName(cat, language, possessionType, id); strings.Contains(name, " #") {
				t.Errorf("%s: missing display name for type %d, id %d", language, possessionType, id)
			}
		}
		for _, banner := range cat.GachaHandler.Premium.Banners {
			for _, item := range banner.ItemsByWeaponId {
				check(int32(model.PossessionTypeWeapon), item.WeaponId)
				if item.CostumeId != 0 {
					check(int32(model.PossessionTypeCostume), item.CostumeId)
				}
			}
		}
		for _, entry := range cat.GachaEntries {
			for _, item := range entry.BoxItems {
				check(item.PossessionType, item.PossessionId)
			}
		}
	}
}

func TestGachaWebFormatsAssetText(t *testing.T) {
	web := &GachaWebHandler{names: assettext.Index{"en": {"name": `<color=#fff>Asset\nName</color>`}}}
	if got := web.gachaName("ja", "name", "fallback"); got != "Asset Name" {
		t.Fatalf("formatted fallback text = %q", got)
	}
	if got := web.gachaName("ja", "missing", "Weapon #1"); got != "Weapon #1" {
		t.Fatalf("missing text fallback = %q", got)
	}
}
