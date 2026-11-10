package sshkeys

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	appsettings "github.com/signoredellarete/bashes/internal/settings"
	"github.com/signoredellarete/bashes/internal/sshauth"
	"golang.org/x/crypto/ssh"
)

type Key struct {
	Name         string `json:"name"`
	PrivateKey   string `json:"privateKey"`
	PublicKey    string `json:"publicKey"`
	Source       string `json:"source"`
	Distribution string `json:"distribution,omitempty"`
}

type Source struct {
	Source       string `json:"source"`
	Label        string `json:"label"`
	Path         string `json:"path"`
	Count        int    `json:"count"`
	Distribution string `json:"distribution,omitempty"`
	Error        string `json:"error,omitempty"`
}

type Inventory struct {
	Keys    []Key    `json:"keys"`
	Sources []Source `json:"sources"`
}

type GenerateInput struct {
	Name string `json:"name"`
}

type ImportInput struct {
	Name       string `json:"name"`
	PrivateKey string `json:"privateKey"`
	PublicKey  string `json:"publicKey"`
}

type AdditionalInventory func() ([]Key, []Source)

type Service struct {
	managedDir          string
	settings            *appsettings.Repository
	homeDir             func() string
	additionalInventory AdditionalInventory
}

func NewService(managedDir string, settings *appsettings.Repository, additional AdditionalInventory) *Service {
	return &Service{
		managedDir:          managedDir,
		settings:            settings,
		homeDir:             userHomeDir,
		additionalInventory: additional,
	}
}

func (s *Service) ManagedDir() string {
	return s.managedDir
}

func (s *Service) Settings() (appsettings.SSHKeys, error) {
	settings, err := s.settings.Load()
	if err != nil {
		return appsettings.SSHKeys{}, err
	}
	return appsettings.SSHKeys{CustomDirectory: settings.CustomDirectory}, nil
}

func (s *Service) SaveSettings(input appsettings.SSHKeys) (appsettings.SSHKeys, error) {
	customDirectory := strings.TrimSpace(input.CustomDirectory)
	if customDirectory != "" {
		dir := expandHome(customDirectory, s.homeDir())
		info, err := os.Stat(dir)
		if err != nil {
			return appsettings.SSHKeys{}, fmt.Errorf("read custom keys directory: %w", err)
		}
		if !info.IsDir() {
			return appsettings.SSHKeys{}, fmt.Errorf("custom keys path is not a directory: %s", customDirectory)
		}
		customDirectory = dir
	}
	if _, err := s.settings.Update(func(settings *appsettings.Data) error {
		settings.CustomDirectory = customDirectory
		return nil
	}); err != nil {
		return appsettings.SSHKeys{}, err
	}
	return appsettings.SSHKeys{CustomDirectory: customDirectory}, nil
}

func (s *Service) Inventory() (Inventory, error) {
	settings, err := s.Settings()
	if err != nil {
		return Inventory{}, err
	}

	inventory := Inventory{Keys: []Key{}, Sources: []Source{}}
	managedKeys, managedSource := ScanSource(s.managedDir, "bashes", "Managed by Bashes", true)
	inventory.Keys = append(inventory.Keys, managedKeys...)
	inventory.Sources = append(inventory.Sources, managedSource)

	systemPath := ""
	if home := s.homeDir(); home != "" {
		systemPath = filepath.Join(home, ".ssh")
	}
	systemKeys, systemSource := ScanSource(systemPath, "system", "System SSH directory", true)
	inventory.Keys = append(inventory.Keys, systemKeys...)
	inventory.Sources = append(inventory.Sources, systemSource)

	if s.additionalInventory != nil {
		keys, sources := s.additionalInventory()
		inventory.Keys = append(inventory.Keys, keys...)
		inventory.Sources = append(inventory.Sources, sources...)
	}

	customKeys, customSource := ScanSource(settings.CustomDirectory, "custom", "Custom directory", false)
	inventory.Keys = append(inventory.Keys, customKeys...)
	inventory.Sources = append(inventory.Sources, customSource)

	inventory.Keys = dedupe(inventory.Keys)
	sort.SliceStable(inventory.Keys, func(i, j int) bool {
		if inventory.Keys[i].Source != inventory.Keys[j].Source {
			return inventory.Keys[i].Source < inventory.Keys[j].Source
		}
		return strings.ToLower(inventory.Keys[i].Name) < strings.ToLower(inventory.Keys[j].Name)
	})
	return inventory, nil
}

