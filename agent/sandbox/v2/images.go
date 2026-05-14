package sandboxv2

import (
	_ "embed"

	"gopkg.in/yaml.v3"
)

//go:embed images.yml
var imagesYAML []byte

// PresetImage describes a curated sandbox image.
type PresetImage struct {
	Name        string   `json:"name" yaml:"name"`
	Image       string   `json:"image" yaml:"image"`
	Description string   `json:"description" yaml:"description"`
	Runners     []string `json:"runners" yaml:"runners"`
	Features    []string `json:"features" yaml:"features"`
}

// PresetImages lists the curated sandbox images available for agents.
// Loaded from the embedded images.yml at init time.
var PresetImages []PresetImage

func init() {
	if err := yaml.Unmarshal(imagesYAML, &PresetImages); err != nil {
		panic("sandbox/v2: failed to parse embedded images.yml: " + err.Error())
	}
}
