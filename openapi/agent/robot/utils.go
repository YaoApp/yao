package robot

import (
	"context"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/yaoapp/gou/model"
)

// GetLocale extracts locale from request
// Priority: query param > Accept-Language header > default
func GetLocale(c *gin.Context) string {
	// Check query param first
	if locale := c.Query("locale"); locale != "" {
		return strings.ToLower(strings.TrimSpace(locale))
	}

	// Check Accept-Language header
	if acceptLang := c.GetHeader("Accept-Language"); acceptLang != "" {
		// Parse first language from header (e.g., "en-US,en;q=0.9" -> "en-us")
		parts := strings.Split(acceptLang, ",")
		if len(parts) > 0 {
			lang := strings.Split(parts[0], ";")[0]
			return strings.ToLower(strings.TrimSpace(lang))
		}
	}

	// Default locale
	return "en-us"
}

// ParseBoolValue parses various string formats into a boolean pointer
func ParseBoolValue(value string) *bool {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "1", "true", "yes", "on":
		v := true
		return &v
	case "0", "false", "no", "off":
		v := false
		return &v
	}
	return nil
}

// ==================== Member ID Generation ====================
// Follows the same pattern as openapi/oauth/providers/user/utils.go

const memberModel = "__yao.member"

// GenerateMemberID generates a new unique member_id for robot creation
// Uses numeric ID (12 characters) with collision detection
func GenerateMemberID(ctx context.Context) (string, error) {
	const maxRetries = 10

	for i := 0; i < maxRetries; i++ {
		// Generate 12-digit numeric ID (matches existing pattern)
		id, err := gonanoid.Generate("0123456789", 12)
		if err != nil {
			return "", fmt.Errorf("failed to generate member_id: %w", err)
		}

		// Check if ID already exists
		exists, err := memberIDExists(ctx, id)
		if err != nil {
			return "", fmt.Errorf("failed to check member_id existence: %w", err)
		}

		if !exists {
			return id, nil
		}
		// ID exists, retry
	}

	return "", fmt.Errorf("failed to generate unique member_id after %d retries", maxRetries)
}

// memberIDExists checks if a member_id already exists in the database
func memberIDExists(ctx context.Context, memberID string) (bool, error) {
	m := model.Select(memberModel)
	if m == nil {
		return false, fmt.Errorf("model %s not found", memberModel)
	}

	members, err := m.Get(model.QueryParam{
		Select: []interface{}{"id"},
		Wheres: []model.QueryWhere{
			{Column: "member_id", Value: memberID},
		},
		Limit: 1,
	})

	if err != nil {
		return false, err
	}

	return len(members) > 0, nil
}
