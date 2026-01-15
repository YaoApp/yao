package manager_test

// Integration tests for Event triggers
// Tests Manager.HandleEvent() with various event types and scenarios

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/xun/capsule"
	"github.com/yaoapp/yao/agent/robot/manager"
	"github.com/yaoapp/yao/agent/robot/pool"
	"github.com/yaoapp/yao/agent/robot/types"
	"github.com/yaoapp/yao/agent/testutils"
)

// ==================== Event Trigger Tests ====================

// TestIntegrationEventTrigger tests event trigger flow
func TestIntegrationEventTrigger(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupIntegrationRobots(t)
	defer cleanupIntegrationRobots(t)

	t.Run("webhook event success", func(t *testing.T) {
		setupEventTestRobot(t, "robot_integ_event_webhook", "team_integ_event")

		config := &manager.Config{
			TickInterval: 100 * time.Millisecond,
			PoolConfig:   &pool.Config{WorkerSize: 3, QueueSize: 20},
		}
		m := manager.NewWithConfig(config)

		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		// Verify robot is loaded into cache
		robot := m.Cache().Get("robot_integ_event_webhook")
		require.NotNil(t, robot, "Robot should be loaded into cache")

		ctx := types.NewContext(context.Background(), nil)
		req := &types.EventRequest{
			MemberID:  "robot_integ_event_webhook",
			Source:    "webhook",
			EventType: "lead.created",
			Data: map[string]interface{}{
				"name":    "John Doe",
				"email":   "john@example.com",
				"company": "Acme Corp",
			},
		}

		result, err := m.HandleEvent(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.ExecutionID)
		assert.Equal(t, types.ExecPending, result.Status)
		assert.Contains(t, result.Message, "webhook")
		assert.Contains(t, result.Message, "lead.created")

		// Wait for execution
		time.Sleep(500 * time.Millisecond)

		// Verify execution completed
		assert.GreaterOrEqual(t, m.Executor().ExecCount(), 1)
	})

	t.Run("database event success", func(t *testing.T) {
		setupEventTestRobot(t, "robot_integ_event_db", "team_integ_event")

		m := manager.New()
		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)
		req := &types.EventRequest{
			MemberID:  "robot_integ_event_db",
			Source:    "database",
			EventType: "order.paid",
			Data: map[string]interface{}{
				"order_id": "ORD-12345",
				"amount":   1500.00,
				"customer": "customer_001",
			},
		}

		result, err := m.HandleEvent(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.ExecutionID)
	})

	t.Run("event with complex data", func(t *testing.T) {
		setupEventTestRobot(t, "robot_integ_event_complex", "team_integ_event")

		m := manager.New()
		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)
		req := &types.EventRequest{
			MemberID:  "robot_integ_event_complex",
			Source:    "webhook",
			EventType: "crm.contact.updated",
			Data: map[string]interface{}{
				"contact": map[string]interface{}{
					"id":    "contact_001",
					"name":  "Jane Smith",
					"email": "jane@example.com",
					"tags":  []string{"vip", "enterprise"},
				},
				"changes": map[string]interface{}{
					"old_status": "active",
					"new_status": "premium",
				},
				"timestamp": time.Now().Unix(),
			},
		}

		result, err := m.HandleEvent(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.ExecutionID)
	})
}

