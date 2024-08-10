package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
)

func TestTemplateRender(t *testing.T) {
	prepare(t)
	defer clean()

	args := []any{"test", "advanced", "<div> {{ name}} </div>", map[string]any{"name": "test"}}
	p, err := process.Of("sui.template.render", args...)
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}
	assert.Contains(t, res, "<div> test ")
}

func TestTemplateRenderWithComponent(t *testing.T) {
	prepare(t)
	defer clean()
	source := `
	  <div>
		<h2>Component Render {{ name }} </h2>
		<div style="display: flex; gap: 20px">
		<div s:for="{{ ['foo', 'bar'] }}">
			<Component
				is="/backend/{{ item }}"
				world="World"
				hello="{{ hello }}"
				index="{{ index }}"
				pets="{{ ['cat', 'dog'] }}"
			>
			{{ upper(item) }} {{ name }}
			</Component>
		</div>
		</div>
	</div>
  `
	args := []any{"test", "advanced", source, map[string]any{"name": "test"}, map[string]any{"theme": "dark"}}
	p, err := process.Of("sui.template.render", args...)
	if err != nil {
		t.Fatal(err)
	}

	res, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Contains(t, res, "Component Render test")
	assert.Contains(t, res, "FOO test")
	assert.Contains(t, res, "BAR test")
}
