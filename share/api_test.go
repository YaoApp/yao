package share

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/session"
)

func TestGetQueryParam(t *testing.T) {
	sid := session.ID()
	s := session.Global().ID(sid).Expire(5000 * time.Microsecond)
	s.MustSet("id", 10086)
	s.MustSet("extra", map[string]interface{}{"gender": "男"})
	query := map[string]interface{}{
		"select": []string{"id", "name"},
		"wheres": []map[string]interface{}{
			{"column": "id", "op": "=", "value": "{{id}}"},
			{"column": "gender", "op": "=", "value": "{{extra.gender}}"},
		},
	}
	param := GetQueryParam(query, sid)
	assert.Equal(t, "id", param.Wheres[0].Column)
	assert.Equal(t, "=", param.Wheres[0].OP)
	assert.Equal(t, float64(10086), param.Wheres[0].Value)
	assert.Equal(t, "gender", param.Wheres[1].Column)
	assert.Equal(t, "=", param.Wheres[1].OP)
	assert.Equal(t, "男", param.Wheres[1].Value)
}
