package webappbrowser

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/signoredellarete/bashes/internal/apperrors"
	appsettings "github.com/signoredellarete/bashes/internal/settings"
)

type PreferencesService struct {
	goos     string
	settings *appsettings.Repository
	detector Detector
}

func NewPreferencesService(goos string, getenv func(string) string, settings *appsettings.Repository) *PreferencesService {
	return &PreferencesService{
		goos:     goos,
		settings: settings,
		detector: NewDetector(goos, getenv),
	}
}

func (s *PreferencesService) List() []Browser {
	return s.detector.Detect()
}

func (s *PreferencesService) Load() (appsettings.WebAppBrowser, error) {
	settings, err := s.settings.Load()
	if err != nil {
		return appsettings.WebAppBrowser{}, err
	}
	return appsettings.WebAppBrowser{
		PreferredBrowserID: settings.PreferredBrowserID,
		CustomBrowserPath:  settings.CustomBrowserPath,
	}, nil
}

func (s *PreferencesService) Save(input appsettings.WebAppBrowser) (appsettings.WebAppBrowser, error) {
	preferredID := strings.TrimSpace(input.PreferredBrowserID)
	customPath := expandHome(strings.TrimSpace(input.CustomBrowserPath))
	if preferredID != "" && !validID(preferredID) {
		return appsettings.WebAppBrowser{}, fmt.Errorf("unsupported browser %q", preferredID)
	}
	if customPath != "" {
		info, err := os.Stat(customPath)
		if err != nil {
			return appsettings.WebAppBrowser{}, fmt.Errorf("read browser executable: %w", err)
		}
		if info.IsDir() {
			return appsettings.WebAppBrowser{}, errors.New("browser executable path points to a directory")
		}
		if s.goos != "windows" && info.Mode().Perm()&0o111 == 0 {
			return appsettings.WebAppBrowser{}, errors.New("browser executable is not executable")
		}
		if preferredID == "" {
			return appsettings.WebAppBrowser{}, errors.New("select the browser type for the custom executable")
		}
	}
	if _, err := s.settings.Update(func(settings *appsettings.Data) error {
		settings.PreferredBrowserID = preferredID
		settings.CustomBrowserPath = customPath
		return nil
	}); err != nil {
		return appsettings.WebAppBrowser{}, err
	}
	return appsettings.WebAppBrowser{
		PreferredBrowserID: preferredID,
		CustomBrowserPath:  customPath,
	}, nil
}

func (s *PreferencesService) Resolve() (Browser, error) {
	settings, err := s.Load()
	if err != nil {
		return Browser{}, err
	}
	if settings.CustomBrowserPath != "" {
		return browserFromSettings(settings)
	}
	browsers := s.List()
	if settings.PreferredBrowserID != "" {
		for _, browser := range browsers {
			if string(browser.ID) == settings.PreferredBrowserID {
				return browser, nil
			}
		}
		return Browser{}, apperrors.New(
			"webapp_browser_missing",
			"The selected browser is not installed",
			map[string]string{"browser": settings.PreferredBrowserID},
		)
	}
	if len(browsers) == 0 {
		return Browser{}, apperrors.New("webapp_browser_missing", "No supported browser was found", nil)
	}
	return browsers[0], nil
}

func validID(id string) bool {
	switch ID(id) {
	case Firefox, Chrome, Chromium, Edge:
		return true
	default:
		return false
	}
}

func browserFromSettings(settings appsettings.WebAppBrowser) (Browser, error) {
	var browser Browser
	switch ID(settings.PreferredBrowserID) {
	case Firefox:
		browser = Browser{ID: Firefox, Name: "Firefox", Family: FamilyFirefox}
	case Chrome:
		browser = Browser{ID: Chrome, Name: "Google Chrome", Family: FamilyChromium}
	case Chromium:
		browser = Browser{ID: Chromium, Name: "Chromium", Family: FamilyChromium}
	case Edge:
		browser = Browser{ID: Edge, Name: "Microsoft Edge", Family: FamilyChromium}
	default:
		return Browser{}, errors.New("select the browser type for the custom executable")
	}
	browser.Path = settings.CustomBrowserPath
	return browser, nil
}

func expandHome(path string) string {
	path = strings.TrimSpace(path)
	if path != "~" && !strings.HasPrefix(path, "~/") && !strings.HasPrefix(path, `~\`) {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if path == "~" {
		return home
	}
	return filepath.Join(home, strings.TrimPrefix(strings.TrimPrefix(path, "~/"), `~\`))
}
