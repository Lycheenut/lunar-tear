package service

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"lunar-tear/server/internal/assettext"
	"lunar-tear/server/internal/gacha"
	"lunar-tear/server/internal/gametime"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
	"lunar-tear/server/internal/store"
)

//go:embed gacha_web.html
var gachaWebFiles embed.FS

var gachaWebTemplate = template.Must(template.New("gacha_web.html").Funcs(template.FuncMap{
	"percent": func(rate float64) string { return fmt.Sprintf("%.6f%%", rate*100) },
}).ParseFS(gachaWebFiles, "gacha_web.html"))

var gachaTextTags = regexp.MustCompile(`<[^>]+>`)

type GachaWebHandler struct {
	users    store.UserRepository
	sessions store.SessionRepository
	catalogs func() *runtime.Catalogs
	names    assettext.Index
}

func NewGachaWebHandler(users store.UserRepository, sessions store.SessionRepository, holder *runtime.Holder, names assettext.Index) *GachaWebHandler {
	return &GachaWebHandler{users: users, sessions: sessions, catalogs: holder.Get, names: names}
}

type gachaWebRow struct {
	Name, Bonus, Kind  string
	ID                 int32
	Count              int32
	Pickup             bool
	Rates              []float64
	Remaining, Maximum int32
	Unlimited          bool
}

type gachaWebChoice struct {
	Label, URL string
	Selected   bool
}
type gachaWebPage struct {
	Language                           string
	Text                               map[string]string
	Title, Kind, Error, Updated, Reset string
	GachaID, BoxNumber                 int32
	Box                                bool
	Columns                            []string
	Groups, Items                      []gachaWebRow
	Phases                             []gachaWebChoice
}

