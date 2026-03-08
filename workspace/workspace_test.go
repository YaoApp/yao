package workspace_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/yaoapp/yao/workspace"
)

func TestCreate(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			ws := createWorkspace(t, m, pc.Name)

			assert.NotEmpty(t, ws.ID)
			assert.Equal(t, "test-workspace", ws.Name)
			assert.Equal(t, "test-user", ws.Owner)
			assert.Equal(t, pc.Name, ws.Node)
			assert.False(t, ws.CreatedAt.IsZero())
			assert.False(t, ws.UpdatedAt.IsZero())
		})
	}
}

func TestCreate_AutoID(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			ws := createWorkspace(t, m, pc.Name)

			assert.True(t, len(ws.ID) > 0)
			assert.Contains(t, ws.ID, "ws-")
		})
	}
}

func TestCreate_ExplicitID(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			ws := createWorkspace(t, m, pc.Name, func(co *workspace.CreateOptions) {
				co.ID = "my-custom-id"
			})

			assert.Equal(t, "my-custom-id", ws.ID)
		})
	}
}

func TestCreate_WithLabels(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			ws := createWorkspace(t, m, pc.Name, func(co *workspace.CreateOptions) {
				co.Labels = map[string]string{"project": "frontend", "env": "dev"}
			})

			assert.Equal(t, "frontend", ws.Labels["project"])
			assert.Equal(t, "dev", ws.Labels["env"])
		})
	}
}

func TestCreate_InvalidNode(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			_, err := m.Create(context.Background(), workspace.CreateOptions{
				Name:  "bad",
				Owner: "user",
				Node:  "",
			})
			assert.ErrorIs(t, err, workspace.ErrNodeMissing)
		})
	}
}

func TestCreate_NodeNotFound(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			_, err := m.Create(context.Background(), workspace.CreateOptions{
				Name:  "bad",
				Owner: "user",
				Node:  "nonexistent-node",
			})
			assert.ErrorIs(t, err, workspace.ErrNodeOffline)
		})
	}
}

func TestGet(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			ws := createWorkspace(t, m, pc.Name)

			got, err := m.Get(context.Background(), ws.ID)
			require.NoError(t, err)
			assert.Equal(t, ws.ID, got.ID)
			assert.Equal(t, ws.Name, got.Name)
			assert.Equal(t, ws.Owner, got.Owner)
			assert.Equal(t, ws.Node, got.Node)
		})
	}
}

func TestGet_NotFound(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			_, err := m.Get(context.Background(), "nonexistent")
			assert.ErrorIs(t, err, workspace.ErrNotFound)
		})
	}
}

func TestList(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			createWorkspace(t, m, pc.Name, func(co *workspace.CreateOptions) { co.Name = "ws-1" })
			createWorkspace(t, m, pc.Name, func(co *workspace.CreateOptions) { co.Name = "ws-2" })

			list, err := m.List(context.Background(), workspace.ListOptions{})
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(list), 2)
		})
	}
}

func TestList_FilterOwner(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			createWorkspace(t, m, pc.Name, func(co *workspace.CreateOptions) {
				co.Owner = "alice"
				co.Name = "alice-ws"
			})
			createWorkspace(t, m, pc.Name, func(co *workspace.CreateOptions) {
				co.Owner = "bob"
				co.Name = "bob-ws"
			})

			list, err := m.List(context.Background(), workspace.ListOptions{Owner: "alice"})
			require.NoError(t, err)
			assert.Len(t, list, 1)
			assert.Equal(t, "alice", list[0].Owner)
		})
	}
}

func TestList_FilterNode(t *testing.T) {
	m, nodeA, nodeB := setupManagerMultiNode(t)

	ctx := context.Background()
	_, err := m.Create(ctx, workspace.CreateOptions{Name: "a", Owner: "u", Node: nodeA})
	require.NoError(t, err)
	_, err = m.Create(ctx, workspace.CreateOptions{Name: "b", Owner: "u", Node: nodeB})
	require.NoError(t, err)

	list, err := m.List(ctx, workspace.ListOptions{Node: nodeA})
	require.NoError(t, err)
	assert.Len(t, list, 1)
	assert.Equal(t, nodeA, list[0].Node)
}

