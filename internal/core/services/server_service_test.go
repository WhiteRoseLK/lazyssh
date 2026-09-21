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

package services

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"testing"

	"github.com/WhiteRoseLK/neossh/internal/core/domain"
	"github.com/WhiteRoseLK/neossh/internal/core/ports"
	"go.uber.org/zap"
)

type mockServerRepository struct {
	ports.ServerRepository
	recordCalls int
	lastAlias   string
	recordErr   error
}

func (m *mockServerRepository) ListServers(string) ([]domain.Server, error) {
	return nil, nil
}

func (m *mockServerRepository) UpdateServer(domain.Server, domain.Server) error { return nil }

func (m *mockServerRepository) AddServer(domain.Server) error { return nil }

func (m *mockServerRepository) AddServers([]domain.Server) error { return nil }

func (m *mockServerRepository) DiscoverKnownHosts(string) ([]domain.Server, domain.ImportResult, error) {
	return []domain.Server{{Alias: "srv1", Host: "1.1.1.1", Port: 22}}, domain.ImportResult{Discovered: 2, Skipped: 1}, nil
}

func (m *mockServerRepository) ImportKnownHosts(string) (domain.ImportResult, error) {
	return domain.ImportResult{Discovered: 2, Imported: 1, Skipped: 1}, nil
}

func (m *mockServerRepository) DeleteServer(domain.Server) error { return nil }

func (m *mockServerRepository) SetPinned(string, bool) error { return nil }

func (m *mockServerRepository) SetHidden(string, bool) error { return nil }

func (m *mockServerRepository) GetConfigFile() string { return "~/.ssh/config" }

func (m *mockServerRepository) GetTheme() (string, error) { return "dark", nil }

func (m *mockServerRepository) SaveTheme(string) error { return nil }

func (m *mockServerRepository) RecordSSH(alias string) error {
	m.recordCalls++
	m.lastAlias = alias
	return m.recordErr
}

func helperCommandFactory(scenario string) func(string) *exec.Cmd {
	return func(alias string) *exec.Cmd {
		cs := []string{"-test.run=TestHelperProcess", "--", scenario, alias}
		cmd := exec.Command(os.Args[0], cs...)
		cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
		cmd.Stdout = io.Discard
		cmd.Stderr = io.Discard
		return cmd
	}
}

func TestServerServiceSSH_RemoteDisconnect(t *testing.T) {
	repo := &mockServerRepository{}
	svc := &serverService{
		logger:           zap.NewNop().Sugar(),
		serverRepository: repo,
		newSSHCommand:    helperCommandFactory("remote"),
	}

	if err := svc.SSH("example"); err != nil {
		t.Fatalf("expected nil error on remote disconnect, got %v", err)
	}

	if repo.recordCalls != 1 {
		t.Fatalf("expected RecordSSH to be called once, got %d", repo.recordCalls)
	}
}

func TestServerServiceSSH_ConnectionReset(t *testing.T) {
	repo := &mockServerRepository{}
	svc := &serverService{
		logger:           zap.NewNop().Sugar(),
		serverRepository: repo,
		newSSHCommand:    helperCommandFactory("reset"),
	}

	if err := svc.SSH("example"); err != nil {
		t.Fatalf("expected nil error on connection reset, got %v", err)
	}

	if repo.recordCalls != 1 {
		t.Fatalf("expected RecordSSH to be called once, got %d", repo.recordCalls)
	}
}

func TestServerServiceSSH_PermissionDenied(t *testing.T) {
	repo := &mockServerRepository{}
	svc := &serverService{
		logger:           zap.NewNop().Sugar(),
		serverRepository: repo,
		newSSHCommand:    helperCommandFactory("permission"),
	}

	if err := svc.SSH("example"); err == nil {
		t.Fatalf("expected error on permission denied, got nil")
	} else if err.Error() != "Permission denied (publickey)." {
		t.Fatalf("expected error 'Permission denied (publickey).', got %q", err.Error())
	}

	if repo.recordCalls != 0 {
		t.Fatalf("expected RecordSSH not to be called on permission denied, got %d", repo.recordCalls)
	}
}

func TestServerServiceSSH_ConnectionRefused(t *testing.T) {
	repo := &mockServerRepository{}
	svc := &serverService{
		logger:           zap.NewNop().Sugar(),
		serverRepository: repo,
		newSSHCommand:    helperCommandFactory("refused"),
	}

	if err := svc.SSH("example"); err == nil {
		t.Fatalf("expected error on connection refused, got nil")
	} else if err.Error() != "ssh: connect to host example port 22: Connection refused" {
		t.Fatalf("expected error 'ssh: connect to host example port 22: Connection refused', got %q", err.Error())
	}

	if repo.recordCalls != 0 {
		t.Fatalf("expected RecordSSH not to be called on connection refused, got %d", repo.recordCalls)
	}
}

