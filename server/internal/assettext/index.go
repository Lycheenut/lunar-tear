// Package assettext reads localized text directly from the installed game assets.
package assettext

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// Index is a language -> text key -> text snapshot, loaded once at startup.
type Index map[string]map[string]string

// Load reads Gacha titles and reward names beside the supplied master-data file.
// Like prod's configuration manager, it decodes Octo masks, UnityFS blocks and
// serialized TextAssets. Base bundles load before their content patches.
func Load(masterDataPath string) (Index, error) {
	assetsRoot := filepath.Dir(filepath.Dir(masterDataPath))
	bundleRoot := filepath.Join(assetsRoot, "revisions", "0", "assetbundle")
	index := make(Index)
	for _, language := range []string{"en", "ja"} {
		entries := make(map[string]string)
		for _, category := range []string{"character", "gacha_title", "possession/weapon", "possession/costume", "possession/material", "possession/consumable_item", "possession/companion"} {
			base := filepath.Join(bundleRoot, "text", language, filepath.FromSlash(category))
			paths := []string{base + ".assetbundle"}
			err := filepath.WalkDir(base, func(path string, entry fs.DirEntry, err error) error {
				if path == base && errors.Is(err, fs.ErrNotExist) {
					return nil // A base bundle does not require a patch directory.
				}
				if err != nil {
					return err
				}
				if !entry.IsDir() && strings.HasSuffix(path, ".assetbundle") {
					paths = append(paths, path)
				}
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("list text patches %s: %w", base, err)
			}
			sort.Strings(paths[1:])
			for _, path := range paths {
				texts, err := readTextAssetBundle(path, bundleRoot)
				if err != nil {
					return nil, fmt.Errorf("load text asset %s: %w", path, err)
				}
				for key, value := range texts {
					entries[key] = value
				}
			}
		}
		index[language] = entries
	}
	return index, nil
}
