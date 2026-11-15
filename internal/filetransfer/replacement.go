package filetransfer

import "fmt"

// ReplaceWithBackup promotes temporary to target while preserving target until
// the promotion succeeds. Backup cleanup is best-effort after the commit.
func ReplaceWithBackup(temporary, target, backup string, rename func(string, string) error, remove func(string) error) error {
	if err := rename(target, backup); err != nil {
		return fmt.Errorf("stage existing destination: %w", err)
	}
	if err := rename(temporary, target); err != nil {
		if restoreErr := rename(backup, target); restoreErr != nil {
			return fmt.Errorf("replace destination: %w; restore original: %v", err, restoreErr)
		}
		return fmt.Errorf("replace destination: %w", err)
	}
	_ = remove(backup)
	return nil
}
