//go:build !windows

package runtime

import "os"

func replaceFile(source, target string) error {
	return os.Rename(source, target)
}