func TestServerServiceSSHWithArgs_Error(t *testing.T) {
	repo := &mockServerRepository{}
	svc := &serverService{
		logger:           zap.NewNop().Sugar(),
		serverRepository: repo,
		newSSHCommandWithArgs: func(alias string, extra []string) *exec.Cmd {
			cs := []string{"-test.run=TestHelperProcess", "--", "refused", alias}
			cmd := exec.Command(os.Args[0], cs...)
			cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
			cmd.Stdout = io.Discard
			cmd.Stderr = io.Discard
			return cmd
		},
	}

	if err := svc.SSHWithArgs("example", []string{"-L", "8080:localhost:80"}); err == nil {
		t.Fatalf("expected error on connection refused, got nil")
	} else if err.Error() != "ssh: connect to host example port 22: Connection refused" {
		t.Fatalf("expected error 'ssh: connect to host example port 22: Connection refused', got %q", err.Error())
	}

	if repo.recordCalls != 0 {
		t.Fatalf("expected RecordSSH not to be called on connection refused, got %d", repo.recordCalls)
	}
}

func TestServerServiceSSH_Success(t *testing.T) {
	repo := &mockServerRepository{}
	svc := &serverService{
		logger:           zap.NewNop().Sugar(),
		serverRepository: repo,
		newSSHCommand:    helperCommandFactory("success"),
	}

	if err := svc.SSH("example"); err != nil {
		t.Fatalf("expected nil error on success, got %v", err)
	}

	if repo.recordCalls != 1 {
		t.Fatalf("expected RecordSSH to be called once, got %d", repo.recordCalls)
	}
}

func TestServerServiceSSH_CommandFactoryReturnsNil(t *testing.T) {
	repo := &mockServerRepository{}
	svc := &serverService{
		logger:           zap.NewNop().Sugar(),
		serverRepository: repo,
		newSSHCommand: func(string) *exec.Cmd {
			return nil
		},
	}

	err := svc.SSH("example")
	if err == nil {
		t.Fatalf("expected error when command factory returns nil")
	}
}

func TestIsRemoteDisconnectError(t *testing.T) {
	if isRemoteDisconnectError(errors.New("plain error"), "Connection closed by remote host") {
		t.Fatalf("expected non-exit error to return false")
	}

	cmd := helperCommandFactory("remote")("example")
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected error for remote disconnect scenario")
	}
	if !isRemoteDisconnectError(err, "Connection to example closed by remote host.\n") {
		t.Fatalf("expected remote disconnect error to be detected")
	}

	cmd = helperCommandFactory("permission")("example")
	err = cmd.Run()
	if err == nil {
		t.Fatalf("expected error for permission scenario")
	}
	if isRemoteDisconnectError(err, "Permission denied (publickey).\n") {
		t.Fatalf("did not expect permission error to be treated as disconnect")
	}
}

