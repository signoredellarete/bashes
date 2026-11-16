package plugin

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	appsettings "github.com/signoredellarete/bashes/internal/settings"
)

func TestRegistryEnablesOnlyAvailablePlugins(t *testing.T) {
	root := t.TempDir()
	createDiscoveryFixture(t, root, "available", HostAPIVersion, true)
	createDiscoveryFixture(t, root, "incompatible", HostAPIVersion+1, true)
	registry := NewRegistry(root, runtime.GOOS, runtime.GOARCH, appsettings.NewRepository(filepath.Join(t.TempDir(), "settings.json")))

	info, err := registry.SetEnabled("org.bashes.available", true)
	if err != nil {
		t.Fatalf("SetEnabled(available) error = %v", err)
	}
	if !info.Enabled || !info.Available {
		t.Fatalf("SetEnabled(available) = %#v", info)
	}
	if _, err := registry.SetEnabled("org.bashes.incompatible", true); err == nil || !strings.Contains(err.Error(), "incompatible") {
		t.Fatalf("SetEnabled(incompatible) error = %v", err)
	}
	if _, err := registry.SetEnabled("org.bashes.missing", true); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("SetEnabled(missing) error = %v", err)
	}
}

func TestRegistryReportsAndDisablesMissingEnabledPlugin(t *testing.T) {
	settings := appsettings.NewRepository(filepath.Join(t.TempDir(), "settings.json"))
	if _, err := settings.Update(func(data *appsettings.Data) error {
		data.EnabledPlugins = []string{"org.bashes.missing"}
		return nil
	}); err != nil {
		t.Fatalf("seed settings: %v", err)
	}
	registry := NewRegistry(t.TempDir(), runtime.GOOS, runtime.GOARCH, settings)

	inventory, err := registry.Inventory()
	if err != nil {
		t.Fatalf("Inventory() error = %v", err)
	}
	if len(inventory.Plugins) != 1 || !inventory.Plugins[0].Enabled || inventory.Plugins[0].Available {
		t.Fatalf("Inventory() = %#v", inventory)
	}
	if _, err := registry.SetEnabled("org.bashes.missing", false); err != nil {
		t.Fatalf("SetEnabled(missing, false) error = %v", err)
	}
	inventory, err = registry.Inventory()
	if err != nil || len(inventory.Plugins) != 0 {
		t.Fatalf("Inventory(after disable) = %#v, %v", inventory, err)
	}
}
