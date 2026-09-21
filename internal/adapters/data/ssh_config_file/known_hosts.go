// Copyright 2025.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ssh_config_file

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/WhiteRoseLK/neossh/internal/core/domain"
)

var hostLabelRegex = regexp.MustCompile(`^[A-Za-z0-9.-]+$`)

// ParseKnownHosts reads a known_hosts formatted stream and extracts unique discovered hosts.
// It handles plain hostnames/IPs, standard port notation like [host]:port,
// safely skips hashed entries (|1|...), comment lines, revoked keys, and invalid entries.
func ParseKnownHosts(r io.Reader) ([]domain.Server, error) {
	scanner := bufio.NewScanner(r)
	var discovered []domain.Server
	seenKeys := make(map[string]bool)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		hostFieldIndex := 0
		if strings.HasPrefix(fields[0], "@") {
			if strings.EqualFold(fields[0], "@revoked") {
				continue
			}
			hostFieldIndex = 1
		}

		if len(fields) < hostFieldIndex+3 {
			continue
		}
		if !isKnownHostKeyType(fields[hostFieldIndex+1]) {
			continue
		}

		hostField := fields[hostFieldIndex]
		if strings.HasPrefix(hostField, "|1|") {
			continue
		}

		tokens := strings.Split(hostField, ",")
		for _, token := range tokens {
			token = strings.TrimSpace(token)
			if token == "" || strings.HasPrefix(token, "|1|") {
				continue
			}

			server, ok := parseKnownHostToken(token)
			if !ok {
				continue
			}

			key := fmt.Sprintf("%s:%d", strings.ToLower(server.Host), server.Port)
			aliasKey := strings.ToLower(server.Alias)
			if seenKeys[key] || seenKeys[aliasKey] {
				continue
			}
			seenKeys[key] = true
			seenKeys[aliasKey] = true
			discovered = append(discovered, server)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return discovered, nil
}

func parseKnownHostToken(token string) (domain.Server, bool) {
	var host string
	port := 22

	if strings.HasPrefix(token, "[") {
		closeIdx := strings.Index(token, "]")
		if closeIdx == -1 {
			return domain.Server{}, false
		}
		host = token[1:closeIdx]
		remainder := token[closeIdx+1:]
		if strings.HasPrefix(remainder, ":") {
			p, err := strconv.Atoi(remainder[1:])
			if err != nil || p < 1 || p > 65535 {
				return domain.Server{}, false
			}
			port = p
		} else if remainder != "" {
			return domain.Server{}, false
		}

	} else {
		if strings.Count(token, ":") == 1 {
			parts := strings.Split(token, ":")
			p, err := strconv.Atoi(parts[1])
			if err == nil && p >= 1 && p <= 65535 {
				host = parts[0]
				port = p
			} else {
				host = token
			}
		} else {
			host = token
		}
	}

	host = strings.TrimSpace(host)
	if host == "" || strings.ContainsAny(host, "*?") || strings.Contains(host, " ") {
		return domain.Server{}, false
	}

	if !isValidHostOrIP(host) {
		return domain.Server{}, false
	}

	alias := deriveAlias(host, port)

	return domain.Server{
		Alias:         alias,
		Aliases:       []string{alias},
		Host:          host,
		Port:          port,
		IdentityFiles: []string{},
	}, true
}

func isKnownHostKeyType(k string) bool {
	k = strings.ToLower(k)
	return strings.HasPrefix(k, "ssh-") ||
		strings.HasPrefix(k, "ecdsa-") ||
		strings.HasPrefix(k, "sk-") ||
		strings.HasPrefix(k, "rsa-")
}

func isValidHostOrIP(host string) bool {
	if net.ParseIP(host) != nil {
		return true
	}
	if !hostLabelRegex.MatchString(host) {
		return false
	}
	if strings.HasPrefix(host, ".") || strings.HasSuffix(host, ".") {
		return false
	}
	for _, lbl := range strings.Split(host, ".") {
		if lbl == "" || strings.HasPrefix(lbl, "-") || strings.HasSuffix(lbl, "-") {
			return false
		}
	}
	return true
}

func deriveAlias(host string, port int) string {
	isIPv6 := strings.Contains(host, ":")
	var baseAlias string
	if isIPv6 {
		baseAlias = "ipv6-" + strings.ReplaceAll(host, ":", "-")
	} else {
		baseAlias = host
	}

	if port != 0 && port != 22 {
		return fmt.Sprintf("%s-%d", baseAlias, port)
	}
	return baseAlias
}

func isHostAlreadyConfigured(cand domain.Server, existing []domain.Server) bool {
	for _, s := range existing {
		if s.IsWildcardServer() {
			continue
		}

		if strings.EqualFold(s.Alias, cand.Alias) || strings.EqualFold(s.Alias, cand.Host) {
			return true
		}
		for _, a := range s.Aliases {
			if strings.EqualFold(a, cand.Alias) || strings.EqualFold(a, cand.Host) {
				return true
			}
		}

		effectiveHost := s.Host
		if effectiveHost == "" {
			effectiveHost = s.Alias
		}
		candPort := cand.Port
		if candPort == 0 {
			candPort = 22
		}
		sPort := s.Port
		if sPort == 0 {
			sPort = 22
		}

		if strings.EqualFold(effectiveHost, cand.Host) && sPort == candPort {
			return true
		}
	}
	return false
}

func resolveKnownHostsPath(path string) string {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return filepath.Join("~", ".ssh", "known_hosts")
		}
		return filepath.Join(home, ".ssh", "known_hosts")
	}
	return domain.ExpandTilde(path)
}

