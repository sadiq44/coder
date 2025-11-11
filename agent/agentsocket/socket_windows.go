//go:build windows

package agentsocket

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/hashicorp/yamux"
	"golang.org/x/xerrors"

	"cdr.dev/slog"
	"github.com/coder/coder/v2/agent/agentsocket/proto"
	"github.com/coder/coder/v2/codersdk/drpcsdk"
)

// createSocket creates a Unix domain socket listener on Windows
// Falls back to named pipe if Unix sockets are not supported
func createSocket(path string) (net.Listener, error) {
	// Try Unix domain socket first (Windows 10 build 17063+)
	listener, err := net.Listen("unix", path)
	if err == nil {
		return listener, nil
	}

	// Fall back to named pipe
	pipePath := `\\.\pipe\coder-agent`
	listener, err = net.Listen("tcp", pipePath)
	if err != nil {
		return nil, err
	}
	return listener, nil
}

// getDefaultSocketPath returns the default socket path for Windows
func getDefaultSocketPath() (string, error) {
	// Try to use a temporary directory
	tempDir := os.TempDir()
	if tempDir == "" {
		tempDir = "C:\\temp"
	}

	// Create a user-specific subdirectory
	uid := os.Getuid()
	userDir := filepath.Join(tempDir, "coder-agent", strconv.Itoa(uid))

	if err := os.MkdirAll(userDir, 0o700); err != nil {
		return "", fmt.Errorf("create user directory: %w", err)
	}

	return filepath.Join(userDir, "agent.sock"), nil
}

// cleanupSocket removes the socket file
func cleanupSocket(path string) error {
	return os.Remove(path)
}

// isSocketAvailable checks if a socket path is available for use
func isSocketAvailable(path string) bool {
	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return true
	}

	// Try to connect to see if it's actually listening
	conn, err := net.Dial("unix", path)
	if err != nil {
		// If we can't connect, the socket is not in use
		// Socket is available for use
		return true
	}
	_ = conn.Close()
	// Socket is in use
	return false
}

// getSocketInfo returns information about the socket file
func GetSocketInfo(path string) (*SocketInfo, error) {
	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	// On Windows, we'll use a simplified approach for now
	// In a real implementation, you'd get the security descriptor
	return &SocketInfo{
		Path:    path,
		UID:     0, // Simplified for now
		GID:     0, // Simplified for now
		Mode:    stat.Mode(),
		ModTime: stat.ModTime(),
		Owner:   "unknown",
		Group:   "unknown",
	}, nil
}

// SocketInfo contains information about a socket file
type SocketInfo struct {
	Path    string
	UID     int
	GID     int
	Mode    os.FileMode
	ModTime time.Time
	Owner   string // Windows SID string
	Group   string // Windows SID string
}

// NewClient creates a DRPC client for the agent socket at the given path.
func NewClient(path string, logger slog.Logger) (*Client, error) {
	conn, err := net.Dial("unix", path)
	if err != nil {
		return nil, xerrors.Errorf("dial unix socket: %w", err)
	}

	config := yamux.DefaultConfig()
	config.LogOutput = nil
	config.Logger = slog.Stdlib(context.Background(), logger, slog.LevelInfo)
	session, err := yamux.Client(conn, config)
	if err != nil {
		_ = conn.Close()
		return nil, xerrors.Errorf("multiplex client: %w", err)
	}
	return &Client{
		DRPCAgentSocketClient: proto.NewDRPCAgentSocketClient(drpcsdk.MultiplexedConn(session)),
		conn:                  conn,
		session:               session,
	}, nil
}
