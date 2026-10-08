package update

import (
	"fmt"
	"os"
)

func replaceExecutable(staged, target string) (string, error) {
	// Windows permits renaming a running executable but cannot overwrite it.
	backup := target + ".old"
	if err := os.Remove(backup); err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("remove previous backup %s: %w", backup, err)
	}
	if err := os.Rename(target, backup); err != nil {
		return "", err
	}
	if err := os.Rename(staged, target); err != nil {
		if rollbackErr := os.Rename(backup, target); rollbackErr != nil {
			return "", fmt.Errorf("install update: %w; restore previous executable: %v; backup remains at %s", err, rollbackErr, backup)
		}
		return "", err
	}
	if err := os.Remove(backup); err != nil {
		return backup, nil
	}
	return "", nil
}
