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
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"lunar-tear/server/internal/activitygroup"
	"lunar-tear/server/internal/campaign"
	"lunar-tear/server/internal/gacha"
	"lunar-tear/server/internal/importantitem"
	"lunar-tear/server/internal/masterdata"
	"lunar-tear/server/internal/masterdata/memorydb"
	"lunar-tear/server/internal/model"
	"lunar-tear/server/internal/questdrop"
	"lunar-tear/server/internal/questflow"
	"lunar-tear/server/internal/store"
	"lunar-tear/server/internal/userdata"
)

// Catalogs is an immutable snapshot of every catalog and catalog-derived
// handler the server needs at runtime. A new *Catalogs is built from scratch
// on every reload and atomically published via Holder.
type Catalogs struct {
	MasterDataHash        string
	GameConfig            *masterdata.GameConfig
	Parts                 *masterdata.PartsCatalog
	Quest                 *masterdata.QuestCatalog
	Mission               *masterdata.MissionCatalog
	GachaEntries          []store.GachaCatalogEntry
	GachaMedals           map[int32]masterdata.GachaMedalInfo
	GachaPool             *masterdata.GachaCatalog
	GachaConfig           *gacha.Config
	GachaConfigHash       string
	GachaConfigExists     bool
	ActivityConfig        *activitygroup.Config
	ActivityConfigHash    string
	QuestDropConfig       *questdrop.Config
	QuestDropConfigHash   string
	QuestDropConfigExists bool
	PremiumGacha          *gacha.PremiumCatalog
	Shop                  *masterdata.ShopCatalog
	DupExchange           map[int32][]model.DupExchangeEntry
	ConditionResolver     *masterdata.ConditionResolver
	CageOrnament          *masterdata.CageOrnamentCatalog
	LoginBonus            *masterdata.LoginBonusCatalog
	CharacterViewer       *masterdata.CharacterViewerCatalog
	Omikuji               *masterdata.OmikujiCatalog
	Material              *masterdata.MaterialCatalog
	ConsumableItem        *masterdata.ConsumableItemCatalog
	Costume               *masterdata.CostumeCatalog
	Weapon                *masterdata.WeaponCatalog
	Explore               *masterdata.ExploreCatalog
	Gimmick               *masterdata.GimmickCatalog
	CharacterBoard        *masterdata.CharacterBoardCatalog
	CharacterRebirth      *masterdata.CharacterRebirthCatalog
	Companion             *masterdata.CompanionCatalog
	SideStory             *masterdata.SideStoryCatalog
	BigHunt               *masterdata.BigHuntCatalog
	Tower                 *masterdata.TowerCatalog
	Labyrinth             *masterdata.LabyrinthCatalog
	LimitContent          *masterdata.LimitContentCatalog
	Campaign              *campaign.Catalog
	ImportantItems        *importantitem.Catalog

	QuestHandler *questflow.QuestHandler
	GachaHandler *gacha.GachaHandler
}

type Holder struct {
	binPath             string
	gachaConfigPath     string
	questDropConfigPath string
	activityConfigPath  string
	cur                 atomic.Pointer[Catalogs]
	mu                  sync.Mutex
}

func NewHolder(binPath string) (*Holder, error) {
	return NewHolderWithConfigs(binPath, "", "", "")
}

func NewHolderWithGachaConfig(binPath, gachaConfigPath string) (*Holder, error) {
	return NewHolderWithConfigs(binPath, gachaConfigPath, "", "")
}

func NewHolderWithConfigs(binPath, gachaConfigPath, questDropConfigPath, activityConfigPath string) (*Holder, error) {
	h := &Holder{binPath: binPath, gachaConfigPath: gachaConfigPath, questDropConfigPath: questDropConfigPath, activityConfigPath: activityConfigPath}
	if err := h.Reload(); err != nil {
		return nil, err
	}
	return h, nil
}

