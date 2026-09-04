package main

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const openFileCacheRetention = 7 * 24 * time.Hour

var openableFileExtensions = map[string]struct{}{
	".aac": {}, ".avi": {}, ".avif": {}, ".bmp": {}, ".conf": {},
	".csv": {}, ".doc": {}, ".docx": {}, ".epub": {}, ".flac": {},
	".gif": {}, ".heic": {}, ".ico": {}, ".ini": {}, ".jpeg": {},
	".jpg": {}, ".json": {}, ".log": {}, ".m4a": {}, ".m4v": {},
	".md": {}, ".mkv": {}, ".mov": {}, ".mp3": {}, ".mp4": {},
	".mpeg": {}, ".mpg": {}, ".odp": {}, ".ods": {}, ".odt": {},
	".ogg": {}, ".pdf": {}, ".png": {}, ".ppt": {}, ".pptx": {},
	".rtf": {}, ".svg": {}, ".tif": {}, ".tiff": {}, ".txt": {},
	".wav": {}, ".webm": {}, ".webp": {}, ".xls": {}, ".xlsx": {},
	".xml": {}, ".yaml": {}, ".yml": {},
}

var unsafeCacheFileName = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1f]`)

func defaultOpenFileCacheDir(dataPath string) string {
	if cacheDir, err := os.UserCacheDir(); err == nil && strings.TrimSpace(cacheDir) != "" {
		return filepath.Join(cacheDir, "Bashes", "opened-files")
	}
	return filepath.Join(filepath.Dir(dataPath), "cache", "opened-files")
}

func validateOpenableFile(filePath string) error {
	ext := strings.ToLower(path.Ext(strings.TrimSpace(filePath)))
	if _, ok := openableFileExtensions[ext]; ok {
		return nil
	}
	if ext == "" {
		return errors.New("files without an extension cannot be opened from Bashes")
	}
	return fmt.Errorf("%s files cannot be opened from Bashes", ext)
}

func cacheFileName(remotePath string) string {
	name := unsafeCacheFileName.ReplaceAllString(path.Base(remotePath), "_")
	name = strings.Trim(name, ". ")
	if name == "" {
		return "remote-file"
	}
	return name
}

func pruneOpenFileCache(cacheDir string, olderThan time.Duration) error {
	entries, err := os.ReadDir(cacheDir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	cutoff := time.Now().Add(-olderThan)
	for _, entry := range entries {
		info, infoErr := entry.Info()
		if infoErr == nil && info.ModTime().Before(cutoff) {
			_ = os.RemoveAll(filepath.Join(cacheDir, entry.Name()))
		}
	}
	return nil
}
