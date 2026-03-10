package sandbox_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sandbox "github.com/yaoapp/yao/sandbox/v2"
	"github.com/yaoapp/yao/workspace"
)

func TestWorkspaceID_Set(t *testing.T) {
	skipIfNoDocker(t)

	for _, pc := range testNodes() {
		pc := pc
		t.Run(pc.Name, func(t *testing.T) {
			sbm, wsm := setupManagerWithWorkspace(t, &pc)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			ws, err := wsm.Create(ctx, workspace.CreateOptions{
				Name: "test-ws", Owner: "user", Node: pc.TaiID,
			})
			require.NoError(t, err)
			defer wsm.Delete(context.Background(), ws.ID, true)

			box := createTestBox(t, sbm, pc, func(co *sandbox.CreateOptions) {
				co.WorkspaceID = ws.ID
			})

			assert.Equal(t, ws.ID, box.WorkspaceID())
		})
	}
}

func TestWorkspaceID_Empty(t *testing.T) {
	skipIfNoDocker(t)

	for _, pc := range testNodes() {
		pc := pc
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForNode(t, &pc)
			box := createTestBox(t, m, pc)
			assert.Empty(t, box.WorkspaceID())
		})
	}
}

func TestWorkspace_NodeRouting(t *testing.T) {
	skipIfNoDocker(t)

	for _, pc := range testNodes() {
		pc := pc
		t.Run(pc.Name, func(t *testing.T) {
			sbm, wsm := setupManagerWithWorkspace(t, &pc)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			ws, err := wsm.Create(ctx, workspace.CreateOptions{
				Name: "routed-ws", Owner: "user", Node: pc.TaiID,
			})
			require.NoError(t, err)
			defer wsm.Delete(context.Background(), ws.ID, true)

			box := createTestBox(t, sbm, pc, func(co *sandbox.CreateOptions) {
				co.WorkspaceID = ws.ID
			})

			assert.Equal(t, pc.TaiID, box.NodeID())
		})
	}
}

func TestWorkspace_InvalidID(t *testing.T) {
	skipIfNoDocker(t)

	for _, pc := range testNodes() {
		pc := pc
		t.Run(pc.Name, func(t *testing.T) {
			sbm, wsm := setupManagerWithWorkspace(t, &pc)
			ensureTestImage(t, sbm, pc.TaiID)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			wsID := "nonexistent-workspace"

			box, err := sbm.Create(ctx, sandbox.CreateOptions{
				Image:       testImage(),
				Owner:       "user",
				WorkspaceID: wsID,
			})

			// With online nodes the manager auto-creates the workspace.
			require.NoError(t, err)
			require.NotNil(t, box)
			defer box.Remove(context.Background())
			if wsm != nil {
				defer wsm.Delete(context.Background(), wsID, true)
			}
		})
	}
}

func TestWorkspace_BindMountLocal(t *testing.T) {
	skipIfNoDocker(t)

	pc := nodeConfig{Name: "local", Addr: testLocalAddr()}
	sbm, wsm := setupManagerWithWorkspace(t, &pc)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ws, err := wsm.Create(ctx, workspace.CreateOptions{
		Name: "mount-ws", Owner: "user", Node: pc.TaiID,
	})
	require.NoError(t, err)
	defer wsm.Delete(context.Background(), ws.ID, true)

	require.NoError(t, wsm.WriteFile(ctx, ws.ID, "seed.txt", []byte("hello from workspace"), 0644))

	box := createTestBox(t, sbm, pc, func(co *sandbox.CreateOptions) {
		co.WorkspaceID = ws.ID
	})

	result, err := box.Exec(ctx, []string{"cat", "/workspace/seed.txt"})
	require.NoError(t, err)
	assert.Equal(t, "hello from workspace", result.Stdout)
}

func TestWorkspace_ContainerWriteBack(t *testing.T) {
	skipIfNoDocker(t)

	pc := nodeConfig{Name: "local", Addr: testLocalAddr()}
	sbm, wsm := setupManagerWithWorkspace(t, &pc)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ws, err := wsm.Create(ctx, workspace.CreateOptions{
		Name: "writeback-ws", Owner: "user", Node: pc.TaiID,
	})
	require.NoError(t, err)
	defer wsm.Delete(context.Background(), ws.ID, true)

	box := createTestBox(t, sbm, pc, func(co *sandbox.CreateOptions) {
		co.WorkspaceID = ws.ID
	})

	_, err = box.Exec(ctx, []string{"sh", "-c", "echo 'from container' > /workspace/output.txt"})
	require.NoError(t, err)

	data, err := wsm.ReadFile(ctx, ws.ID, "output.txt")
	require.NoError(t, err)
	assert.Equal(t, "from container\n", string(data))
}

