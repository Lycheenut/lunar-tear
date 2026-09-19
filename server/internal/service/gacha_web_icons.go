package service

import (
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/runtime"
)

var gachaAttributes = map[int32]string{1: "dark", 2: "fire", 3: "light", 4: "none", 5: "water", 6: "wind"}
var gachaWeaponTypes = map[int32]string{1: "sword", 2: "spear", 3: "bigsword", 4: "fist", 5: "staff", 6: "gun"}
var gachaRarities = map[int32]string{1: "bronze", 2: "silver", 3: "gold", 4: "rainbow"}

type gachaWebIcon struct {
	Sprite string
	Label  string
}

type gachaWebRarity struct {
	Count  int32
	Sprite string
}

type gachaWebReward struct {
	Name, Character string
	Rarity          gachaWebRarity
	Attribute       gachaWebIcon
	Weapon          gachaWebIcon
}

// Pre-extracted PNGs are supplied in assets/ui/gacha at deployment time.
// Inline PNGs avoid extra authenticated requests and external asset host dependencies.
func loadGachaWebIcons(assetsRoot string) map[string]template.URL {
	var names []string
	for _, rarity := range gachaRarities {
		names = append(names, "thumbnail_rarity_icon_"+rarity)
	}
	for _, attribute := range gachaAttributes {
		names = append(names, "attribute_"+attribute)
		if attribute != "none" {
			names = append(names, "proper_attribute_"+attribute)
		}
	}
	for _, weapon := range gachaWeaponTypes {
		names = append(names, "weapon_icon_"+weapon)
	}
	icons := make(map[string]template.URL, len(names))
	for _, name := range names {
		data, err := os.ReadFile(filepath.Join(assetsRoot, "ui", "gacha", name+".png"))
		if err == nil && http.DetectContentType(data) == "image/png" {
			icons[name] = template.URL("data:image/png;base64," + base64.StdEncoding.EncodeToString(data))
		}
	}
	log.Printf("loaded Gacha UI icons from assets: %d/%d (missing icons use text labels)", len(icons), len(names))
	return icons
}

func (s *GachaWebHandler) gachaRarity(star int32) gachaWebRarity {
	return gachaWebRarity{Count: star, Sprite: s.gachaSprite("thumbnail_rarity_icon_" + gachaRarities[star])}
}

func (s *GachaWebHandler) gachaSprite(name string) string {
	if s.icons[name] != "" {
		return name
	}
	return ""
}

func (s *GachaWebHandler) gachaAttribute(language string, id int32, proper bool) gachaWebIcon {
	name, ok := gachaAttributes[id]
	if !ok {
		return gachaWebIcon{}
	}
	prefix := "attribute_"
	if proper && name != "none" {
		prefix = "proper_attribute_"
	}
	return gachaWebIcon{Sprite: s.gachaSprite(prefix + name), Label: gachaWebText[language]["attribute_"+name]}
}

func (s *GachaWebHandler) gachaWeaponIcon(language string, id int32) gachaWebIcon {
	name, ok := gachaWeaponTypes[id]
	if !ok {
		return gachaWebIcon{}
	}
	return gachaWebIcon{Sprite: s.gachaSprite("weapon_icon_" + name), Label: gachaWebText[language]["weapon_"+name]}
}

func (s *GachaWebHandler) gachaWeaponReward(cat *runtime.Catalogs, language string, id, star int32) gachaWebReward {
	reward := gachaWebReward{Name: s.gachaPossessionName(cat, language, int32(model.PossessionTypeWeapon), id), Rarity: s.gachaRarity(star)}
	if cat.Weapon != nil {
		weapon := cat.Weapon.Weapons[id]
		reward.Attribute = s.gachaAttribute(language, weapon.AttributeType, false)
		reward.Weapon = s.gachaWeaponIcon(language, weapon.WeaponType)
	}
	return reward
}

func (s *GachaWebHandler) gachaCostumeReward(cat *runtime.Catalogs, language string, id int32) *gachaWebReward {
	reward := &gachaWebReward{Name: s.gachaPossessionName(cat, language, int32(model.PossessionTypeCostume), id)}
	if cat.Costume != nil {
		costume, ok := cat.Costume.Costumes[id]
		if ok {
			reward.Name = s.gachaName(language, fmt.Sprintf("costume.name.ch%03d%03d", costume.ActorSkeletonId, costume.AssetVariationId), reward.Name)
			reward.Character = s.gachaName(language, fmt.Sprintf("character.name.%d", costume.CharacterId), "")
			reward.Rarity = s.gachaRarity(costume.RarityType / 10)
			reward.Attribute = s.gachaAttribute(language, cat.Costume.ProperAttributes[id], true)
			reward.Weapon = s.gachaWeaponIcon(language, costume.SkillfulWeaponType)
		}
	}
	return reward
}
