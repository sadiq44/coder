package agentsocket_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"cdr.dev/slog"
	"github.com/coder/coder/v2/agent/agentsocket"
	"github.com/coder/coder/v2/agent/agentsocket/proto"
	"github.com/coder/coder/v2/agent/unit"
)

func TestDRPCAgentSocketService(t *testing.T) {
	t.Parallel()

	t.Run("Ping", func(t *testing.T) {
		t.Parallel()

		socketPath := filepath.Join(t.TempDir(), "test.sock")

		server, err := agentsocket.NewServer(
			socketPath,
			slog.Make().Leveled(slog.LevelDebug),
		)
		require.NoError(t, err)

		err = server.Start()
		require.NoError(t, err)
		defer server.Stop()

		client, err := agentsocket.NewClient(socketPath, slog.Make().Leveled(slog.LevelDebug))
		require.NoError(t, err)
		defer client.Close()

		response, err := client.Ping(context.Background(), &proto.PingRequest{})
		require.NoError(t, err)
		require.Equal(t, "pong", response.Message)
	})

	t.Run("SyncStart", func(t *testing.T) {
		t.Parallel()

		t.Run("NewUnit", func(t *testing.T) {
			t.Parallel()
			socketPath := filepath.Join(t.TempDir(), "test.sock")

			server, err := agentsocket.NewServer(
				socketPath,
				slog.Make().Leveled(slog.LevelDebug),
			)
			require.NoError(t, err)
			err = server.Start()
			require.NoError(t, err)
			defer server.Stop()

			client, err := agentsocket.NewClient(socketPath, slog.Make().Leveled(slog.LevelDebug))
			require.NoError(t, err)
			defer client.Close()

			_, err = client.SyncStart(context.Background(), &proto.SyncStartRequest{Unit: "test-unit"})
			require.NoError(t, err)

			statusResp, err := client.SyncStatus(context.Background(), &proto.SyncStatusRequest{Unit: "test-unit", Recursive: false})
			require.NoError(t, err)
			status := statusResp
			require.Equal(t, "started", status.Status)
		})

		t.Run("UnitAlreadyStarted", func(t *testing.T) {
			t.Parallel()

			socketPath := filepath.Join(t.TempDir(), "test.sock")

			server, err := agentsocket.NewServer(
				socketPath,
				slog.Make().Leveled(slog.LevelDebug),
			)
			require.NoError(t, err)
			err = server.Start()
			require.NoError(t, err)
			defer server.Stop()

			client, err := agentsocket.NewClient(socketPath, slog.Make().Leveled(slog.LevelDebug))
			require.NoError(t, err)
			defer client.Close()

			_, err = client.SyncStart(context.Background(), &proto.SyncStartRequest{Unit: "test-unit"})
			require.NoError(t, err)

			// First Start
			statusResp, err := client.SyncStatus(context.Background(), &proto.SyncStatusRequest{Unit: "test-unit", Recursive: false})
			require.NoError(t, err)
			status := statusResp
			require.Equal(t, "started", status.Status)

			statusResp, err = client.SyncStatus(context.Background(), &proto.SyncStatusRequest{Unit: "test-unit", Recursive: false})
			require.NoError(t, err)
			status = statusResp
			require.Equal(t, "started", status.Status)

			// Second Start
			_, err = client.SyncStart(context.Background(), &proto.SyncStartRequest{Unit: "test-unit"})
			require.ErrorContains(t, err, unit.ErrSameStatusAlreadySet.Error())

			statusResp, err = client.SyncStatus(context.Background(), &proto.SyncStatusRequest{Unit: "test-unit", Recursive: false})
			require.NoError(t, err)
			status = statusResp
			require.Equal(t, "started", status.Status)
		})

		t.Run("UnitAlreadyCompleted", func(t *testing.T) {
			t.Parallel()

			socketPath := filepath.Join(t.TempDir(), "test.sock")

			server, err := agentsocket.NewServer(
				socketPath,
				slog.Make().Leveled(slog.LevelDebug),
			)
			require.NoError(t, err)
			err = server.Start()
			require.NoError(t, err)
			defer server.Stop()

			client, err := agentsocket.NewClient(socketPath, slog.Make().Leveled(slog.LevelDebug))
			require.NoError(t, err)
			defer client.Close()

			// First start
			_, err = client.SyncStart(context.Background(), &proto.SyncStartRequest{Unit: "test-unit"})
			require.NoError(t, err)

			statusResp, err := client.SyncStatus(context.Background(), &proto.SyncStatusRequest{Unit: "test-unit", Recursive: false})
			require.NoError(t, err)
			status := statusResp
			require.Equal(t, "started", status.Status)

			// Complete the unit
			_, err = client.SyncComplete(context.Background(), &proto.SyncCompleteRequest{Unit: "test-unit"})
			require.NoError(t, err)

			statusResp, err = client.SyncStatus(context.Background(), &proto.SyncStatusRequest{Unit: "test-unit", Recursive: false})
			require.NoError(t, err)
			status = statusResp
			require.Equal(t, "completed", status.Status)

			// Second start
			_, err = client.SyncStart(context.Background(), &proto.SyncStartRequest{Unit: "test-unit"})
			require.NoError(t, err)

			statusResp, err = client.SyncStatus(context.Background(), &proto.SyncStatusRequest{Unit: "test-unit", Recursive: false})
			require.NoError(t, err)
			status = statusResp
			require.Equal(t, "started", status.Status)
		})

		t.Run("UnitNotReady", func(t *testing.T) {
			t.Parallel()

			socketPath := filepath.Join(t.TempDir(), "test.sock")

			server, err := agentsocket.NewServer(
				socketPath,
				slog.Make().Leveled(slog.LevelDebug),
			)
			require.NoError(t, err)
			err = server.Start()
			require.NoError(t, err)
			defer server.Stop()

			client, err := agentsocket.NewClient(socketPath, slog.Make().Leveled(slog.LevelDebug))
			require.NoError(t, err)
			defer client.Close()

			_, err = client.SyncWant(context.Background(), &proto.SyncWantRequest{Unit: "test-unit", DependsOn: "dependency-unit"})
			require.NoError(t, err)

			_, err = client.SyncStart(context.Background(), &proto.SyncStartRequest{Unit: "test-unit"})
			require.ErrorContains(t, err, "Unit is not ready")

			statusResp, err := client.SyncStatus(context.Background(), &proto.SyncStatusRequest{Unit: "test-unit", Recursive: false})
			require.NoError(t, err)
			status := statusResp
			require.Equal(t, "", status.Status)
		})
	})

	t.Run("SyncWant", func(t *testing.T) {
		t.Parallel()

		t.Run("NewUnits", func(t *testing.T) {
			t.Parallel()

			socketPath := filepath.Join(t.TempDir(), "test.sock")

			server, err := agentsocket.NewServer(
				socketPath,
				slog.Make().Leveled(slog.LevelDebug),
			)
			require.NoError(t, err)
			err = server.Start()
			require.NoError(t, err)
			defer server.Stop()

			client, err := agentsocket.NewClient(socketPath, slog.Make().Leveled(slog.LevelDebug))
			require.NoError(t, err)
			defer client.Close()

			// If units are not registered, they are registered automatically
			_, err = client.SyncWant(context.Background(), &proto.SyncWantRequest{Unit: "test-unit", DependsOn: "dependency-unit"})
			require.NoError(t, err)

			statusResp, err := client.SyncStatus(context.Background(), &proto.SyncStatusRequest{Unit: "test-unit", Recursive: false})
			require.NoError(t, err)
			status := statusResp
			require.Equal(t, "dependency-unit", status.Dependencies[0].DependsOn)
			require.Equal(t, "completed", status.Dependencies[0].RequiredStatus)
		})

		t.Run("DependencyAlreadyRegistered", func(t *testing.T) {
			t.Parallel()

			socketPath := filepath.Join(t.TempDir(), "test.sock")

			server, err := agentsocket.NewServer(
				socketPath,
				slog.Make().Leveled(slog.LevelDebug),
			)
			require.NoError(t, err)
			err = server.Start()
			require.NoError(t, err)
			defer server.Stop()

			client, err := agentsocket.NewClient(socketPath, slog.Make().Leveled(slog.LevelDebug))
			require.NoError(t, err)
			defer client.Close()

			// Start the dependency unit
			_, err = client.SyncStart(context.Background(), &proto.SyncStartRequest{Unit: "dependency-unit"})
			require.NoError(t, err)

			statusResp, err := client.SyncStatus(context.Background(), &proto.SyncStatusRequest{Unit: "dependency-unit", Recursive: false})
			require.NoError(t, err)
			status := statusResp
			require.Equal(t, "started", status.Status)

			// Add the dependency after the dependency unit has already started
			_, err = client.SyncWant(context.Background(), &proto.SyncWantRequest{Unit: "test-unit", DependsOn: "dependency-unit"})

			// Dependencies can be added even if the dependency unit has already started
			require.NoError(t, err)

			// The dependency is now reflected in the test unit's status
			statusResp, err = client.SyncStatus(context.Background(), &proto.SyncStatusRequest{Unit: "test-unit", Recursive: false})
			require.NoError(t, err)
			status = statusResp
			require.Equal(t, "dependency-unit", status.Dependencies[0].DependsOn)
			require.Equal(t, "completed", status.Dependencies[0].RequiredStatus)
		})

		t.Run("DependencyAddedAfterDependentStarted", func(t *testing.T) {
			t.Parallel()

			socketPath := filepath.Join(t.TempDir(), "test.sock")

			server, err := agentsocket.NewServer(
				socketPath,
				slog.Make().Leveled(slog.LevelDebug),
			)
			require.NoError(t, err)
			err = server.Start()
			require.NoError(t, err)
			defer server.Stop()

			client, err := agentsocket.NewClient(socketPath, slog.Make().Leveled(slog.LevelDebug))
			require.NoError(t, err)
			defer client.Close()

			// Start the dependent unit
			_, err = client.SyncStart(context.Background(), &proto.SyncStartRequest{Unit: "test-unit"})
			require.NoError(t, err)

			statusResp, err := client.SyncStatus(context.Background(), &proto.SyncStatusRequest{Unit: "test-unit", Recursive: false})
			require.NoError(t, err)
			status := statusResp
			require.Equal(t, "started", status.Status)

			// Add the dependency after the dependency unit has already started
			_, err = client.SyncWant(context.Background(), &proto.SyncWantRequest{Unit: "test-unit", DependsOn: "dependency-unit"})

			// Dependencies can be added even if the dependent unit has already started.
			// The dependency applies the next time a unit is started. The current status is not updated.
			// This is to allow flexible dependency management. It does mean that users of this API should
			// take care to add dependencies before they start their dependent units.
			require.NoError(t, err)

			// The dependency is now reflected in the test unit's status
			statusResp, err = client.SyncStatus(context.Background(), &proto.SyncStatusRequest{Unit: "test-unit", Recursive: false})
			require.NoError(t, err)
			status = statusResp
			require.Equal(t, "dependency-unit", status.Dependencies[0].DependsOn)
			require.Equal(t, "completed", status.Dependencies[0].RequiredStatus)
		})
	})
}
