package sandbox_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sandbox "github.com/yaoapp/yao/sandbox/v2"
)

func TestImageExists(t *testing.T) {
	for _, pc := range testPools() {
		pc := pc
		t.Run(pc.Name, func(t *testing.T) {
			if pc.Name == "k8s" {
				t.Run("always_true", func(t *testing.T) {
					m := setupManagerForPool(t, pc)
					ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
					defer cancel()
					exists, err := m.ImageExists(ctx, pc.Name, "anything:nonexistent")
					require.NoError(t, err)
					assert.True(t, exists, "k8s mode should always return true")
				})
				return
			}

			m := setupManagerForPool(t, pc)
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			t.Run("existing", func(t *testing.T) {
				exists, err := m.ImageExists(ctx, pc.Name, "alpine:latest")
				require.NoError(t, err)
				assert.True(t, exists)
			})

			t.Run("missing", func(t *testing.T) {
				exists, err := m.ImageExists(ctx, pc.Name, "nonexistent/image:no-such-tag-ever-12345")
				require.NoError(t, err)
				assert.False(t, exists)
			})
		})
	}
}

func TestImagePull(t *testing.T) {
	for _, pc := range testPools() {
		pc := pc
		t.Run(pc.Name, func(t *testing.T) {
			if pc.Name == "k8s" {
				t.Run("noop", func(t *testing.T) {
					m := setupManagerForPool(t, pc)
					ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
					defer cancel()
					ch, err := m.PullImage(ctx, pc.Name, "alpine:latest", sandbox.ImagePullOptions{})
					require.NoError(t, err)
					assert.Nil(t, ch, "k8s mode should return nil channel")
				})
				return
			}

			m := setupManagerForPool(t, pc)
			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()

			t.Run("pull_with_progress", func(t *testing.T) {
				ch, err := m.PullImage(ctx, pc.Name, "alpine:latest", sandbox.ImagePullOptions{})
				require.NoError(t, err)
				require.NotNil(t, ch)

				var count int
				for p := range ch {
					if p.Error != "" {
						t.Fatalf("pull error: %s", p.Error)
					}
					count++
				}
				assert.Greater(t, count, 0, "should receive at least one progress event")
			})
		})
	}
}

func TestEnsureImage(t *testing.T) {
	for _, pc := range testPools() {
		pc := pc
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()

			err := m.EnsureImage(ctx, pc.Name, "alpine:latest", sandbox.ImagePullOptions{})
			require.NoError(t, err)

			if pc.Name != "k8s" {
				exists, err := m.ImageExists(ctx, pc.Name, "alpine:latest")
				require.NoError(t, err)
				assert.True(t, exists)
			}
		})
	}
}

func TestEnsureImage_BadRef(t *testing.T) {
	for _, pc := range testPools() {
		pc := pc
		if pc.Name == "k8s" {
			continue
		}
		t.Run(pc.Name, func(t *testing.T) {
			m := setupManagerForPool(t, pc)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			err := m.EnsureImage(ctx, pc.Name, "nonexistent/image:no-such-tag-ever-12345", sandbox.ImagePullOptions{})
			assert.Error(t, err)
		})
	}
}
