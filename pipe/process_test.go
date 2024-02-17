package pipe

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/utils"
	"github.com/yaoapp/yao/test"
)

func TestProcessPipes(t *testing.T) {
	prepare(t)
	defer test.Clean()

	p, err := process.Of("pipes.cli.translator", map[string]interface{}{"placeholder": "translate\nhello world"})
	if err != nil {
		t.Fatal(err)
	}

	output, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	utils.Dump(output)
	res := any.Of(output).Map().MapStrAny.Dot()
	assert.True(t, res.Has("global"))
	assert.True(t, res.Has("input"))
	assert.True(t, res.Has("output"))
	assert.True(t, res.Has("sid"))
	assert.True(t, res.Has("switch"))
	assert.Equal(t, "translate\nhello world", res.Get("input[0].placeholder"))
	assert.Len(t, res.Get("switch"), 2)
}

func TestProcessRun(t *testing.T) {
	prepare(t)
	defer test.Clean()

	p, err := process.Of("pipe.Run", "cli.translator", map[string]interface{}{"placeholder": "translate\nhello world"})
	if err != nil {
		t.Fatal(err)
	}

	output, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	res := any.Of(output).Map().MapStrAny.Dot()
	assert.True(t, res.Has("global"))
	assert.True(t, res.Has("input"))
	assert.True(t, res.Has("output"))
	assert.True(t, res.Has("sid"))
	assert.True(t, res.Has("switch"))
	assert.Equal(t, "translate\nhello world", res.Get("input[0].placeholder"))
	assert.Len(t, res.Get("switch"), 2)
}

func TestProcessCreate(t *testing.T) {

	prepare(t)
	defer test.Clean()

	dsl := `{
		"whitelist": ["utils.fmt.Print"],
		"name": "test",
		"label": "Test",
		"nodes": [
			{
				"name": "print",
				"process": {"name":"utils.fmt.Print", "args": "{{ $in }}"},
				"output": "print"
			}
		],
		"output": {"input": "{{ $input }}" }
	}`

	p, err := process.Of("pipe.Create", dsl, "hello world")
	if err != nil {
		t.Fatal(err)
	}

	output, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	res := any.Of(output).Map().MapStrAny.Dot()
	assert.Equal(t, "hello world", res.Get("input[0]"))
}

func TestProcessCreateWith(t *testing.T) {
	prepare(t)
	defer test.Clean()

	dsl := `{
		"whitelist": ["utils.fmt.Print"],
		"name": "test",
		"label": "Test",
		"input": "{{ $global.placeholder }}",
		"nodes": [
			{
				"name": "print",
				"process": {"name":"utils.fmt.Print", "args": "{{ $in }}"},
				"output": "print"
			}
		],
		"output": {"input": "{{ $input }}" }
	}`

	p, err := process.Of("pipe.CreateWith", dsl, map[string]interface{}{"placeholder": "hello world"})
	if err != nil {
		t.Fatal(err)
	}

	output, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	res := any.Of(output).Map().MapStrAny.Dot()
	assert.Equal(t, "hello world", res.Get("input[0]"))

}

func TestProcessResume(t *testing.T) {
	prepare(t)
	defer test.Clean()

	p, err := process.Of("pipe.Run", "web.translator", "hello web world")
	if err != nil {
		t.Fatal(err)
	}

	web, err := p.Exec()
	resume := web.(ResumeContext)

	p, err = process.Of("pipe.Resume", resume.ID, "translate", "hello web world")
	output, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	res := any.Of(output).Map().MapStrAny.Dot()
	assert.True(t, res.Has("global"))
	assert.True(t, res.Has("input"))
	assert.True(t, res.Has("output"))
	assert.True(t, res.Has("sid"))
	assert.True(t, res.Has("switch"))
	assert.Equal(t, "hello web world", res.Get("input[0]"))
	assert.Len(t, res.Get("switch"), 2)
}

func TestProcessResumeWith(t *testing.T) {
	prepare(t)
	defer test.Clean()

	p, err := process.Of("pipe.Run", "web.translator", "hello web world")
	if err != nil {
		t.Fatal(err)
	}

	web, err := p.Exec()
	resume := web.(ResumeContext)

	p, err = process.Of("pipe.ResumeWith", resume.ID, map[string]interface{}{"foo": "bar"}, "translate", "hello web world")
	output, err := p.Exec()
	if err != nil {
		t.Fatal(err)
	}

	res := any.Of(output).Map().MapStrAny.Dot()
	assert.True(t, res.Has("global"))
	assert.True(t, res.Has("input"))
	assert.True(t, res.Has("output"))
	assert.True(t, res.Has("sid"))
	assert.True(t, res.Has("switch"))
	assert.Equal(t, "hello web world", res.Get("input[0]"))
	assert.Equal(t, "bar", res.Get("global.foo"))
	assert.Len(t, res.Get("switch"), 2)
}

func TestProcessClose(t *testing.T) {
	prepare(t)
	defer test.Clean()

	p, err := process.Of("pipe.Run", "web.translator", "hello web world")
	if err != nil {
		t.Fatal(err)
	}

	web, err := p.Exec()
	resume := web.(ResumeContext)

	p, err = process.Of("pipe.Close", resume.ID)
	p.Exec()

	p, err = process.Of("pipe.Resume", resume.ID, "translate", "hello web world")
	_, err = p.Exec()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "not found")
}
