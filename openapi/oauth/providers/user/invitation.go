package user

import (
	"context"
	"fmt"
	"time"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/kun/maps"
)

// Invitation Code Resource (Official Platform Invitation Codes)

// CreateInvitationCodes creates invitation codes in batch
// Supports creating multiple invitation codes at once for efficiency
func (u *DefaultUser) CreateInvitationCodes(ctx context.Context, codeData []maps.MapStrAny) ([]string, error) {
	if len(codeData) == 0 {
		return []string{}, nil
	}

	codes := make([]string, 0, len(codeData))
	m := model.Select(u.invitationModel)

	// Validate and prepare data - collect all possible columns
	columnsSet := make(map[string]bool)
	columnsSet["code"] = true
	columnsSet["status"] = true
	columnsSet["is_published"] = true
	columnsSet["code_type"] = true

	for i := range codeData {
		// Validate required fields
		code, hasCode := codeData[i]["code"].(string)
		if !hasCode || code == "" {
			return nil, fmt.Errorf("code is required in codeData at index %d", i)
		}

		// Set default values if not provided
		if _, exists := codeData[i]["status"]; !exists {
			codeData[i]["status"] = "draft"
		}
		if _, exists := codeData[i]["is_published"]; !exists {
			codeData[i]["is_published"] = false
		}
		if _, exists := codeData[i]["code_type"]; !exists {
			codeData[i]["code_type"] = "official"
		}

		// Collect optional columns
		for _, col := range []string{"owner_id", "description", "source", "expires_at", "metadata"} {
			if _, exists := codeData[i][col]; exists {
				columnsSet[col] = true
			}
		}

		codes = append(codes, code)
	}

	// Build ordered column list
	columns := []string{"code", "status", "is_published", "code_type"}
	for _, col := range []string{"owner_id", "description", "source", "expires_at", "metadata"} {
		if columnsSet[col] {
			columns = append(columns, col)
		}
	}

	// Build values matrix
	values := make([][]interface{}, 0, len(codeData))
	for i := range codeData {
		row := make([]interface{}, len(columns))
		for j, col := range columns {
			if val, exists := codeData[i][col]; exists {
				row[j] = val
			} else {
				row[j] = nil
			}
		}
		values = append(values, row)
	}

	// Batch insert
	err := m.Insert(columns, values)
	if err != nil {
		return nil, fmt.Errorf(ErrFailedToCreateInvitationCode, err)
	}

	return codes, nil
}

// UseInvitationCode marks an invitation code as used (redemption)
// This is called when a user successfully uses an invitation code during registration
func (u *DefaultUser) UseInvitationCode(ctx context.Context, code string, userID string) error {
	m := model.Select(u.invitationModel)

	// First, get the invitation code to validate it
	invitations, err := m.Get(model.QueryParam{
		Select: []interface{}{"id", "code", "status", "is_published", "expires_at", "used_by"},
		Wheres: []model.QueryWhere{
			{Column: "code", Value: code},
		},
		Limit: 1,
	})

	if err != nil {
		return fmt.Errorf(ErrFailedToUseInvitationCode, err)
	}

	if len(invitations) == 0 {
		return fmt.Errorf(ErrInvitationCodeNotFound)
	}

	invitation := invitations[0]

	// Check if already used
	if usedBy := invitation["used_by"]; usedBy != nil && usedBy != "" {
		return fmt.Errorf(ErrInvitationCodeAlreadyUsed)
	}

	// Check if published (handle both bool and int64 types from different databases)
	isPublished := false
	switch v := invitation["is_published"].(type) {
	case bool:
		isPublished = v
	case int64:
		isPublished = v != 0
	case int:
		isPublished = v != 0
	}
	if !isPublished {
		return fmt.Errorf(ErrInvitationCodeNotPublished)
	}

	// Check status
	status, ok := invitation["status"].(string)
	if !ok || status != "active" {
		return fmt.Errorf("invitation code status must be 'active' to use, current status: %s", status)
	}

	// Check if expired
	if expiresAt := invitation["expires_at"]; expiresAt != nil {
		if expired, err := checkTimeExpired(expiresAt); err == nil && expired {
			return fmt.Errorf(ErrInvitationCodeExpired)
		}
	}

	// Mark as used
	updateData := maps.MapStrAny{
		"used_by": userID,
		"used_at": time.Now(),
		"status":  "used",
	}

	affected, err := m.UpdateWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "code", Value: code},
		},
		Limit: 1,
	}, updateData)

	if err != nil {
		return fmt.Errorf(ErrFailedToUseInvitationCode, err)
	}

	if affected == 0 {
		return fmt.Errorf(ErrInvitationCodeNotFound)
	}

	return nil
}

// DeleteInvitationCode soft deletes an invitation code
func (u *DefaultUser) DeleteInvitationCode(ctx context.Context, code string) error {
	// First check if invitation code exists
	m := model.Select(u.invitationModel)
	invitations, err := m.Get(model.QueryParam{
		Select: []interface{}{"id", "code"},
		Wheres: []model.QueryWhere{
			{Column: "code", Value: code},
		},
		Limit: 1,
	})

	if err != nil {
		return fmt.Errorf(ErrFailedToDeleteInvitationCode, err)
	}

	if len(invitations) == 0 {
		return fmt.Errorf(ErrInvitationCodeNotFound)
	}

	// Proceed with soft delete
	affected, err := m.DeleteWhere(model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "code", Value: code},
		},
		Limit: 1,
	})

	if err != nil {
		return fmt.Errorf(ErrFailedToDeleteInvitationCode, err)
	}

	if affected == 0 {
		return fmt.Errorf(ErrInvitationCodeNotFound)
	}

	return nil
}
