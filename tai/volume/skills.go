package volume

import (
	"bufio"
	"os"
	"strings"
)

// parseSkillFrontmatter reads the first 10 lines of a SKILL.md looking for
// YAML frontmatter fields "name" and "description".
func parseSkillFrontmatter(path string) (name, description string) {
	f, err := os.Open(path)
	if err != nil {
		return "", ""
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	inFrontmatter := false
	lineCount := 0

	for scanner.Scan() && lineCount < 10 {
		lineCount++
		line := scanner.Text()

		if lineCount == 1 && strings.TrimSpace(line) == "---" {
			inFrontmatter = true
			continue
		}
		if inFrontmatter && strings.TrimSpace(line) == "---" {
			break
		}
		if !inFrontmatter {
			break
		}

		if k, v, ok := parseYAMLField(line); ok {
			switch k {
			case "name":
				name = v
			case "description":
				description = v
			}
		}
	}
	return name, description
}

func parseYAMLField(line string) (key, value string, ok bool) {
	idx := strings.IndexByte(line, ':')
	if idx < 0 {
		return "", "", false
	}
	key = strings.TrimSpace(line[:idx])
	value = strings.TrimSpace(line[idx+1:])
	if key == "" {
		return "", "", false
	}
	return key, value, true
}
