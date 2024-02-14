package pipe

import (
	"crypto/md5"
	"fmt"
)

func ref(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))[:6]
}

func promptsToMap(prompts []Prompt) []map[string]interface{} {
	maps := []map[string]interface{}{}
	for _, prompt := range prompts {
		maps = append(maps, map[string]interface{}{
			"role":    prompt.Role,
			"content": prompt.Content,
		})
	}
	return maps
}

func (promt Prompt) finger() string {
	raw := fmt.Sprintf("%s|%s", promt.Role, promt.Content)
	return fmt.Sprintf("%x", md5.Sum([]byte(raw)))
}