func TestUpdate_Name(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			ws := createWorkspace(t, m, pc.Name)

			newName := "renamed-workspace"
			updated, err := m.Update(context.Background(), ws.ID, workspace.UpdateOptions{
				Name: &newName,
			})
			require.NoError(t, err)
			assert.Equal(t, newName, updated.Name)
			assert.Equal(t, ws.Owner, updated.Owner)
			assert.True(t, updated.UpdatedAt.After(ws.UpdatedAt) || updated.UpdatedAt.Equal(ws.UpdatedAt))
		})
	}
}

func TestUpdate_Labels(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			ws := createWorkspace(t, m, pc.Name, func(co *workspace.CreateOptions) {
				co.Labels = map[string]string{"old": "value"}
			})

			updated, err := m.Update(context.Background(), ws.ID, workspace.UpdateOptions{
				Labels: map[string]string{"new": "label"},
			})
			require.NoError(t, err)
			assert.Equal(t, "label", updated.Labels["new"])
			assert.Empty(t, updated.Labels["old"])
		})
	}
}

func TestUpdate_NotFound(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			_, err := m.Update(context.Background(), "nonexistent", workspace.UpdateOptions{})
			assert.ErrorIs(t, err, workspace.ErrNotFound)
		})
	}
}

func TestDelete(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			ws, err := m.Create(context.Background(), workspace.CreateOptions{
				Name: "to-delete", Owner: "user", Node: pc.Name,
			})
			require.NoError(t, err)

			err = m.Delete(context.Background(), ws.ID, false)
			require.NoError(t, err)

			_, err = m.Get(context.Background(), ws.ID)
			assert.ErrorIs(t, err, workspace.ErrNotFound)
		})
	}
}

func TestDelete_NotFound(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			err := m.Delete(context.Background(), "nonexistent", false)
			assert.ErrorIs(t, err, workspace.ErrNotFound)
		})
	}
}

func TestNodes(t *testing.T) {
	m, nodeA, nodeB := setupManagerMultiNode(t)
	nodes := m.Nodes()
	assert.GreaterOrEqual(t, len(nodes), 2)

	names := make(map[string]bool)
	for _, n := range nodes {
		names[n.Name] = true
	}
	assert.True(t, names[nodeA])
	assert.True(t, names[nodeB])
}

func TestNodeForWorkspace(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			ws := createWorkspace(t, m, pc.Name)

			node, err := m.NodeForWorkspace(context.Background(), ws.ID)
			require.NoError(t, err)
			assert.Equal(t, pc.Name, node)
		})
	}
}

func TestNodeForWorkspace_NotFound(t *testing.T) {
	for _, pc := range testPools() {
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			_, err := m.NodeForWorkspace(context.Background(), "nonexistent")
			assert.ErrorIs(t, err, workspace.ErrNotFound)
		})
	}
}

func TestRegistryDrivenNodes(t *testing.T) {
	m, nodeA, nodeB := setupManagerMultiNode(t)
	nodes := m.Nodes()
	assert.GreaterOrEqual(t, len(nodes), 2)

	names := make(map[string]bool)
	for _, n := range nodes {
		names[n.Name] = true
	}
	assert.True(t, names[nodeA])
	assert.True(t, names[nodeB])
}

func TestMountPath(t *testing.T) {
	m := setupManagerForPool(t, poolConfig{Name: "local", Addr: "local"})
	ws := createWorkspace(t, m, "local")

	mountPath, err := m.MountPath(context.Background(), ws.ID)
	require.NoError(t, err)
	assert.Contains(t, mountPath, ws.ID)
}

func TestMountPath_NotFound(t *testing.T) {
	m := setupManagerForPool(t, poolConfig{Name: "local", Addr: "local"})
	_, err := m.MountPath(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, workspace.ErrNotFound)
}
