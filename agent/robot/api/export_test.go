package api

import "github.com/yaoapp/yao/agent/robot/types"

// ApplyDefaults exposes applyDefaults for external tests.
func (q *ListQuery) ApplyDefaults() {
	q.applyDefaults()
}

// PaginateRobotsForTest exposes paginateRobots for external tests.
func PaginateRobotsForTest(robots []*types.Robot, query *ListQuery) *ListResult {
	return paginateRobots(robots, query)
}

// ExportLegacyResume exposes legacyResume for external tests.
func ExportLegacyResume(ctx *types.Context, req *InteractRequest) (*InteractResult, error) {
	return legacyResume(ctx, req)
}
