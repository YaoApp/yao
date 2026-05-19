package xun

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testStruct struct{ Name string }

func TestIsNil(t *testing.T) {
	// Untyped nil
	t.Run("UntypedNil", func(t *testing.T) {
		assert.True(t, isNil(nil))
	})

	// Typed nil pointer
	t.Run("TypedNilPointer", func(t *testing.T) {
		var p *testStruct
		assert.True(t, isNil(p))
	})

	// Typed nil map
	t.Run("TypedNilMap", func(t *testing.T) {
		var m map[string]string
		assert.True(t, isNil(m))
	})

	// Typed nil slice
	t.Run("TypedNilSlice", func(t *testing.T) {
		var s []string
		assert.True(t, isNil(s))
	})

	// Non-nil pointer
	t.Run("NonNilPointer", func(t *testing.T) {
		p := &testStruct{Name: "test"}
		assert.False(t, isNil(p))
	})

	// Non-nil map (empty)
	t.Run("NonNilEmptyMap", func(t *testing.T) {
		m := map[string]string{}
		assert.False(t, isNil(m))
	})

	// Non-nil map with values
	t.Run("NonNilMap", func(t *testing.T) {
		m := map[string]string{"a": "1"}
		assert.False(t, isNil(m))
	})

	// Non-nil slice (empty)
	t.Run("NonNilEmptySlice", func(t *testing.T) {
		s := []string{}
		assert.False(t, isNil(s))
	})

	// Non-nil slice with values
	t.Run("NonNilSlice", func(t *testing.T) {
		s := []string{"a"}
		assert.False(t, isNil(s))
	})

	// Scalar types (never nil)
	t.Run("String", func(t *testing.T) {
		assert.False(t, isNil("hello"))
	})
	t.Run("EmptyString", func(t *testing.T) {
		assert.False(t, isNil(""))
	})
	t.Run("Int", func(t *testing.T) {
		assert.False(t, isNil(42))
	})
	t.Run("Bool", func(t *testing.T) {
		assert.False(t, isNil(false))
	})
}

func TestMarshalJSONFields(t *testing.T) {
	t.Run("SkipUntypedNil", func(t *testing.T) {
		data := make(map[string]interface{})
		err := marshalJSONFields(data, map[string]interface{}{
			"field1": nil,
		})
		require.NoError(t, err)
		_, exists := data["field1"]
		assert.False(t, exists, "untyped nil should be skipped")
	})

	t.Run("SkipTypedNilMap", func(t *testing.T) {
		data := make(map[string]interface{})
		var m map[string]string
		err := marshalJSONFields(data, map[string]interface{}{
			"deps": m,
		})
		require.NoError(t, err)
		_, exists := data["deps"]
		assert.False(t, exists, "typed nil map should be skipped")
	})

	t.Run("SkipTypedNilSlice", func(t *testing.T) {
		data := make(map[string]interface{})
		var s []string
		err := marshalJSONFields(data, map[string]interface{}{
			"tags": s,
		})
		require.NoError(t, err)
		_, exists := data["tags"]
		assert.False(t, exists, "typed nil slice should be skipped")
	})

	t.Run("SkipTypedNilPointer", func(t *testing.T) {
		data := make(map[string]interface{})
		var p *testStruct
		err := marshalJSONFields(data, map[string]interface{}{
			"kb": p,
		})
		require.NoError(t, err)
		_, exists := data["kb"]
		assert.False(t, exists, "typed nil pointer should be skipped")
	})

	t.Run("MarshalNonNilMap", func(t *testing.T) {
		data := make(map[string]interface{})
		err := marshalJSONFields(data, map[string]interface{}{
			"deps": map[string]string{"echo": "^1.0.0"},
		})
		require.NoError(t, err)
		assert.Equal(t, `{"echo":"^1.0.0"}`, data["deps"])
	})

	t.Run("MarshalEmptyMap", func(t *testing.T) {
		data := make(map[string]interface{})
		err := marshalJSONFields(data, map[string]interface{}{
			"deps": map[string]string{},
		})
		require.NoError(t, err)
		assert.Equal(t, `{}`, data["deps"])
	})

	t.Run("MarshalSlice", func(t *testing.T) {
		data := make(map[string]interface{})
		err := marshalJSONFields(data, map[string]interface{}{
			"tags": []string{"ai", "bot"},
		})
		require.NoError(t, err)
		assert.Equal(t, `["ai","bot"]`, data["tags"])
	})

	t.Run("MarshalPointer", func(t *testing.T) {
		data := make(map[string]interface{})
		err := marshalJSONFields(data, map[string]interface{}{
			"kb": &testStruct{Name: "test"},
		})
		require.NoError(t, err)
		assert.Equal(t, `{"Name":"test"}`, data["kb"])
	})

	t.Run("MixedNilAndNonNil", func(t *testing.T) {
		data := make(map[string]interface{})
		var nilMap map[string]string
		var nilSlice []string
		var nilPtr *testStruct

		err := marshalJSONFields(data, map[string]interface{}{
			"nil_map":   nilMap,
			"nil_slice": nilSlice,
			"nil_ptr":   nilPtr,
			"nil_raw":   nil,
			"good_map":  map[string]string{"k": "v"},
			"good_list": []string{"a"},
		})
		require.NoError(t, err)

		assert.Len(t, data, 2, "only non-nil fields should be written")
		assert.Equal(t, `{"k":"v"}`, data["good_map"])
		assert.Equal(t, `["a"]`, data["good_list"])

		_, exists := data["nil_map"]
		assert.False(t, exists)
		_, exists = data["nil_slice"]
		assert.False(t, exists)
		_, exists = data["nil_ptr"]
		assert.False(t, exists)
		_, exists = data["nil_raw"]
		assert.False(t, exists)
	})
}