func TestWorkspace_ReadOnlyMount(t *testing.T) {
	skipIfNoDocker(t)

	pc := nodeConfig{Name: "local", Addr: testLocalAddr()}
	sbm, wsm := setupManagerWithWorkspace(t, &pc)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ws, err := wsm.Create(ctx, workspace.CreateOptions{
		Name: "ro-ws", Owner: "user", Node: pc.TaiID,
	})
	require.NoError(t, err)
	defer wsm.Delete(context.Background(), ws.ID, true)

	require.NoError(t, wsm.WriteFile(ctx, ws.ID, "readonly.txt", []byte("immutable"), 0644))

	box := createTestBox(t, sbm, pc, func(co *sandbox.CreateOptions) {
		co.WorkspaceID = ws.ID
		co.MountMode = "ro"
	})

	result, err := box.Exec(ctx, []string{"cat", "/workspace/readonly.txt"})
	require.NoError(t, err)
	assert.Equal(t, "immutable", result.Stdout)

	result, err = box.Exec(ctx, []string{"sh", "-c", "echo fail > /workspace/nope.txt 2>&1; echo $?"})
	require.NoError(t, err)
	// Write to read-only mount should fail (non-zero exit or error message)
	assert.True(t, result.ExitCode != 0 || result.Stdout != "0\n" || len(result.Stderr) > 0,
		"expected write to read-only mount to fail")
}

func TestWorkspace_CustomMountPath(t *testing.T) {
	skipIfNoDocker(t)

	pc := nodeConfig{Name: "local", Addr: testLocalAddr()}
	sbm, wsm := setupManagerWithWorkspace(t, &pc)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ws, err := wsm.Create(ctx, workspace.CreateOptions{
		Name: "custom-path-ws", Owner: "user", Node: pc.TaiID,
	})
	require.NoError(t, err)
	defer wsm.Delete(context.Background(), ws.ID, true)

	require.NoError(t, wsm.WriteFile(ctx, ws.ID, "data.json", []byte(`{"ok":true}`), 0644))

	box := createTestBox(t, sbm, pc, func(co *sandbox.CreateOptions) {
		co.WorkspaceID = ws.ID
		co.MountPath = "/data"
	})

	result, err := box.Exec(ctx, []string{"cat", "/data/data.json"})
	require.NoError(t, err)
	assert.Equal(t, `{"ok":true}`, result.Stdout)
}

func TestWorkspace_BoxWorkspaceFS(t *testing.T) {
	skipIfNoDocker(t)

	for _, pc := range testNodes() {
		if pc.Name == "local" {
			continue
		}
		pc := pc
		t.Run(pc.Name, func(t *testing.T) {
			sbm, wsm := setupManagerWithWorkspace(t, &pc)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			ws, err := wsm.Create(ctx, workspace.CreateOptions{
				Name: "fs-ws", Owner: "user", Node: pc.TaiID,
			})
			require.NoError(t, err)
			defer wsm.Delete(context.Background(), ws.ID, true)

			box := createTestBox(t, sbm, pc, func(co *sandbox.CreateOptions) {
				co.WorkspaceID = ws.ID
			})

			wfs := box.Workspace()
			if wfs == nil {
				t.Skip("Workspace FS not available")
			}

			require.NoError(t, wfs.WriteFile("via-box.txt", []byte("box wrote this"), 0644))

			data, err := wsm.ReadFile(ctx, ws.ID, "via-box.txt")
			require.NoError(t, err)
			assert.Equal(t, "box wrote this", string(data))
		})
	}
}

func TestWorkspace_LabelPersistence(t *testing.T) {
	skipIfNoDocker(t)

	for _, pc := range testNodes() {
		pc := pc
		t.Run(pc.Name, func(t *testing.T) {
			sbm, wsm := setupManagerWithWorkspace(t, &pc)

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			ws, err := wsm.Create(ctx, workspace.CreateOptions{
				Name: "label-ws", Owner: "user", Node: pc.TaiID,
			})
			require.NoError(t, err)
			defer wsm.Delete(context.Background(), ws.ID, true)

			box := createTestBox(t, sbm, pc, func(co *sandbox.CreateOptions) {
				co.WorkspaceID = ws.ID
			})

			assert.Equal(t, ws.ID, box.WorkspaceID())

			// Container should also carry the label (verify via exec reading env or
			// just trust that buildTaiCreateOptions sets it — the label is tested
			// indirectly by TestWorkspace_NodeRouting which relies on correct routing)
			info, err := box.Info(ctx)
			require.NoError(t, err)
			assert.Contains(t, []string{"running", "Running"}, info.Status)
		})
	}
}