func ScanSource(dir string, source string, label string, ignoreMissing bool) ([]Key, Source) {
	dir = expandHome(strings.TrimSpace(dir), userHomeDir())
	info := Source{Source: source, Label: label, Path: dir}
	if dir == "" {
		return []Key{}, info
	}
	keys, err := ListDirectory(dir, source, ignoreMissing)
	if err != nil {
		info.Error = err.Error()
		return []Key{}, info
	}
	info.Count = len(keys)
	return keys, info
}

func ListDirectory(dir string, source string, ignoreMissing bool) ([]Key, error) {
	dir = expandHome(dir, userHomeDir())
	entries, err := os.ReadDir(dir)
	if err != nil {
		if ignoreMissing && errors.Is(err, os.ErrNotExist) {
			return []Key{}, nil
		}
		return nil, err
	}

	keys := []Key{}
	for _, entry := range entries {
		if entry.IsDir() || strings.HasSuffix(entry.Name(), ".pub") {
			continue
		}
		privatePath := filepath.Join(dir, entry.Name())
		publicPath := privatePath + ".pub"
		hasPublic := false
		if _, err := os.Stat(publicPath); err == nil {
			hasPublic = true
		}
		if !hasPublic && !isLikelyPrivateKeyName(entry.Name()) {
			continue
		}
		if !hasPublic {
			publicPath = ""
		}
		keys = append(keys, Key{
			Name:       entry.Name(),
			PrivateKey: privatePath,
			PublicKey:  publicPath,
			Source:     source,
		})
	}
	return keys, nil
}

func (s *Service) Generate(input GenerateInput) (Key, error) {
	name := SanitizeName(input.Name)
	hasExplicitName := name != ""
	if err := os.MkdirAll(s.managedDir, 0o700); err != nil {
		return Key{}, err
	}
	if !hasExplicitName {
		name = nextAvailableName(s.managedDir, "bashes")
	}

	privatePath := filepath.Join(s.managedDir, name)
	publicPath := privatePath + ".pub"
	if fileExists(privatePath) || fileExists(publicPath) {
		return Key{}, fmt.Errorf("ssh key %q already exists", name)
	}

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return Key{}, fmt.Errorf("generate ed25519 key: %w", err)
	}
	block, err := ssh.MarshalPrivateKey(privateKey, fmt.Sprintf("bashes:%s", name))
	if err != nil {
		return Key{}, fmt.Errorf("marshal private key: %w", err)
	}
	sshPublicKey, err := ssh.NewPublicKey(publicKey)
	if err != nil {
		return Key{}, fmt.Errorf("marshal public key: %w", err)
	}
	if err := WritePairAtomic(
		privatePath,
		pem.EncodeToMemory(block),
		publicPath,
		ssh.MarshalAuthorizedKey(sshPublicKey),
	); err != nil {
		return Key{}, err
	}
	return Key{Name: name, PrivateKey: privatePath, PublicKey: publicPath, Source: "bashes"}, nil
}

func (s *Service) ImportPair(input ImportInput) (Key, error) {
	privateData := appendTrailingNewline([]byte(strings.TrimSpace(input.PrivateKey)))
	publicData := appendTrailingNewline([]byte(strings.TrimSpace(input.PublicKey)))
	if len(bytes.TrimSpace(privateData)) == 0 {
		return Key{}, errors.New("private key is required")
	}
	if len(bytes.TrimSpace(publicData)) == 0 {
		return Key{}, errors.New("public key is required")
	}
	if err := validatePair(privateData, publicData); err != nil {
		return Key{}, err
	}
	if err := os.MkdirAll(s.managedDir, 0o700); err != nil {
		return Key{}, err
	}
	name := SanitizeName(input.Name)
	if name == "" {
		name = nextAvailableName(s.managedDir, "imported-key")
	}
	privatePath := filepath.Join(s.managedDir, name)
	publicPath := privatePath + ".pub"
	if fileExists(privatePath) || fileExists(publicPath) {
		return Key{}, fmt.Errorf("ssh key %q already exists", name)
	}
	if err := WritePairAtomic(privatePath, privateData, publicPath, publicData); err != nil {
		return Key{}, err
	}
	return Key{Name: name, PrivateKey: privatePath, PublicKey: publicPath, Source: "bashes"}, nil
}

