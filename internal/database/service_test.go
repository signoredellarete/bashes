package database

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/signoredellarete/bashes/internal/application"
	"github.com/signoredellarete/bashes/internal/domain"
	"github.com/signoredellarete/bashes/internal/store"
)

func TestServiceExportsAndImportsPortableStore(t *testing.T) {
	root := t.TempDir()
	sourceApplication := application.NewService(store.NewRepository(filepath.Join(root, "source", "hosts.json")))
	source := NewService(sourceApplication)
	if _, err := sourceApplication.AddHost(application.EndpointInput{
		Hostname: "server", IP: "192.0.2.10", Port: 22, User: "deploy",
	}); err != nil {
		t.Fatal(err)
	}
	exportPath := filepath.Join(root, "backup.json")
	if err := source.Export(exportPath); err != nil {
		t.Fatalf("Export() error = %v", err)
	}

	targetApplication := application.NewService(store.NewRepository(filepath.Join(root, "target", "hosts.json")))
	target := NewService(targetApplication)
	if err := target.Import(exportPath); err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	hosts, err := targetApplication.ListHosts()
	if err != nil || len(hosts) != 1 || hosts[0].Hostname != "server" {
		t.Fatalf("imported hosts = %+v, error = %v", hosts, err)
	}
}

func TestServicePreviewHostsFileDoesNotPersist(t *testing.T) {
	root := t.TempDir()
	applicationService := application.NewService(store.NewRepository(filepath.Join(root, "data", "hosts.json")))
	service := NewService(applicationService)
	hostsPath := filepath.Join(root, "hosts")
	if err := os.WriteFile(hostsPath, []byte("192.0.2.20 preview-host\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	result, preview, err := service.PreviewHostsFile(hostsPath, "deploy", "")
	if err != nil {
		t.Fatalf("PreviewHostsFile() error = %v", err)
	}
	if result.Imported != 1 || len(preview.Hosts) != 1 {
		t.Fatalf("preview result = %+v, store = %+v", result, preview)
	}
	hosts, err := applicationService.ListHosts()
	if err != nil || len(hosts) != 0 {
		t.Fatalf("persisted hosts = %+v, error = %v", hosts, err)
	}
}

func TestServiceImportHostsFileSkipsExistingAddress(t *testing.T) {
	root := t.TempDir()
	applicationService := application.NewService(store.NewRepository(filepath.Join(root, "data", "hosts.json")))
	service := NewService(applicationService)
	if _, err := applicationService.AddHost(application.EndpointInput{
		Hostname: "existing", IP: "192.0.2.30", Port: 22, User: "deploy",
	}); err != nil {
		t.Fatal(err)
	}
	hostsPath := filepath.Join(root, "hosts")
	if err := os.WriteFile(hostsPath, []byte("192.0.2.30 duplicate\n192.0.2.31 new-host\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	result, err := service.ImportHostsFile(hostsPath, "deploy", "")
	if err != nil {
		t.Fatalf("ImportHostsFile() error = %v", err)
	}
	if result.Imported != 1 || result.Skipped != 1 {
		t.Fatalf("result = %+v", result)
	}
	data, err := applicationService.StoreSnapshot()
	if err != nil || len(domain.NormalizeStore(data).Hosts) != 2 {
		t.Fatalf("store = %+v, error = %v", data, err)
	}
}