func (s *GachaWebHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !IsGachaWebPath(r.URL.Path) {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		w.Header().Set("Allow", "GET, HEAD")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	language := "en"
	if strings.Contains(r.URL.Path, "/ja/") || strings.HasPrefix(strings.ToLower(r.URL.Query().Get("language")), "ja") {
		language = "ja"
	}
	page := gachaWebPage{Language: language, Text: gachaWebText[language]}
	fail := func(code int, key string) { page.Error = page.Text[key]; renderGachaWeb(w, r, code, page) }
	query, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		fail(http.StatusBadRequest, "invalid")
		return
	}
	ids := query["gachaId"]
	if len(ids) == 0 {
		fail(http.StatusBadRequest, "invalid")
		return
	}
	requested := make(map[int32]bool)
	var firstID int32
	for _, value := range ids {
		id, err := strconv.ParseInt(value, 10, 32)
		if err != nil || id <= 0 {
			fail(http.StatusBadRequest, "invalid")
			return
		}
		if firstID == 0 {
			firstID = int32(id)
		}
		requested[int32(id)] = true
	}
	id := firstID
	if values := query["nowPlayingId"]; len(values) > 0 {
		current, err := strconv.ParseInt(values[0], 10, 32)
		if err != nil || len(values) != 1 || !requested[int32(current)] {
			fail(http.StatusBadRequest, "invalid")
			return
		}
		id = int32(current)
	} else if len(requested) > 1 {
		fail(http.StatusBadRequest, "invalid")
		return
	}
	// Ignore userId/playerId: only a validated session can select player state.
	keys := query["sessionKey"]
	if len(keys) != 1 || keys[0] == "" {
		fail(http.StatusUnauthorized, "session")
		return
	}
	userID, err := s.sessions.ResolveUserId(keys[0])
	if err != nil {
		fail(http.StatusUnauthorized, "session")
		return
	}
	user, err := s.users.LoadUser(userID)
	if err != nil {
		fail(http.StatusInternalServerError, "unavailable")
		return
	}
	cat := s.catalogs()
	entry := findCatalogEntry(cat.GachaEntries, id)
	now := gametime.NowMillis()
	if entry == nil || !gachaVisibleForUser(cat, &user, *entry, now) {
		fail(http.StatusNotFound, "missing")
		return
	}
	state := user.Gacha.BannerStates[id]
	page.GachaID = id
	page.Title = s.gachaName(language, "gacha.title."+entry.BannerAssetName, entry.BannerAssetName)
	page.Updated = time.UnixMilli(now).UTC().Format("2006-01-02 15:04:05 UTC")
	switch entry.GachaLabelType {
	case model.GachaLabelPremium:
		page.Kind = "PREMIUM GACHA"
		if cat.GachaHandler == nil || cat.GachaHandler.Premium == nil {
			fail(http.StatusServiceUnavailable, "unavailable")
			return
		}
		phase, ok := gachaWebPhase(*entry, state, query.Get("gachaPricePhaseId"))
		if !ok {
			fail(http.StatusBadRequest, "invalid")
			return
		}
		odds, err := gacha.PremiumOdds(cat.GachaHandler.Premium.Banners[id], *entry, phase)
		if err != nil {
			fail(http.StatusServiceUnavailable, "unavailable")
			return
		}
		for _, candidate := range entry.PricePhases {
			if candidate.DrawCount <= 0 {
				continue
			}
			q := r.URL.Query()
			q.Set("gachaPricePhaseId", strconv.Itoa(int(candidate.PhaseId)))
			label := fmt.Sprintf(page.Text["draws"], candidate.DrawCount)
			if candidate.StepNumber > 0 {
				label = fmt.Sprintf("STEP %d · %s", candidate.StepNumber, label)
			}
			if candidate.PriceType == model.PriceTypeConsumableItem {
				label += " · " + page.Text["ticket"]
			}
			page.Phases = append(page.Phases, gachaWebChoice{label, r.URL.Path + "?" + q.Encode(), candidate.PhaseId == phase.PhaseId})
		}
		for _, slot := range odds {
			label := fmt.Sprintf(page.Text["slot"], slot.First)
			if slot.First != slot.Last {
				label = fmt.Sprintf(page.Text["slots"], slot.First, slot.Last)
			}
			page.Columns = append(page.Columns, label)
		}
		for groupIndex, group := range odds[0].Groups {
			kind := page.Text["weaponOnly"]
			if group.GrantType == gacha.GrantCharacterWeapon {
				kind = page.Text["characterWeapon"]
			}
			row := gachaWebRow{Name: fmt.Sprintf("%d★", group.Star), Kind: kind}
			for _, slot := range odds {
				row.Rates = append(row.Rates, slot.Groups[groupIndex].Rate)
			}
			page.Groups = append(page.Groups, row)
			for itemIndex, item := range group.Items {
				row := gachaWebRow{Name: s.gachaPossessionName(cat, language, int32(model.PossessionTypeWeapon), item.WeaponId), ID: item.WeaponId, Kind: fmt.Sprintf("%d★ · %s", group.Star, kind), Pickup: item.Pickup}
				if item.CostumeId != 0 {
					row.Bonus = s.gachaPossessionName(cat, language, int32(model.PossessionTypeCostume), item.CostumeId)
				}
				for _, slot := range odds {
					row.Rates = append(row.Rates, slot.Groups[groupIndex].Items[itemIndex].Rate)
				}
				page.Items = append(page.Items, row)
			}
		}
	case model.GachaLabelChapter, model.GachaLabelEvent:
		page.Box, page.BoxNumber = true, max(state.BoxNumber, 1)
		page.Kind = "EVENT GACHA"
		if entry.GachaLabelType == model.GachaLabelChapter {
			page.Kind = "CHAPTER GACHA"
			page.Reset = time.UnixMilli(gametime.StartOfNextBusinessMonthAtMillis(now)).UTC().Format("2006-01-02 15:04 UTC")
		}
		if len(entry.BoxItems) == 0 {
			fail(http.StatusServiceUnavailable, "unavailable")
			return
		}
		for _, item := range gacha.BoxOdds(*entry, state, now) {
			page.Items = append(page.Items, gachaWebRow{Name: s.gachaPossessionName(cat, language, item.PossessionType, item.PossessionId), ID: item.PossessionId, Count: item.Count, Remaining: item.Remaining, Maximum: item.MaxCount, Unlimited: item.Unlimited, Rates: []float64{item.Rate}})
		}
	default:
		fail(http.StatusNotFound, "missing")
		return
	}
	// Stable ordering keeps searches and refreshes easy to follow.
	if !page.Box {
		sort.SliceStable(page.Items, func(i, j int) bool {
			if page.Items[i].Pickup != page.Items[j].Pickup {
				return page.Items[i].Pickup
			}
			return page.Items[i].ID < page.Items[j].ID
		})
	}
	renderGachaWeb(w, r, http.StatusOK, page)
}