func (h *Holder) Reload() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	c, err := loadCatalogs(h.binPath, h.gachaConfigPath, h.questDropConfigPath, h.activityConfigPath, false, false)
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

	c, err := loadCatalogs(candidatePath, h.gachaConfigPath, h.questDropConfigPath, h.activityConfigPath, false, false)
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
var ErrMasterDataConflict = errors.New("master data changed since it was loaded")
var ErrActivityConfigConflict = errors.New("activity group config changed since it was loaded")

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
	c, err := loadCatalogs(h.binPath, candidatePath, h.questDropConfigPath, h.activityConfigPath, true, false)
	if err != nil {
		return fmt.Errorf("validate Gacha config candidate: %w", err)
	}
	if err := replaceFile(candidatePath, h.gachaConfigPath); err != nil {
		return fmt.Errorf("install Gacha config candidate: %w", err)
	}
	h.publish(c)
	return nil
}

// InstallGachaConfigAndMasterData validates a Gacha config together with its
// explicitly edited master-data candidate, then publishes both as one runtime
// snapshot. If installing the config fails after the master-data replacement,
// the original master data is restored before returning.
func (h *Holder) InstallGachaConfigAndMasterData(gachaCandidatePath, masterDataCandidatePath, expectedGachaHash, expectedMasterDataHash string) error {
	return h.installActivityCandidates("", gachaCandidatePath, masterDataCandidatePath, "", expectedGachaHash, expectedMasterDataHash, true)
}

// InstallActivityConfig preserves existing pool completeness while publishing
// explicit activity membership or schedule edits.
func (h *Holder) InstallActivityConfig(activityCandidatePath, gachaCandidatePath, masterDataCandidatePath, expectedActivityHash, expectedGachaHash, expectedMasterDataHash string) error {
	if activityCandidatePath == "" || h.activityConfigPath == "" {
		return fmt.Errorf("activity group config path is not configured")
	}
	return h.installActivityCandidates(activityCandidatePath, gachaCandidatePath, masterDataCandidatePath, expectedActivityHash, expectedGachaHash, expectedMasterDataHash, false)
}

func (h *Holder) installActivityCandidates(activityCandidatePath, gachaCandidatePath, masterDataCandidatePath, expectedActivityHash, expectedGachaHash, expectedMasterDataHash string, requireComplete bool) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if gachaCandidatePath != "" && h.gachaConfigPath == "" {
		return fmt.Errorf("Gacha config path is not configured")
	}
	current := h.cur.Load()
	if current == nil || expectedGachaHash == "" || current.GachaConfigHash != expectedGachaHash {
		return ErrGachaConfigConflict
	}
	if expectedMasterDataHash == "" || current.MasterDataHash != expectedMasterDataHash {
		return ErrMasterDataConflict
	}
	if activityCandidatePath != "" && (expectedActivityHash == "" || current.ActivityConfigHash != expectedActivityHash) {
		return ErrActivityConfigConflict
	}
	type replacement struct {
		candidate, target, label string
		original                 []byte
		mode                     os.FileMode
		existed                  bool
	}
	files := []replacement{
		{candidate: masterDataCandidatePath, target: h.binPath, label: "master data"},
		{candidate: gachaCandidatePath, target: h.gachaConfigPath, label: "Gacha config"},
		{candidate: activityCandidatePath, target: h.activityConfigPath, label: "activity group config"},
	}
	paths := make([]string, len(files))
	for i := range files {
		file := &files[i]
		paths[i] = file.target
		if file.candidate == "" {
			continue
		}
		paths[i] = file.candidate
		if err := prepareReplacementCandidate(file.candidate, file.target, file.label); err != nil {
			return err
		}
		var err error
		file.original, err = os.ReadFile(file.target)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("back up %s: %w", file.label, err)
		}
		file.existed = err == nil
		if info, err := os.Stat(file.target); err == nil {
			file.mode = info.Mode().Perm()
		}
	}
	catalogs, err := loadCatalogs(paths[0], paths[1], h.questDropConfigPath, paths[2], requireComplete, false)
	if err != nil {
		_ = memorydb.Init(h.binPath)
		return fmt.Errorf("validate activity candidates: %w", err)
	}
	for i, file := range files {
		if file.candidate == "" {
			continue
		}
		if err := replaceFile(file.candidate, file.target); err != nil {
			failure := fmt.Errorf("install %s: %w", file.label, err)
			for j := i - 1; j >= 0; j-- {
				previous := files[j]
				if previous.candidate == "" {
					continue
				}
				var rollbackErr error
				if previous.existed {
					rollbackErr = restoreFile(previous.target, previous.original, previous.mode)
				} else {
					rollbackErr = os.Remove(previous.target)
				}
				if rollbackErr != nil {
					failure = errors.Join(failure, fmt.Errorf("restore %s: %w", previous.label, rollbackErr))
				}
			}
			_ = memorydb.Init(h.binPath)
			return failure
		}
	}
	h.publish(catalogs)
	if masterDataCandidatePath != "" {
		h.touch()
	}
	return nil
}