func (s *Service) Delete(name string) error {
	name = strings.TrimSpace(name)
	if name == "" || SanitizeName(name) != name {
		return errors.New("invalid managed SSH key name")
	}
	paths := []string{
		filepath.Join(s.managedDir, name),
		filepath.Join(s.managedDir, name+".pub"),
	}
	existing := make([]string, 0, len(paths))
	for _, path := range paths {
		if _, err := os.Lstat(path); err == nil {
			existing = append(existing, path)
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspect managed SSH key: %w", err)
		}
	}
	if len(existing) == 0 {
		return fmt.Errorf("managed SSH key %q does not exist", name)
	}

	suffix := fmt.Sprintf(".deleting-%d", time.Now().UnixNano())
	renamed := make([][2]string, 0, len(existing))
	for _, path := range existing {
		temporaryPath := path + suffix
		if err := os.Rename(path, temporaryPath); err != nil {
			for index := len(renamed) - 1; index >= 0; index-- {
				_ = os.Rename(renamed[index][1], renamed[index][0])
			}
			return fmt.Errorf("prepare managed SSH key deletion: %w", err)
		}
		renamed = append(renamed, [2]string{path, temporaryPath})
	}
	for _, pair := range renamed {
		if err := os.Remove(pair[1]); err != nil {
			return fmt.Errorf("delete managed SSH key: %w", err)
		}
	}
	return nil
}

func (s *Service) ReadManagedPublic(name string) (string, error) {
	return ReadPublicPath(filepath.Join(s.managedDir, SanitizeName(name)+".pub"))
}

func (s *Service) ReadManagedPrivate(name string) (string, error) {
	return ReadPrivatePath(filepath.Join(s.managedDir, SanitizeName(name)))
}

func (s *Service) ManagedPrivatePath(name string) string {
	return filepath.Join(s.managedDir, SanitizeName(name))
}

func ReadPrivatePath(path string) (string, error) {
	path = expandHome(path, userHomeDir())
	if strings.TrimSpace(path) == "" {
		return "", errors.New("ssh private key path is empty")
	}
	if strings.HasSuffix(strings.ToLower(path), ".pub") {
		return "", errors.New("the selected path points to a public key")
	}
	key, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(key), nil
}

func ReadPublicPath(path string) (string, error) {
	path = expandHome(path, userHomeDir())
	if strings.TrimSpace(path) == "" {
		return "", errors.New("ssh public key path is empty")
	}
	if strings.HasSuffix(path, ".pub") {
		key, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(key), nil
	}

	publicPath := path + ".pub"
	key, err := os.ReadFile(publicPath)
	if err == nil {
		return string(key), nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}

	privateKey, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	publicKey, _, err := publicKeyFromPrivateData(privateKey)
	if err != nil {
		return "", fmt.Errorf("read public key from private key: %w", err)
	}
	if publicKey == nil {
		return "", errors.New("read public key from private key: encrypted key does not expose its public key")
	}
	return string(ssh.MarshalAuthorizedKey(publicKey)), nil
}

func ExpandPath(path string) string {
	return expandHome(path, userHomeDir())
}

func SanitizeName(name string) string {
	name = strings.TrimSpace(name)
	name = regexp.MustCompile(`[^A-Za-z0-9_.-]+`).ReplaceAllString(name, "-")
	return strings.Trim(name, ".-")
}

func WritePairAtomic(privatePath string, privateData []byte, publicPath string, publicData []byte) error {
	dir := filepath.Dir(privatePath)
	privateTemp, err := writeTempFile(dir, filepath.Base(privatePath), privateData, 0o600)
	if err != nil {
		return fmt.Errorf("prepare private key: %w", err)
	}
	defer os.Remove(privateTemp)
	publicTemp, err := writeTempFile(dir, filepath.Base(publicPath), publicData, 0o644)
	if err != nil {
		return fmt.Errorf("prepare public key: %w", err)
	}
	defer os.Remove(publicTemp)

	if err := publishFile(privateTemp, privatePath, privateData, 0o600); err != nil {
		return fmt.Errorf("publish private key: %w", err)
	}
	if err := publishFile(publicTemp, publicPath, publicData, 0o644); err != nil {
		_ = os.Remove(privatePath)
		return fmt.Errorf("publish public key: %w", err)
	}
	return nil
}

