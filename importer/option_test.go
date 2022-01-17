package importer

import (
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
)

var testDataOption = map[string][]byte{
	"normal": []byte(`{
		"autoMatching": true,
		"chunkSize":200,
		"mappingPreview": "always",
		"dataPreview": "never"
	}`),
	"defaults": []byte(`{}`),
	"failure":  []byte(`""`),
}

func TestOptionUnmarshalJSON(t *testing.T) {
	var normal Option
	err := jsoniter.Unmarshal(testDataOption["normal"], &normal)
	assert.Nil(t, err)
	assert.Equal(t, true, normal.UseTemplate)
	assert.Equal(t, 200, normal.ChunkSize)
	assert.Equal(t, "always", normal.MappingPreview)
	assert.Equal(t, "never", normal.DataPreview)

	var defaults Option
	err = jsoniter.Unmarshal(testDataOption["defaults"], &defaults)
	assert.Nil(t, err)
	assert.Equal(t, true, defaults.UseTemplate)
	assert.Equal(t, 500, defaults.ChunkSize)
	assert.Equal(t, "auto", defaults.MappingPreview)
	assert.Equal(t, "auto", defaults.DataPreview)

	var failure Option
	err = jsoniter.Unmarshal(testDataOption["failure"], &failure)
	assert.NotNil(t, err)
}
