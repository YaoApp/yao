package utils_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/yao/utils"
)

func TestProcessUnauthorized(t *testing.T) {
	utils.Init()
	proc := process.New("utils.throw.Unauthorized", "Authentication required")
	err := proc.Execute()
	assert.NotNil(t, err)
	assert.Equal(t, "Exception|401: Authentication required", err.Error())
}

func TestProcessForbidden(t *testing.T) {
	utils.Init()
	proc := process.New("utils.throw.Forbidden", "Access denied")
	err := proc.Execute()
	assert.NotNil(t, err)
	assert.Equal(t, "Exception|403: Access denied", err.Error())
}

func TestProcessNotFound(t *testing.T) {
	utils.Init()
	proc := process.New("utils.throw.NotFound", "Resource not found")
	err := proc.Execute()
	assert.NotNil(t, err)
	assert.Equal(t, "Exception|404: Resource not found", err.Error())
}

func TestProcessBadRequest(t *testing.T) {
	utils.Init()
	proc := process.New("utils.throw.BadRequest", "Bad Request")
	err := proc.Execute()
	assert.NotNil(t, err)
	assert.Equal(t, "Exception|400: Bad Request", err.Error())
}

func TestProcessInternalError(t *testing.T) {
	utils.Init()
	proc := process.New("utils.throw.InternalError", "Internal Error")
	err := proc.Execute()
	assert.NotNil(t, err)
	assert.Equal(t, "Exception|500: Internal Error", err.Error())
}

func TestProcessException(t *testing.T) {
	utils.Init()
	proc := process.New("utils.throw.Exception", "I'm a teapot", 418)
	err := proc.Execute()
	assert.NotNil(t, err)
	assert.Equal(t, "Exception|418: I'm a teapot", err.Error())
}