func TestLimitedBuffer(t *testing.T) {
	buf := newLimitedBuffer(10)
	n, err := buf.Write([]byte("hello world 12345"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 17 {
		t.Fatalf("expected Write to report writing full slice length 17, got %d", n)
	}
	if buf.String() != "hello worl" {
		t.Fatalf("expected buffer to cap at 10 bytes 'hello worl', got %q", buf.String())
	}

	// Additional writes should be discarded
	n2, err := buf.Write([]byte("more"))
	if err != nil || n2 != 4 {
		t.Fatalf("unexpected error or length on saturated buffer")
	}
	if buf.String() != "hello worl" {
		t.Fatalf("buffer content should not change")
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	args := os.Args
	for i := 0; i < len(args); i++ {
		if args[i] == "--" && i+1 < len(args) {
			scenario := args[i+1]
			alias := ""
			if i+2 < len(args) {
				alias = args[i+2]
			}

			switch scenario {
			case "remote":
				_, _ = os.Stderr.WriteString("Connection to " + alias + " closed by remote host.\n")
				os.Exit(255)
			case "reset":
				_, _ = os.Stderr.WriteString("Read from remote host " + alias + ": Connection reset by peer\n")
				os.Exit(255)
			case "permission":
				_, _ = os.Stderr.WriteString("Permission denied (publickey).\n")
				os.Exit(255)
			case "refused":
				_, _ = os.Stderr.WriteString("ssh: connect to host " + alias + " port 22: Connection refused\n")
				os.Exit(255)
			case "success":
				os.Exit(0)
			default:
				_, _ = os.Stderr.WriteString("unknown scenario\n")
				os.Exit(1)
			}
		}
	}
	os.Exit(1)
}

func TestCopySSHKeyMissingBinary(t *testing.T) {
	// Temporarily empty PATH to simulate missing ssh-copy-id binary
	origPath := os.Getenv("PATH")
	t.Cleanup(func() {
		_ = os.Setenv("PATH", origPath)
	})
	_ = os.Setenv("PATH", "")

	logger := zap.NewNop().Sugar()
	svc := &serverService{
		logger: logger,
	}

	err := svc.CopySSHKey("test-server")
	if err == nil {
		t.Fatal("expected error when ssh-copy-id is not in PATH, got nil")
	}
}

type multiAliasRepo struct {
	ports.ServerRepository
	servers []domain.Server
}

func (m *multiAliasRepo) ListServers(query string) ([]domain.Server, error) {
	return m.servers, nil
}

func TestListServers_FindByMultipleAliases(t *testing.T) {
	repo := &multiAliasRepo{
		servers: []domain.Server{
			{
				Alias:   "web1",
				Aliases: []string{"web1", "web.prod.internal", "prod-web"},
				Host:    "10.0.0.1",
				User:    "admin",
			},
			{
				Alias:   "db1",
				Aliases: []string{"db1", "database.internal"},
				Host:    "10.0.0.2",
				User:    "postgres",
			},
		},
	}

	svc := &serverService{
		logger:           zap.NewNop().Sugar(),
		serverRepository: repo,
	}

	// Search by secondary alias "prod-web"
	res, err := svc.ListServers("prod-web")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) == 0 || res[0].Alias != "web1" {
		t.Fatalf("expected web1 to match 'prod-web', got %+v", res)
	}

	// Search by secondary alias "database.internal"
	res, err = svc.ListServers("database")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res) == 0 || res[0].Alias != "db1" {
		t.Fatalf("expected db1 to match 'database', got %+v", res)
	}
}

func TestServerService_ReadOnlyMode(t *testing.T) {
	logger := zap.NewNop().Sugar()
	mockRepo := &mockServerRepository{}

	svc := NewServerService(logger, mockRepo, WithReadOnly(true))

	testSrv := domain.Server{
		Alias: "new-srv",
		Host:  "5.6.7.8",
		User:  "admin",
		Port:  22,
	}
	existingSrv := domain.Server{
		Alias: "existing-srv",
		Host:  "1.2.3.4",
		User:  "root",
		Port:  22,
	}

	// AddServer should fail with ErrReadOnly
	if err := svc.AddServer(testSrv); !errors.Is(err, ErrReadOnly) {
		t.Fatalf("AddServer in readonly mode: expected ErrReadOnly, got %v", err)
	}

	// UpdateServer should fail with ErrReadOnly
	if err := svc.UpdateServer(existingSrv, testSrv); !errors.Is(err, ErrReadOnly) {
		t.Fatalf("UpdateServer in readonly mode: expected ErrReadOnly, got %v", err)
	}

	// DeleteServer should fail with ErrReadOnly
	if err := svc.DeleteServer(existingSrv); !errors.Is(err, ErrReadOnly) {
		t.Fatalf("DeleteServer in readonly mode: expected ErrReadOnly, got %v", err)
	}

	// CopySSHKey should fail with ErrReadOnly
	if err := svc.CopySSHKey("existing-srv"); !errors.Is(err, ErrReadOnly) {
		t.Fatalf("CopySSHKey in readonly mode: expected ErrReadOnly, got %v", err)
	}

	// ImportKnownHosts should fail with ErrReadOnly
	if _, err := svc.ImportKnownHosts("/dummy/known_hosts"); !errors.Is(err, ErrReadOnly) {
		t.Fatalf("ImportKnownHosts in readonly mode: expected ErrReadOnly, got %v", err)
	}

	// SetHidden should fail with ErrReadOnly
	if err := svc.SetHidden("existing-srv", true); !errors.Is(err, ErrReadOnly) {
		t.Fatalf("SetHidden in readonly mode: expected ErrReadOnly, got %v", err)
	}

	// DiscoverKnownHosts should succeed in readonly mode (non-modifying)
	discovered, res, err := svc.DiscoverKnownHosts("/dummy/known_hosts")
	if err != nil {
		t.Fatalf("DiscoverKnownHosts in readonly mode failed: %v", err)
	}
	if len(discovered) != 1 || res.Discovered != 2 {
		t.Fatalf("unexpected DiscoverKnownHosts result: %+v, %+v", discovered, res)
	}

	// Non-modifying operations should succeed
	servers, err := svc.ListServers("")
	if err != nil {
		t.Fatalf("ListServers failed in readonly mode: %v", err)
	}
	if servers != nil {
		t.Fatalf("expected nil servers from mock, got %v", servers)
	}
}

func TestServerService_ImportKnownHosts_NormalMode(t *testing.T) {
	logger := zap.NewNop().Sugar()
	mockRepo := &mockServerRepository{}
	svc := NewServerService(logger, mockRepo, WithReadOnly(false))

	res, err := svc.ImportKnownHosts("/dummy/known_hosts")
	if err != nil {
		t.Fatalf("ImportKnownHosts failed: %v", err)
	}
	if res.Discovered != 2 || res.Imported != 1 || res.Skipped != 1 {
		t.Fatalf("unexpected ImportResult: %+v", res)
	}
}

func TestServerService_SetHidden_NormalMode(t *testing.T) {
	logger := zap.NewNop().Sugar()
	mockRepo := &mockServerRepository{}
	svc := NewServerService(logger, mockRepo, WithReadOnly(false))

	if err := svc.SetHidden("existing-srv", true); err != nil {
		t.Fatalf("SetHidden failed in normal mode: %v", err)
	}
}