// TestIntegrationEventTriggerErrors tests error cases for event triggers
func TestIntegrationEventTriggerErrors(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupIntegrationRobots(t)
	defer cleanupIntegrationRobots(t)

	t.Run("robot not found", func(t *testing.T) {
		m := manager.New()
		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)
		req := &types.EventRequest{
			MemberID:  "robot_nonexistent",
			Source:    "webhook",
			EventType: "test.event",
		}

		_, err = m.HandleEvent(ctx, req)
		assert.Error(t, err)
		assert.Equal(t, types.ErrRobotNotFound, err)
	})

	t.Run("robot paused", func(t *testing.T) {
		setupEventTestRobotPaused(t, "robot_integ_event_paused", "team_integ_event")

		m := manager.New()
		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)
		req := &types.EventRequest{
			MemberID:  "robot_integ_event_paused",
			Source:    "webhook",
			EventType: "test.event",
		}

		_, err = m.HandleEvent(ctx, req)
		assert.Error(t, err)
		assert.Equal(t, types.ErrRobotPaused, err)
	})

	t.Run("event trigger disabled", func(t *testing.T) {
		setupEventTestRobotDisabled(t, "robot_integ_event_disabled", "team_integ_event")

		m := manager.New()
		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)
		req := &types.EventRequest{
			MemberID:  "robot_integ_event_disabled",
			Source:    "webhook",
			EventType: "test.event",
		}

		_, err = m.HandleEvent(ctx, req)
		assert.Error(t, err)
		assert.Equal(t, types.ErrTriggerDisabled, err)
	})

	t.Run("invalid request - empty member_id", func(t *testing.T) {
		m := manager.New()
		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)
		req := &types.EventRequest{
			MemberID:  "", // Empty
			Source:    "webhook",
			EventType: "test.event",
		}

		_, err = m.HandleEvent(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "member_id")
	})

	t.Run("invalid request - empty source", func(t *testing.T) {
		setupEventTestRobot(t, "robot_integ_event_nosource", "team_integ_event")

		m := manager.New()
		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)
		req := &types.EventRequest{
			MemberID:  "robot_integ_event_nosource",
			Source:    "", // Empty
			EventType: "test.event",
		}

		_, err = m.HandleEvent(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "source")
	})

	t.Run("invalid request - empty event_type", func(t *testing.T) {
		setupEventTestRobot(t, "robot_integ_event_notype", "team_integ_event")

		m := manager.New()
		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)
		req := &types.EventRequest{
			MemberID:  "robot_integ_event_notype",
			Source:    "webhook",
			EventType: "", // Empty
		}

		_, err = m.HandleEvent(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "event_type")
	})

	t.Run("manager not started", func(t *testing.T) {
		m := manager.New()
		// Don't start

		ctx := types.NewContext(context.Background(), nil)
		req := &types.EventRequest{
			MemberID:  "robot_test",
			Source:    "webhook",
			EventType: "test.event",
		}

		_, err := m.HandleEvent(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not started")
	})
}

// TestIntegrationEventTriggerTypes tests various event types
func TestIntegrationEventTriggerTypes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupIntegrationRobots(t)
	defer cleanupIntegrationRobots(t)

	// Common event types to test
	eventTypes := []struct {
		name      string
		eventType string
		data      map[string]interface{}
	}{
		{
			name:      "lead.created",
			eventType: "lead.created",
			data:      map[string]interface{}{"name": "John", "email": "john@example.com"},
		},
		{
			name:      "order.paid",
			eventType: "order.paid",
			data:      map[string]interface{}{"order_id": "ORD-001", "amount": 100.0},
		},
		{
			name:      "customer.signup",
			eventType: "customer.signup",
			data:      map[string]interface{}{"customer_id": "cust_001", "plan": "premium"},
		},
		{
			name:      "ticket.created",
			eventType: "ticket.created",
			data:      map[string]interface{}{"ticket_id": "TKT-001", "priority": "high"},
		},
		{
			name:      "inventory.low",
			eventType: "inventory.low",
			data:      map[string]interface{}{"product_id": "PRD-001", "quantity": 5},
		},
	}

	for _, tc := range eventTypes {
		t.Run(tc.name, func(t *testing.T) {
			memberID := "robot_integ_event_type_" + tc.name
			setupEventTestRobot(t, memberID, "team_integ_event")

			m := manager.New()
			err := m.Start()
			require.NoError(t, err)
			defer m.Stop()

			ctx := types.NewContext(context.Background(), nil)
			req := &types.EventRequest{
				MemberID:  memberID,
				Source:    "webhook",
				EventType: tc.eventType,
				Data:      tc.data,
			}

			result, err := m.HandleEvent(ctx, req)
			assert.NoError(t, err, "Event type %s should succeed", tc.eventType)
			assert.NotNil(t, result)
			assert.NotEmpty(t, result.ExecutionID)
			assert.Equal(t, types.ExecPending, result.Status)
		})
	}
}

// TestIntegrationEventTriggerSources tests different event sources
func TestIntegrationEventTriggerSources(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupIntegrationRobots(t)
	defer cleanupIntegrationRobots(t)

	sources := []string{"webhook", "database", "api", "scheduler", "internal"}

	for _, source := range sources {
		t.Run("source_"+source, func(t *testing.T) {
			memberID := "robot_integ_event_src_" + source
			setupEventTestRobot(t, memberID, "team_integ_event")

			m := manager.New()
			err := m.Start()
			require.NoError(t, err)
			defer m.Stop()

			ctx := types.NewContext(context.Background(), nil)
			req := &types.EventRequest{
				MemberID:  memberID,
				Source:    source,
				EventType: "test.event",
				Data:      map[string]interface{}{"source": source},
			}

			result, err := m.HandleEvent(ctx, req)
			assert.NoError(t, err, "Source %s should succeed", source)
			assert.NotNil(t, result)
			assert.NotEmpty(t, result.ExecutionID)
		})
	}
}

