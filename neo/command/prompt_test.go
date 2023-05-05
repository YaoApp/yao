package command

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/kun/maps"
)

func TestPromptReplace(t *testing.T) {

	prompt := Prompt{
		Role:    "{{ role }}",
		Name:    "{{ name }}",
		Content: "{{ content }}",
	}

	data := maps.Of(map[string]interface{}{
		"role":    "User",
		"name":    "Name",
		"content": "- Content\n",
	}).Dot()

	prompt = prompt.Replace(data)
	assert.Equal(t, "User", prompt.Role)
	assert.Equal(t, "Name", prompt.Name)
	assert.Equal(t, "- Content\n", prompt.Content)

	prompt = Prompt{
		Role:    "Role",
		Name:    "{{ notfound }}",
		Content: "{{ content }}",
	}

	prompt = prompt.Replace(data)
	assert.Equal(t, "Role", prompt.Role)
	assert.Equal(t, "", prompt.Name)
	assert.Equal(t, "- Content\n", prompt.Content)
}