// DiscoverKnownHosts scans the specified known_hosts file, parses valid entries,
// and compares them against existing hosts in the SSH configuration.
// Returns unconfigured candidates and an ImportResult summary without altering files.
func (r *Repository) DiscoverKnownHosts(knownHostsPath string) ([]domain.Server, domain.ImportResult, error) {
	resolvedPath := resolveKnownHostsPath(knownHostsPath)

	file, err := r.fileSystem.Open(resolvedPath)
	if err != nil {
		if r.fileSystem.IsNotExist(err) || os.IsNotExist(err) {
			return nil, domain.ImportResult{}, fmt.Errorf("known_hosts file not found: %s", resolvedPath)
		}
		return nil, domain.ImportResult{}, fmt.Errorf("failed to read known_hosts file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, domain.ImportResult{}, fmt.Errorf("failed to read known_hosts file: %w", err)
	}

	discovered, err := ParseKnownHosts(bytes.NewReader(data))
	if err != nil {
		return nil, domain.ImportResult{}, fmt.Errorf("failed to parse known_hosts: %w", err)
	}

	lc, err := r.loadConfig()
	if err != nil {
		return nil, domain.ImportResult{}, fmt.Errorf("failed to load SSH config: %w", err)
	}
	existing := r.toDomainServer(lc)

	var candidates []domain.Server
	skipped := 0

	for _, srv := range discovered {
		if isHostAlreadyConfigured(srv, existing) {
			skipped++
			continue
		}
		candidates = append(candidates, srv)
	}

	result := domain.ImportResult{
		Discovered: len(discovered),
		Skipped:    skipped,
		Imported:   0,
	}

	return candidates, result, nil
}

// ImportKnownHosts discovers unconfigured hosts from known_hosts and appends them
// to the SSH configuration in a single atomic save operation.
func (r *Repository) ImportKnownHosts(knownHostsPath string) (domain.ImportResult, error) {
	candidates, result, err := r.DiscoverKnownHosts(knownHostsPath)
	if err != nil {
		return domain.ImportResult{}, err
	}

	if len(candidates) == 0 {
		return result, nil
	}

	if err := r.AddServers(candidates); err != nil {
		return domain.ImportResult{}, fmt.Errorf("failed to import hosts: %w", err)
	}

	result.Imported = len(candidates)
	return result, nil
}

// AddServers appends multiple servers to the main SSH config file in a single batch.
func (r *Repository) AddServers(servers []domain.Server) error {
	if len(servers) == 0 {
		return nil
	}

	lc, err := r.loadConfig()
	if err != nil {
		return err
	}

	if len(lc.files) == 0 {
		return fmt.Errorf("no target SSH config file found to import hosts")
	}
	target := &lc.files[0]

	for _, srv := range servers {
		if r.serverExists(lc, srv.Alias) {
			continue
		}
		host := r.createHostFromServer(srv)
		target.cfg.Hosts = append(target.cfg.Hosts, host)
	}

	if err := r.saveFiles(lc, []string{target.path}); err != nil {
		r.logger.Warnf("Failed to save config while adding new servers: %v", err)
		return fmt.Errorf("failed to save config: %w", err)
	}

	for _, srv := range servers {
		_ = r.metadataManager.updateServer(srv, srv.Alias)
	}

	return nil
}