// TestIntegrationEventTriggerWithEmptyData tests event with empty or nil data
func TestIntegrationEventTriggerWithEmptyData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	testutils.Prepare(t)
	defer testutils.Clean(t)

	cleanupIntegrationRobots(t)
	defer cleanupIntegrationRobots(t)

	t.Run("nil data", func(t *testing.T) {
		setupEventTestRobot(t, "robot_integ_event_nildata", "team_integ_event")

		m := manager.New()
		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)
		req := &types.EventRequest{
			MemberID:  "robot_integ_event_nildata",
			Source:    "webhook",
			EventType: "ping",
			Data:      nil, // Nil data
		}

		result, err := m.HandleEvent(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.ExecutionID)
	})

	t.Run("empty data map", func(t *testing.T) {
		setupEventTestRobot(t, "robot_integ_event_emptydata", "team_integ_event")

		m := manager.New()
		err := m.Start()
		require.NoError(t, err)
		defer m.Stop()

		ctx := types.NewContext(context.Background(), nil)
		req := &types.EventRequest{
			MemberID:  "robot_integ_event_emptydata",
			Source:    "webhook",
			EventType: "heartbeat",
			Data:      map[string]interface{}{}, // Empty map
		}

		result, err := m.HandleEvent(ctx, req)
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.ExecutionID)
	})
}

// ==================== Test Data Setup Helpers ====================

// setupEventTestRobot creates a robot with event trigger enabled
func setupEventTestRobot(t *testing.T, memberID, teamID string) {
	m := model.Select("__yao.member")
	tableName := m.MetaData.Table.Name
	qb := capsule.Query()

	robotConfig := map[string]interface{}{
		"identity": map[string]interface{}{
			"role":   "Event Test Robot",
			"duties": []string{"Handle event triggers"},
		},
		"quota": map[string]interface{}{
			"max":      5,
			"queue":    20,
			"priority": 5,
		},
		"triggers": map[string]interface{}{
			"clock":     map[string]interface{}{"enabled": false},
			"intervene": map[string]interface{}{"enabled": false},
			"event":     map[string]interface{}{"enabled": true},
		},
		"events": []map[string]interface{}{
			{
				"type":   "webhook",
				"source": "/webhook/events",
			},
			{
				"type":   "database",
				"source": "orders",
			},
		},
	}
	configJSON, _ := json.Marshal(robotConfig)

	err := qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       memberID,
			"team_id":         teamID,
			"member_type":     "robot",
			"display_name":    "Event Test Robot " + memberID,
			"status":          "active",
			"role_id":         "member",
			"autonomous_mode": true,
			"robot_status":    "idle",
			"robot_config":    string(configJSON),
		},
	})
	if err != nil {
		t.Fatalf("Failed to insert %s: %v", memberID, err)
	}
}

// setupEventTestRobotPaused creates a paused robot
func setupEventTestRobotPaused(t *testing.T, memberID, teamID string) {
	m := model.Select("__yao.member")
	tableName := m.MetaData.Table.Name
	qb := capsule.Query()

	robotConfig := map[string]interface{}{
		"identity": map[string]interface{}{"role": "Paused Event Robot"},
		"triggers": map[string]interface{}{
			"event": map[string]interface{}{"enabled": true},
		},
	}
	configJSON, _ := json.Marshal(robotConfig)

	err := qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       memberID,
			"team_id":         teamID,
			"member_type":     "robot",
			"display_name":    "Paused Event Robot " + memberID,
			"status":          "active",
			"role_id":         "member",
			"autonomous_mode": true,
			"robot_status":    "paused", // Paused
			"robot_config":    string(configJSON),
		},
	})
	if err != nil {
		t.Fatalf("Failed to insert %s: %v", memberID, err)
	}
}

// setupEventTestRobotDisabled creates a robot with event trigger disabled
func setupEventTestRobotDisabled(t *testing.T, memberID, teamID string) {
	m := model.Select("__yao.member")
	tableName := m.MetaData.Table.Name
	qb := capsule.Query()

	robotConfig := map[string]interface{}{
		"identity": map[string]interface{}{"role": "Event Disabled Robot"},
		"triggers": map[string]interface{}{
			"event": map[string]interface{}{"enabled": false}, // Disabled
		},
	}
	configJSON, _ := json.Marshal(robotConfig)

	err := qb.Table(tableName).Insert([]map[string]interface{}{
		{
			"member_id":       memberID,
			"team_id":         teamID,
			"member_type":     "robot",
			"display_name":    "Event Disabled Robot " + memberID,
			"status":          "active",
			"role_id":         "member",
			"autonomous_mode": true,
			"robot_status":    "idle",
			"robot_config":    string(configJSON),
		},
	})
	if err != nil {
		t.Fatalf("Failed to insert %s: %v", memberID, err)
	}
}
