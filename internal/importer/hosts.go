package importer

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/signoredellarete/bashes/internal/domain"
)

type HostsFileResult struct {
	Path      string            `json:"path"`
	Imported  int               `json:"imported"`
	Skipped   int               `json:"skipped"`
	User      string            `json:"user"`
	Hostnames []string          `json:"hostnames"`
	Hosts     []HostsFileTarget `json:"hosts"`
}

type HostsFileTarget struct {
	Hostname string `json:"hostname"`
	IP       string `json:"ip"`
	User     string `json:"user"`
	Port     int    `json:"port"`
}

type hostEntry struct {
	IP       string
	Hostname string
}

type sshHostConfig struct {
	aliases  []string
	hostname string
	user     string
	port     int
}

type sshConfig []sshHostConfig

func PrepareHostsFile(path, user, sshConfigPath string, data domain.Store) (HostsFileResult, domain.Store, error) {
	result := HostsFileResult{Path: path, User: user}
	if strings.TrimSpace(path) == "" {
		return result, domain.Store{}, errors.New("hosts file path is required")
	}
	if strings.TrimSpace(user) == "" {
		return result, domain.Store{}, errors.New("default SSH user is required")
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return result, domain.Store{}, fmt.Errorf("read hosts file: %w", err)
	}
	entries := parseHostsFile(raw)
	config := loadSSHConfig(sshConfigPath)
	seenHosts, seenIPs := existingHostKeys(data.Hosts)

	for _, entry := range entries {
		hostnameKey := strings.ToLower(entry.Hostname)
		ipKey := strings.ToLower(entry.IP)
		if _, exists := seenHosts[hostnameKey]; exists {
			result.Skipped++
			continue
		}
		if _, exists := seenIPs[ipKey]; exists {
			result.Skipped++
			continue
		}

		host := domain.Host{
			Hostname:   entry.Hostname,
			IP:         entry.IP,
			Port:       22,
			User:       user,
			Subsystems: []domain.Endpoint{},
		}
		if override, ok := config.match(entry); ok {
			if override.user != "" {
				host.User = override.user
			}
			if override.port > 0 {
				host.Port = override.port
			}
		}
		host.ID = domain.StableID(domain.ResourceHost, host.Hostname, host.IP, host.Port, host.User, len(data.Hosts))
		data.Hosts = append(data.Hosts, host)
		seenHosts[hostnameKey] = struct{}{}
		seenIPs[ipKey] = struct{}{}
		result.Imported++
		result.Hostnames = append(result.Hostnames, host.Hostname)
		result.Hosts = append(result.Hosts, HostsFileTarget{
			Hostname: host.Hostname,
			IP:       host.IP,
			User:     host.User,
			Port:     host.Port,
		})
	}
	return result, data, nil
}

func parseHostsFile(data []byte) []hostEntry {
	var entries []hostEntry
	seen := map[string]struct{}{}
	for _, line := range strings.Split(string(data), "\n") {
		if cut := strings.Index(line, "#"); cut >= 0 {
			line = line[:cut]
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		ip := net.ParseIP(fields[0])
		if skipHostsFileIP(ip) {
			continue
		}
		for _, hostname := range fields[1:] {
			hostname = strings.TrimSpace(hostname)
			if skipHostsFileHostname(hostname) {
				continue
			}
			key := strings.ToLower(ip.String() + "|" + hostname)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			entries = append(entries, hostEntry{IP: ip.String(), Hostname: hostname})
			break
		}
	}
	return entries
}

func loadSSHConfig(path string) sshConfig {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return parseSSHConfig(data)
}

func parseSSHConfig(data []byte) sshConfig {
	var result sshConfig
	var current *sshHostConfig
	flush := func() {
		if current != nil && len(current.aliases) > 0 {
			result = append(result, *current)
		}
	}

	for _, line := range strings.Split(string(data), "\n") {
		if cut := strings.Index(line, "#"); cut >= 0 {
			line = line[:cut]
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		switch strings.ToLower(fields[0]) {
		case "host":
			flush()
			current = &sshHostConfig{}
			for _, alias := range fields[1:] {
				if isExplicitSSHHostAlias(alias) {
					current.aliases = append(current.aliases, alias)
				}
			}
			if len(current.aliases) == 0 {
				current = nil
			}
		case "match":
			flush()
			current = nil
		case "hostname":
			if current != nil && len(fields) > 1 && isExplicitSSHHostAlias(fields[1]) {
				current.hostname = fields[1]
			}
		case "user":
			if current != nil && len(fields) > 1 {
				current.user = fields[1]
			}
		case "port":
			if current != nil && len(fields) > 1 {
				port, err := strconv.Atoi(fields[1])
				if err == nil && port > 0 && port <= 65535 {
					current.port = port
				}
			}
		}
	}
	flush()
	return result
}

func isExplicitSSHHostAlias(alias string) bool {
	alias = strings.TrimSpace(alias)
	if alias == "" || strings.HasPrefix(alias, "!") {
		return false
	}
	return !strings.ContainsAny(alias, "*?[]{}")
}

func (config sshConfig) match(entry hostEntry) (sshHostConfig, bool) {
	for _, item := range config {
		if item.matches(entry) {
			return item, true
		}
	}
	return sshHostConfig{}, false
}

func (item sshHostConfig) matches(entry hostEntry) bool {
	candidates := []string{
		strings.ToLower(strings.TrimSpace(entry.Hostname)),
		strings.ToLower(strings.TrimSpace(entry.IP)),
	}
	for _, candidate := range candidates {
		if candidate == "" {
			continue
		}
		for _, alias := range item.aliases {
			if strings.ToLower(strings.TrimSpace(alias)) == candidate {
				return true
			}
		}
		if strings.ToLower(strings.TrimSpace(item.hostname)) == candidate {
			return true
		}
	}
	return false
}

func skipHostsFileIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	return ip.IsLoopback() || ip.IsUnspecified() || ip.IsMulticast() ||
		ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.Equal(net.IPv4bcast)
}

func skipHostsFileHostname(hostname string) bool {
	hostname = strings.TrimSpace(strings.ToLower(hostname))
	if hostname == "" || hostname == "localhost" || hostname == "localhost.localdomain" {
		return true
	}
	if strings.ContainsAny(hostname, "\r\n\t ") {
		return true
	}
	return net.ParseIP(hostname) != nil
}

func existingHostKeys(hosts []domain.Host) (map[string]struct{}, map[string]struct{}) {
	hostnames := map[string]struct{}{}
	ips := map[string]struct{}{}
	for _, host := range hosts {
		addHostKey(hostnames, host.Hostname)
		addHostKey(ips, host.IP)
		addEndpointKeys(hostnames, ips, host.Subsystems)
	}
	return hostnames, ips
}

func addEndpointKeys(hostnames, ips map[string]struct{}, endpoints []domain.Endpoint) {
	for _, endpoint := range endpoints {
		addHostKey(hostnames, endpoint.Hostname)
		addHostKey(ips, endpoint.IP)
		addEndpointKeys(hostnames, ips, endpoint.Subsystems)
	}
}

func addHostKey(values map[string]struct{}, value string) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value != "" {
		values[value] = struct{}{}
	}
}
