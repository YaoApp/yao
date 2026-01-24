package api

import (
	"context"
	"fmt"
	"time"

	"github.com/yaoapp/yao/agent/robot/store"
	"github.com/yaoapp/yao/agent/robot/types"
)

// ==================== Result Types ====================

// ResultQuery - query parameters for listing results
type ResultQuery struct {
	TriggerType types.TriggerType `json:"trigger_type,omitempty"` // clock | human | event
	Keyword     string            `json:"keyword,omitempty"`      // Search in name/summary
	Page        int               `json:"page,omitempty"`
	PageSize    int               `json:"pagesize,omitempty"`
}

// ResultItem - result list item (subset of execution)
type ResultItem struct {
	ID             string            `json:"id"`
	MemberID       string            `json:"member_id"`
	TriggerType    types.TriggerType `json:"trigger_type"`
	Status         types.ExecStatus  `json:"status"`
	Name           string            `json:"name"`
	Summary        string            `json:"summary"`
	StartTime      time.Time         `json:"start_time"`
	EndTime        *time.Time        `json:"end_time,omitempty"`
	HasAttachments bool              `json:"has_attachments"`
}

// ResultDetail - full result with delivery content
type ResultDetail struct {
	ID          string                `json:"id"`
	MemberID    string                `json:"member_id"`
	TriggerType types.TriggerType     `json:"trigger_type"`
	Status      types.ExecStatus      `json:"status"`
	Name        string                `json:"name"`
	Delivery    *types.DeliveryResult `json:"delivery,omitempty"`
	StartTime   time.Time             `json:"start_time"`
	EndTime     *time.Time            `json:"end_time,omitempty"`
}

// ResultListResponse - paginated response
type ResultListResponse struct {
	Data     []*ResultItem `json:"data"`
	Total    int           `json:"total"`
	Page     int           `json:"page"`
	PageSize int           `json:"pagesize"`
}

// ==================== Result API Functions ====================

// ListResults returns completed executions with delivery content for a robot
func ListResults(ctx *types.Context, memberID string, query *ResultQuery) (*ResultListResponse, error) {
	if memberID == "" {
		return nil, fmt.Errorf("member_id is required")
	}

	if query == nil {
		query = &ResultQuery{}
	}
	query.applyDefaults()

	// Build store options
	opts := &store.ResultListOptions{
		MemberID: memberID,
		Limit:    query.PageSize,
		Offset:   (query.Page - 1) * query.PageSize,
	}

	if query.TriggerType != "" {
		opts.TriggerType = query.TriggerType
	}
	if query.Keyword != "" {
		opts.Keyword = query.Keyword
	}

	// Query from store
	result, err := getExecutionStore().ListResults(context.Background(), opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list results: %w", err)
	}

	// Transform to ResultItem slice
	items := make([]*ResultItem, 0, len(result.Data))
	for _, record := range result.Data {
		item := recordToResultItem(record)
		if item != nil {
			items = append(items, item)
		}
	}

	return &ResultListResponse{
		Data:     items,
		Total:    result.Total,
		Page:     result.Page,
		PageSize: result.PageSize,
	}, nil
}

// GetResult returns a single result by execution ID
func GetResult(ctx *types.Context, execID string) (*ResultDetail, error) {
	if execID == "" {
		return nil, fmt.Errorf("execution_id is required")
	}

	// Get from store
	record, err := getExecutionStore().Get(context.Background(), execID)
	if err != nil {
		return nil, fmt.Errorf("failed to get result: %w", err)
	}
	if record == nil {
		return nil, fmt.Errorf("result not found: %s", execID)
	}

	// Verify it has delivery content
	if record.Delivery == nil || record.Delivery.Content == nil {
		return nil, fmt.Errorf("result not found: %s (no delivery content)", execID)
	}

	return recordToResultDetail(record), nil
}

// ==================== Helper Functions ====================

// applyDefaults applies default values to ResultQuery
func (q *ResultQuery) applyDefaults() {
	if q.Page <= 0 {
		q.Page = 1
	}
	if q.PageSize <= 0 {
		q.PageSize = 20
	}
	if q.PageSize > 100 {
		q.PageSize = 100
	}
}

// recordToResultItem converts ExecutionRecord to ResultItem
func recordToResultItem(record *store.ExecutionRecord) *ResultItem {
	if record == nil {
		return nil
	}

	item := &ResultItem{
		ID:          record.ExecutionID,
		MemberID:    record.MemberID,
		TriggerType: record.TriggerType,
		Status:      record.Status,
		Name:        record.Name,
	}

	// Set times
	if record.StartTime != nil {
		item.StartTime = *record.StartTime
	}
	item.EndTime = record.EndTime

	// Extract summary and attachments from delivery
	if record.Delivery != nil && record.Delivery.Content != nil {
		item.Summary = record.Delivery.Content.Summary
		item.HasAttachments = len(record.Delivery.Content.Attachments) > 0
	}

	return item
}

// recordToResultDetail converts ExecutionRecord to ResultDetail
func recordToResultDetail(record *store.ExecutionRecord) *ResultDetail {
	if record == nil {
		return nil
	}

	detail := &ResultDetail{
		ID:          record.ExecutionID,
		MemberID:    record.MemberID,
		TriggerType: record.TriggerType,
		Status:      record.Status,
		Name:        record.Name,
		Delivery:    record.Delivery,
	}

	// Set times
	if record.StartTime != nil {
		detail.StartTime = *record.StartTime
	}
	detail.EndTime = record.EndTime

	return detail
}
