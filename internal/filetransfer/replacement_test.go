package filetransfer

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestReplaceWithBackupRestoresOriginalWhenPromotionFails(t *testing.T) {
	promotionErr := errors.New("promotion failed")
	var renames [][2]string
	err := ReplaceWithBackup("temporary", "target", "backup", func(oldPath, newPath string) error {
		renames = append(renames, [2]string{oldPath, newPath})
		if oldPath == "temporary" {
			return promotionErr
		}
		return nil
	}, func(string) error {
		t.Fatal("remove must not run when promotion fails")
		return nil
	})

	if !strings.Contains(err.Error(), promotionErr.Error()) {
		t.Fatalf("ReplaceWithBackup() error = %v, want promotion failure", err)
	}
	want := [][2]string{{"target", "backup"}, {"temporary", "target"}, {"backup", "target"}}
	if !reflect.DeepEqual(renames, want) {
		t.Fatalf("rename calls = %v, want %v", renames, want)
	}
}

func TestReplaceWithBackupCommitsBeforeCleanup(t *testing.T) {
	var operations []string
	err := ReplaceWithBackup("temporary", "target", "backup", func(oldPath, newPath string) error {
		operations = append(operations, "rename "+oldPath+" "+newPath)
		return nil
	}, func(path string) error {
		operations = append(operations, "remove "+path)
		return errors.New("best-effort cleanup")
	})
	if err != nil {
		t.Fatalf("ReplaceWithBackup() error = %v", err)
	}
	want := []string{"rename target backup", "rename temporary target", "remove backup"}
	if !reflect.DeepEqual(operations, want) {
		t.Fatalf("operations = %v, want %v", operations, want)
	}
}