func prepareReplacementCandidate(candidatePath, targetPath, label string) error {
	info, err := os.Stat(candidatePath)
	if err != nil {
		return fmt.Errorf("stat %s candidate: %w", label, err)
	}
	if info.Size() == 0 {
		return fmt.Errorf("%s candidate is empty", label)
	}
	if currentInfo, statErr := os.Stat(targetPath); statErr == nil {
		if err := os.Chmod(candidatePath, currentInfo.Mode().Perm()); err != nil {
			return fmt.Errorf("preserve %s permissions: %w", label, err)
		}
	}
	return nil
}

func restoreFile(targetPath string, data []byte, mode os.FileMode) (err error) {
	file, err := os.CreateTemp(filepath.Dir(targetPath), ".rollback-*")
	if err != nil {
		return err
	}
	path := file.Name()
	defer func() {
		_ = file.Close()
		_ = os.Remove(path)
	}()
	if _, err = file.Write(data); err != nil {
		return err
	}
	if err = file.Sync(); err != nil {
		return err
	}
	if err = file.Close(); err != nil {
		return err
	}
	if mode != 0 {
		if err = os.Chmod(path, mode); err != nil {
			return err
		}
	}
	return replaceFile(path, targetPath)
}

var ErrQuestDropConfigConflict = errors.New("quest drop config changed since it was loaded")

func (h *Holder) InstallQuestDropConfig(candidatePath, expectedHash string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.questDropConfigPath == "" {
		return fmt.Errorf("quest drop config path is not configured")
	}
	current := h.cur.Load()
	if current == nil || expectedHash == "" || current.QuestDropConfigHash != expectedHash {
		return ErrQuestDropConfigConflict
	}
	candidateInfo, err := os.Stat(candidatePath)
	if err != nil {
		return fmt.Errorf("stat quest drop config candidate: %w", err)
	}
	if candidateInfo.Size() == 0 {
		return fmt.Errorf("quest drop config candidate is empty")
	}
	if currentInfo, statErr := os.Stat(h.questDropConfigPath); statErr == nil {
		if err := os.Chmod(candidatePath, currentInfo.Mode().Perm()); err != nil {
			return fmt.Errorf("preserve quest drop config permissions: %w", err)
		}
	}
	c, err := loadCatalogs(h.binPath, h.gachaConfigPath, candidatePath, h.activityConfigPath, false, true)
	if err != nil {
		return fmt.Errorf("validate quest drop config candidate: %w", err)
	}
	if err := replaceFile(candidatePath, h.questDropConfigPath); err != nil {
		return fmt.Errorf("install quest drop config candidate: %w", err)
	}
	h.publish(c)
	return nil
}

func loadCatalogs(path, gachaConfigPath, questDropConfigPath, activityConfigPath string, requireCompleteGacha, requireCurrentQuestDrops bool) (*Catalogs, error) {
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
	questDropConfig := questdrop.DefaultConfig()
	questDropConfigHash := questdrop.ContentHash(nil)
	questDropConfigExists := false
	if questDropConfigPath != "" {
		questDropConfig, questDropConfigHash, questDropConfigExists, err = questdrop.ReadConfig(questDropConfigPath)
		if err != nil {
			return nil, err
		}
	}
	var activityConfig *activitygroup.Config
	activityConfigHash := gacha.ContentHash(nil)
	if activityConfigPath != "" {
		activityConfig, activityConfigHash, err = activitygroup.ReadConfig(activityConfigPath)
		if err != nil {
			return nil, err
		}
	}
	c, err := buildCatalogs(config, configHash, configExists, questDropConfig, questDropConfigHash, questDropConfigExists, masterDataHash, requireCompleteGacha, requireCurrentQuestDrops)
	if err != nil {
		return nil, fmt.Errorf("buildCatalogs: %w", err)
	}
	c.ActivityConfig = activityConfig
	c.ActivityConfigHash = activityConfigHash
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
