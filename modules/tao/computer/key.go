package computer

import (
	"context"
	"fmt"
	"strings"

	"github.com/yaoapp/gou/model"
)

// KeyResolver resolves a user's bound Tao API key.
type KeyResolver interface {
	ResolveTaoKey(context.Context, string) (string, error)
}

// ModelKeyResolver reads the key from __yao.user.tao_api_key.
type ModelKeyResolver struct{}

// ResolveTaoKey returns the active Tao key for a Yao user.
func (ModelKeyResolver) ResolveTaoKey(ctx context.Context, userID string) (string, error) {
	if strings.TrimSpace(userID) == "" {
		return "", fmt.Errorf("user_id is required")
	}
	m := model.Select("__yao.user")
	if m == nil {
		return "", fmt.Errorf("user model is not loaded")
	}
	rows, err := m.Get(model.QueryParam{Wheres: []model.QueryWhere{{Column: "user_id", Value: userID}}})
	if err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "", fmt.Errorf("tao key is not bound")
	}
	key, _ := rows[0]["tao_api_key"].(string)
	if key == "" {
		return "", fmt.Errorf("tao key is empty")
	}
	return key, nil
}

// BindKey stores an encrypted Tao key on the user row.
func BindKey(ctx context.Context, userID, key string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(userID) == "" || !strings.HasPrefix(key, "sk-tao-") {
		return fmt.Errorf("invalid user_id or tao key")
	}
	m := model.Select("__yao.user")
	if m == nil {
		return fmt.Errorf("user model is not loaded")
	}
	rows, err := m.Get(model.QueryParam{Wheres: []model.QueryWhere{{Column: "user_id", Value: userID}}})
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return fmt.Errorf("user not found")
	}
	id := rows[0]["id"]
	if id == nil || fmt.Sprint(id) == "" {
		return fmt.Errorf("user id is empty")
	}
	return m.Update(id, map[string]interface{}{"tao_api_key": key})
}

// DeleteKey removes a user's bound Tao key.
func DeleteKey(ctx context.Context, userID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	m := model.Select("__yao.user")
	if m == nil {
		return fmt.Errorf("user model is not loaded")
	}
	rows, err := m.Get(model.QueryParam{Wheres: []model.QueryWhere{{Column: "user_id", Value: userID}}})
	if err != nil {
		return err
	}
	for _, row := range rows {
		id := row["id"]
		if id == nil || fmt.Sprint(id) == "" {
			continue
		}
		if err := m.Update(id, map[string]interface{}{"tao_api_key": ""}); err != nil {
			return err
		}
	}
	return nil
}

// HasKey reports whether the user has a bound Tao key.
func HasKey(ctx context.Context, userID string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	m := model.Select("__yao.user")
	if m == nil {
		return false, fmt.Errorf("user model is not loaded")
	}
	rows, err := m.Get(model.QueryParam{Wheres: []model.QueryWhere{{Column: "user_id", Value: userID}}})
	if err != nil {
		return false, err
	}
	if len(rows) == 0 {
		return false, nil
	}
	key, _ := rows[0]["tao_api_key"].(string)
	return key != "", nil
}
