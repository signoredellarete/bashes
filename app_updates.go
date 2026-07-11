package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"
)

func (a *App) GetAppInfo() AppInfo {
	return AppInfo{
		Name:        "Bashes",
		Version:     appVersion(),
		Platform:    runtime.GOOS,
		Arch:        runtime.GOARCH,
		DataPath:    a.dataPath,
		RepoURL:     repositoryURL,
		ReadmeURL:   readmeURL,
		ReleasesURL: releasesURL,
	}
}

func (a *App) CheckForUpdate() (UpdateInfo, error) {
	current := appVersion()
	latest, err := latestGitHubRelease()
	if err != nil {
		return UpdateInfo{}, err
	}

	info := UpdateInfo{
		CurrentVersion:  current,
		LatestVersion:   latest,
		UpdateAvailable: isNewerVersion(latest, current),
		ReleaseURL:      releasesURL + "/tag/" + latest,
		RepoURL:         repositoryURL,
	}
	if _, ok := versionParts(current); !ok {
		info.Message = fmt.Sprintf("Latest release: %s. This build reports version %s, so it cannot be compared automatically.", latest, current)
	} else if info.UpdateAvailable {
		info.Message = fmt.Sprintf("Bashes %s is available. You are running %s.", latest, current)
	} else {
		info.Message = fmt.Sprintf("Bashes is up to date. Current version: %s.", current)
	}
	return info, nil
}

func appVersion() string {
	value := strings.TrimSpace(version)
	if value == "" {
		return "dev"
	}
	return value
}

func latestGitHubRelease() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/repos/signoredellarete/bashes/releases/latest", nil)
	if err != nil {
		return "", err
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "Bashes/"+appVersion())

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", fmt.Errorf("check latest release: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("check latest release: GitHub returned %s", response.Status)
	}

	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode latest release: %w", err)
	}
	if strings.TrimSpace(payload.TagName) == "" {
		return "", errors.New("latest release response did not include a tag")
	}
	return strings.TrimSpace(payload.TagName), nil
}

func isNewerVersion(latest string, current string) bool {
	latestParts, latestOK := versionParts(latest)
	currentParts, currentOK := versionParts(current)
	if !latestOK || !currentOK {
		return false
	}
	for index := 0; index < len(latestParts); index++ {
		if latestParts[index] > currentParts[index] {
			return true
		}
		if latestParts[index] < currentParts[index] {
			return false
		}
	}
	return false
}

func versionParts(value string) ([3]int, bool) {
	var result [3]int
	value = strings.TrimPrefix(strings.TrimSpace(value), "v")
	parts := strings.Split(value, ".")
	if len(parts) < 2 {
		return result, false
	}
	for index := 0; index < len(result); index++ {
		if index >= len(parts) {
			break
		}
		part := parts[index]
		if cut := strings.IndexFunc(part, func(r rune) bool { return r < '0' || r > '9' }); cut >= 0 {
			part = part[:cut]
		}
		number, err := strconv.Atoi(part)
		if err != nil {
			return result, false
		}
		result[index] = number
	}
	return result, true
}