func gachaWebPhase(entry store.GachaCatalogEntry, state store.GachaBannerState, requested string) (store.GachaPricePhaseEntry, bool) {
	var selected store.GachaPricePhaseEntry
	for _, phase := range entry.PricePhases {
		if phase.DrawCount <= 0 {
			continue
		}
		if requested != "" {
			if strconv.Itoa(int(phase.PhaseId)) == requested {
				return phase, true
			}
			continue
		}
		if entry.GachaModeType == model.GachaModeStepup && phase.StepNumber != max(state.StepNumber, 1) {
			continue
		}
		if phase.DrawCount > selected.DrawCount {
			selected = phase
		}
	}
	return selected, selected.DrawCount > 0
}

func renderGachaWeb(w http.ResponseWriter, r *http.Request, code int, page gachaWebPage) {
	var body bytes.Buffer
	if err := gachaWebTemplate.Execute(&body, page); err != nil {
		http.Error(w, "Gacha details unavailable", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	if r.Method != http.MethodHead {
		w.Write(body.Bytes())
	}
}

func (s *GachaWebHandler) gachaName(language, key, fallback string) string {
	name := s.names[language][key]
	if name == "" {
		name = s.names["en"][key]
	}
	if name != "" {
		return strings.TrimSpace(strings.ReplaceAll(gachaTextTags.ReplaceAllString(name, ""), `\n`, " "))
	}
	return fallback
}

func (s *GachaWebHandler) gachaPossessionName(cat *runtime.Catalogs, language string, possessionType, id int32) string {
	key, kind := "", "Item"
	switch model.PossessionType(possessionType) {
	case model.PossessionTypeWeapon:
		kind = "Weapon"
		if cat.Weapon != nil {
			v := cat.Weapon.Weapons[id]
			key = fmt.Sprintf("weapon.name.wp%03d%03d.1", v.WeaponType, v.AssetVariationId)
		}
	case model.PossessionTypeCostume:
		kind = "Costume"
		if cat.Costume != nil {
			v := cat.Costume.Costumes[id]
			key = fmt.Sprintf("costume.name.ch%03d%03d", v.ActorSkeletonId, v.AssetVariationId)
			name := s.gachaName(language, key, "")
			if name != "" {
				return s.gachaName(language, fmt.Sprintf("character.name.%d", v.CharacterId), "") + " · " + name
			}
		}
	case model.PossessionTypeMaterial:
		kind = "Material"
		if cat.Material != nil {
			v := cat.Material.All[id]
			key = fmt.Sprintf("material.name.%03d%03d", v.AssetCategoryId, v.AssetVariationId)
		}
	case model.PossessionTypeConsumableItem:
		if cat.ConsumableItem != nil {
			v := cat.ConsumableItem.All[id]
			key = fmt.Sprintf("consumable_item.name.%03d%03d", v.AssetCategoryId, v.AssetVariationId)
		}
	case model.PossessionTypeCompanion:
		kind = "Companion"
		if cat.Companion != nil {
			v := cat.Companion.CompanionById[id]
			key = fmt.Sprintf("companion.name.cm%03d%03d", v.ActorSkeletonId, v.AssetVariationId)
		}
	case model.PossessionTypeFreeGem, model.PossessionTypePaidGem:
		return gachaWebText[language]["gems"]
	}
	return s.gachaName(language, key, fmt.Sprintf("%s #%d", kind, id))
}
