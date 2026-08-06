// Package runtime owns the live, hot-swappable view of master data.
//
// The Holder atomically swaps a *Catalogs aggregate every time the operator
// asks the server to re-read assets/release/20240404193219.bin.e (typically via
// the admin service in cmd/lunar-tear/admin.go). gRPC services hold a *Holder
// and call Get() at the start of each RPC, so they always see a consistent
// snapshot.
package runtime

import (
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"lunar-tear/server/internal/campaign"
	"lunar-tear/server/internal/gacha"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/questflow"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/userdata"
)

// Catalogs is an immutable snapshot of every catalog and catalog-derived
// handler the server needs at runtime. A new *Catalogs is built from scratch
// on every reload and atomically published via Holder.
type Catalogs struct {
	MasterDataHash    string
	GameConfig        *masterdata.GameConfig
	Parts             *masterdata.PartsCatalog
	Quest             *masterdata.QuestCatalog
	Mission           *masterdata.MissionCatalog
	GachaEntries      []store.GachaCatalogEntry
	GachaMedals       map[int32]masterdata.GachaMedalInfo
	GachaPool         *masterdata.GachaCatalog
	GachaConfig       *gacha.Config
	GachaConfigHash   string
	GachaConfigExists bool
	PremiumGacha      *gacha.PremiumCatalog
	Shop              *masterdata.ShopCatalog
	DupExchange       map[int32][]model.DupExchangeEntry
	ConditionResolver *masterdata.ConditionResolver
	CageOrnament      *masterdata.CageOrnamentCatalog
	LoginBonus        *masterdata.LoginBonusCatalog
	CharacterViewer   *masterdata.CharacterViewerCatalog
	Omikuji           *masterdata.OmikujiCatalog
	Material          *masterdata.MaterialCatalog
	ConsumableItem    *masterdata.ConsumableItemCatalog
	Costume           *masterdata.CostumeCatalog
	Weapon            *masterdata.WeaponCatalog
	Explore           *masterdata.ExploreCatalog
	Gimmick           *masterdata.GimmickCatalog
	CharacterBoard    *masterdata.CharacterBoardCatalog
	CharacterRebirth  *masterdata.CharacterRebirthCatalog
	Companion         *masterdata.CompanionCatalog
	SideStory         *masterdata.SideStoryCatalog
	BigHunt           *masterdata.BigHuntCatalog
	Tower             *masterdata.TowerCatalog
	Labyrinth         *masterdata.LabyrinthCatalog
	LimitContent      *masterdata.LimitContentCatalog
	Campaign          *campaign.Catalog

	QuestHandler *questflow.QuestHandler
	GachaHandler *gacha.GachaHandler
}

type Holder struct {
	binPath         string
	gachaConfigPath string
	cur             atomic.Pointer[Catalogs]
	mu              sync.Mutex
}

func NewHolder(binPath string) (*Holder, error) {
	return NewHolderWithGachaConfig(binPath, "")
}

func NewHolderWithGachaConfig(binPath, gachaConfigPath string) (*Holder, error) {
	h := &Holder{binPath: binPath, gachaConfigPath: gachaConfigPath}
	if err := h.Reload(); err != nil {
		return nil, err
	}
	return h, nil
}

func (h *Holder) Reload() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	c, err := loadCatalogs(h.binPath, h.gachaConfigPath, false)
	if err != nil {
		return err
	}
	h.publish(c)
	h.touch()
	return nil
}

// InstallAndReload fully loads a candidate master-data file before atomically
// replacing the file consumed by the game server and CDN. A bad candidate is
// never published and never replaces the current file.
func (h *Holder) InstallAndReload(candidatePath string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	candidateInfo, err := os.Stat(candidatePath)
	if err != nil {
		return fmt.Errorf("stat candidate: %w", err)
	}
	if candidateInfo.Size() == 0 {
		return fmt.Errorf("candidate master data is empty")
	}
	if currentInfo, statErr := os.Stat(h.binPath); statErr == nil {
		if err := os.Chmod(candidatePath, currentInfo.Mode().Perm()); err != nil {
			return fmt.Errorf("preserve master-data permissions: %w", err)
		}
	}

	c, err := loadCatalogs(candidatePath, h.gachaConfigPath, false)
	if err != nil {
		_ = memorydb.Init(h.binPath)
		return fmt.Errorf("validate candidate: %w", err)
	}
	if err := replaceFile(candidatePath, h.binPath); err != nil {
		_ = memorydb.Init(h.binPath)
		return fmt.Errorf("install candidate: %w", err)
	}
	h.publish(c)
	h.touch()
	return nil
}

var ErrGachaConfigConflict = errors.New("Gacha config changed since it was loaded")

func (h *Holder) InstallGachaConfig(candidatePath, expectedHash string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.gachaConfigPath == "" {
		return fmt.Errorf("Gacha config path is not configured")
	}
	current := h.cur.Load()
	if current == nil || expectedHash == "" || current.GachaConfigHash != expectedHash {
		return ErrGachaConfigConflict
	}
	candidateInfo, err := os.Stat(candidatePath)
	if err != nil {
		return fmt.Errorf("stat Gacha config candidate: %w", err)
	}
	if candidateInfo.Size() == 0 {
		return fmt.Errorf("Gacha config candidate is empty")
	}
	if currentInfo, statErr := os.Stat(h.gachaConfigPath); statErr == nil {
		if err := os.Chmod(candidatePath, currentInfo.Mode().Perm()); err != nil {
			return fmt.Errorf("preserve Gacha config permissions: %w", err)
		}
	}
	c, err := loadCatalogs(h.binPath, candidatePath, true)
	if err != nil {
		return fmt.Errorf("validate Gacha config candidate: %w", err)
	}
	if err := replaceFile(candidatePath, h.gachaConfigPath); err != nil {
		return fmt.Errorf("install Gacha config candidate: %w", err)
	}
	h.publish(c)
	return nil
}

func loadCatalogs(path, gachaConfigPath string, requireCompleteGacha bool) (*Catalogs, error) {
	if err := memorydb.Init(path); err != nil {
		return nil, fmt.Errorf("memorydb.Init: %w", err)
	}
	masterDataHash, err := gacha.FileHash(path)
	if err != nil {
		return nil, fmt.Errorf("hash master data: %w", err)
	}
	config := gacha.DefaultConfig()
	configHash := gacha.ContentHash(nil)
	configExists := false
	if gachaConfigPath != "" {
		config, configHash, configExists, err = gacha.ReadConfig(gachaConfigPath)
		if err != nil {
			return nil, err
		}
	}
	c, err := buildCatalogs(config, configHash, configExists, masterDataHash, requireCompleteGacha)
	if err != nil {
		return nil, fmt.Errorf("buildCatalogs: %w", err)
	}
	return c, nil
}

func (h *Holder) publish(c *Catalogs) {
	h.cur.Store(c)
	userdata.SetQuestHandler(c.QuestHandler)
}

func (h *Holder) touch() {
	now := time.Now()
	if err := os.Chtimes(h.binPath, now, now); err != nil {
		log.Printf("[runtime] os.Chtimes(%s) failed (clients may not invalidate cache): %v", h.binPath, err)
	}
}

func (h *Holder) Get() *Catalogs {
	return h.cur.Load()
}
