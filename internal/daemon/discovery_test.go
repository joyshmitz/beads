//go:build integration
// +build integration

package daemon

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/steveyegge/beads/internal/rpc"
	"github.com/steveyegge/beads/internal/storage/sqlite"
)

func TestDiscoverDaemon(t *testing.T) {
	tmpDir := t.TempDir()
	workspace := filepath.Join(tmpDir, ".beads")
	os.MkdirAll(workspace, 0755)

	// Start daemon
	dbPath := filepath.Join(workspace, "test.db")
	socketPath := filepath.Join(workspace, "bd.sock")
	store, err := sqlite.New(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer store.Close()

	server := rpc.NewServer(socketPath, store, tmpDir, dbPath)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go server.Start(ctx)
	<-server.WaitReady()
	defer server.Stop()

	// Test discoverDaemon directly
	daemon := discoverDaemon(socketPath)
	if !daemon.Alive {
		t.Errorf("daemon not alive: %s", daemon.Error)
	}
	if daemon.PID != os.Getpid() {
		t.Errorf("wrong PID: expected %d, got %d", os.Getpid(), daemon.PID)
	}
	if daemon.UptimeSeconds <= 0 {
		t.Errorf("invalid uptime: %f", daemon.UptimeSeconds)
	}
	if daemon.WorkspacePath != tmpDir {
		t.Errorf("wrong workspace: expected %s, got %s", tmpDir, daemon.WorkspacePath)
	}
}

func TestFindDaemonByWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	workspace := filepath.Join(tmpDir, ".beads")
	os.MkdirAll(workspace, 0755)

	// Start daemon
	dbPath := filepath.Join(workspace, "test.db")
	socketPath := filepath.Join(workspace, "bd.sock")
	store, err := sqlite.New(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer store.Close()

	server := rpc.NewServer(socketPath, store, tmpDir, dbPath)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go server.Start(ctx)
	<-server.WaitReady()
	defer server.Stop()

	// Find daemon by workspace
	daemon, err := FindDaemonByWorkspace(tmpDir)
	if err != nil {
		t.Fatalf("failed to find daemon: %v", err)
	}
	if daemon == nil {
		t.Fatal("daemon not found")
	}
	if !daemon.Alive {
		t.Errorf("daemon not alive: %s", daemon.Error)
	}
	if daemon.WorkspacePath != tmpDir {
		t.Errorf("wrong workspace: expected %s, got %s", tmpDir, daemon.WorkspacePath)
	}
}

func TestCleanupStaleSockets(t *testing.T) {
	tmpDir := t.TempDir()

	// Create stale socket file
	stalePath := filepath.Join(tmpDir, "stale.sock")
	if err := os.WriteFile(stalePath, []byte{}, 0644); err != nil {
		t.Fatalf("failed to create stale socket: %v", err)
	}

	daemons := []DaemonInfo{
		{
			SocketPath: stalePath,
			Alive:      false,
		},
	}

	cleaned, err := CleanupStaleSockets(daemons)
	if err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}
	if cleaned != 1 {
		t.Errorf("expected 1 cleaned, got %d", cleaned)
	}

	// Verify socket was removed
	if _, err := os.Stat(stalePath); !os.IsNotExist(err) {
		t.Error("stale socket still exists")
	}
}

func TestDiscoverDaemons_Legacy(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow daemon discovery test in short mode")
	}
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	os.MkdirAll(beadsDir, 0755)

	// Start a test daemon
	dbPath := filepath.Join(beadsDir, "test.db")
	socketPath := filepath.Join(beadsDir, "bd.sock")
	store, err := sqlite.New(context.Background(), dbPath)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	defer store.Close()

	server := rpc.NewServer(socketPath, store, tmpDir, dbPath)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go server.Start(ctx)
	<-server.WaitReady()
	defer server.Stop()

	// Test legacy discovery with explicit search roots
	daemons, err := DiscoverDaemons([]string{tmpDir})
	if err != nil {
		t.Fatalf("DiscoverDaemons failed: %v", err)
	}

	if len(daemons) != 1 {
		t.Fatalf("Expected 1 daemon, got %d", len(daemons))
	}

	daemon := daemons[0]
	if !daemon.Alive {
		t.Errorf("Daemon not alive: %s", daemon.Error)
	}
	if daemon.WorkspacePath != tmpDir {
		t.Errorf("Wrong workspace path: expected %s, got %s", tmpDir, daemon.WorkspacePath)
	}
}

func TestCheckDaemonErrorFile(t *testing.T) {
	tmpDir := t.TempDir()
	beadsDir := filepath.Join(tmpDir, ".beads")
	os.MkdirAll(beadsDir, 0755)
	socketPath := filepath.Join(beadsDir, "bd.sock")

	// Test 1: No error file exists
	errMsg := checkDaemonErrorFile(socketPath)
	if errMsg != "" {
		t.Errorf("Expected empty error message, got: %s", errMsg)
	}

	// Test 2: Error file exists with content
	errorFilePath := filepath.Join(beadsDir, "daemon-error")
	expectedError := "failed to start: database locked"
	os.WriteFile(errorFilePath, []byte(expectedError), 0644)

	errMsg = checkDaemonErrorFile(socketPath)
	if errMsg != expectedError {
		t.Errorf("Expected error message %q, got %q", expectedError, errMsg)
	}
}