func PublishFileWithLink(tempPath string, path string, data []byte, perm os.FileMode, link func(string, string) error) error {
	if err := link(tempPath, path); err == nil {
		return nil
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	cleanup := func() {
		_ = file.Close()
		_ = os.Remove(path)
	}
	if _, err := file.Write(data); err != nil {
		cleanup()
		return err
	}
	if err := file.Sync(); err != nil {
		cleanup()
		return err
	}
	if err := file.Chmod(perm); err != nil {
		cleanup()
		return err
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return err
	}
	return nil
}

func publishFile(tempPath string, path string, data []byte, perm os.FileMode) error {
	return PublishFileWithLink(tempPath, path, data, perm, os.Link)
}

func writeTempFile(dir string, base string, data []byte, perm os.FileMode) (string, error) {
	tmp, err := os.CreateTemp(dir, "."+base+".*.tmp")
	if err != nil {
		return "", fmt.Errorf("create temporary file: %w", err)
	}
	name := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(name)
	}
	if _, err := tmp.Write(data); err != nil {
		cleanup()
		return "", fmt.Errorf("write temporary file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		cleanup()
		return "", fmt.Errorf("sync temporary file: %w", err)
	}
	if err := tmp.Chmod(perm); err != nil {
		cleanup()
		return "", fmt.Errorf("chmod temporary file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(name)
		return "", fmt.Errorf("close temporary file: %w", err)
	}
	return name, nil
}

func validatePair(privateData, publicData []byte) error {
	publicKey, _, _, rest, err := ssh.ParseAuthorizedKey(publicData)
	if err != nil {
		return fmt.Errorf("parse public key: %w", err)
	}
	if len(bytes.TrimSpace(rest)) != 0 {
		return errors.New("public key contains more than one key")
	}
	privatePublicKey, comparable, err := publicKeyFromPrivateData(privateData)
	if err != nil {
		return err
	}
	if comparable && !bytes.Equal(privatePublicKey.Marshal(), publicKey.Marshal()) {
		return errors.New("public key does not match the private key")
	}
	return nil
}

func publicKeyFromPrivateData(privateData []byte) (ssh.PublicKey, bool, error) {
	return sshauth.PublicKeyFromPrivateData(privateData)
}

func dedupe(keys []Key) []Key {
	seen := map[string]struct{}{}
	deduped := make([]Key, 0, len(keys))
	for _, key := range keys {
		identity := key.Source + ":" + key.Name
		if key.PrivateKey != "" {
			identity = filepath.Clean(expandHome(key.PrivateKey, userHomeDir()))
		}
		if _, exists := seen[identity]; exists {
			continue
		}
		seen[identity] = struct{}{}
		deduped = append(deduped, key)
	}
	return deduped
}

func isLikelyPrivateKeyName(name string) bool {
	lower := strings.ToLower(name)
	if strings.HasPrefix(lower, "known_hosts") {
		return false
	}
	switch lower {
	case "config", "authorized_keys", "allowed_signers", "revoked_keys", "environment":
		return false
	}
	return strings.HasPrefix(lower, "id_") || strings.HasSuffix(lower, ".ppk")
}

func nextAvailableName(dir string, base string) string {
	name := SanitizeName(base)
	if name == "" {
		name = "bashes"
	}
	if !fileExists(filepath.Join(dir, name)) && !fileExists(filepath.Join(dir, name)+".pub") {
		return name
	}
	for index := 2; ; index++ {
		candidate := fmt.Sprintf("%s-%d", name, index)
		if !fileExists(filepath.Join(dir, candidate)) && !fileExists(filepath.Join(dir, candidate)+".pub") {
			return candidate
		}
	}
}

func appendTrailingNewline(data []byte) []byte {
	data = bytes.TrimSpace(data)
	if len(data) == 0 {
		return nil
	}
	return append(data, '\n')
}

func fileExists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func expandHome(path string, home string) string {
	path = strings.TrimSpace(path)
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, strings.TrimPrefix(path, "~/"))
	}
	if strings.HasPrefix(path, `~\`) {
		return filepath.Join(home, strings.TrimPrefix(path, `~\`))
	}
	return path
}

func userHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}
