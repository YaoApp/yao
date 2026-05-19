//go:build unit

package xun_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yaoapp/yao/agent/store/xun"
)

type testStruct struct{ Name string }

func TestIsNil(t *testing.T) {
	t.Run("UntypedNil", func(t *testing.T) {
		assert.True(t, xun.IsNilForTest(nil))
	})

	t.Run("TypedNilPointer", func(t *testing.T) {
		var p *testStruct
		assert.True(t, xun.IsNilForTest(p))
	})

	t.Run("TypedNilMap", func(t *testing.T) {
		var m map[string]string
		assert.True(t, xun.IsNilForTest(m))
	})

	t.Run("TypedNilSlice", func(t *testing.T) {
		var s []string
		assert.True(t, xun.IsNilForTest(s))
	})

	t.Run("NonNilPointer", func(t *testing.T) {
		p := &testStruct{Name: "test"}
		assert.False(t, xun.IsNilForTest(p))
	})

	t.Run("NonNilEmptyMap", func(t *testing.T) {
		m := map[string]string{}
		assert.False(t, xun.IsNilForTest(m))
	})

	t.Run("NonNilMap", func(t *testing.T) {
		m := map[string]string{"a": "1"}
		assert.False(t, xun.IsNilForTest(m))
	})

	t.Run("NonNilEmptySlice", func(t *testing.T) {
		s := []string{}
		assert.False(t, xun.IsNilForTest(s))
	})

	t.Run("NonNilSlice", func(t *testing.T) {
		s := []string{"a"}
		assert.False(t, xun.IsNilForTest(s))
	})

	t.Run("ScalarInt", func(t *testing.T) {
		assert.False(t, xun.IsNilForTest(42))
	})

	t.Run("ScalarString", func(t *testing.T) {
		assert.False(t, xun.IsNilForTest("hello"))
	})

	t.Run("ScalarEmptyString", func(t *testing.T) {
		assert.False(t, xun.IsNilForTest(""))
	})

	t.Run("ScalarBool", func(t *testing.T) {
		assert.False(t, xun.IsNilForTest(false))
	})
}

func TestMarshalJSONFields(t *testing.T) {
	t.Run("SkipUntypedNil", func(t *testing.T) {
		data := make(map[string]interface{})
		err := xun.MarshalJSONFieldsForTest(data, map[string]interface{}{
			"field1": nil,
		})
		require.NoError(t, err)
		_, exists := data["field1"]
		assert.False(t, exists, "untyped nil should be skipped")
	})

	t.Run("SkipTypedNilPointer", func(t *testing.T) {
		data := make(map[string]interface{})
		var p *testStruct
		err := xun.MarshalJSONFieldsForTest(data, map[string]interface{}{
			"kb": p,
		})
		require.NoError(t, err)
		_, exists := data["kb"]
		assert.False(t, exists, "typed nil pointer should be skipped")
	})

	t.Run("SkipTypedNilMap", func(t *testing.T) {
		data := make(map[string]interface{})
		var m map[string]string
		err := xun.MarshalJSONFieldsForTest(data, map[string]interface{}{
			"deps": m,
		})
		require.NoError(t, err)
		_, exists := data["deps"]
		assert.False(t, exists, "typed nil map should be skipped")
	})

	t.Run("SkipTypedNilSlice", func(t *testing.T) {
		data := make(map[string]interface{})
		var s []string
		err := xun.MarshalJSONFieldsForTest(data, map[string]interface{}{
			"tags": s,
		})
		require.NoError(t, err)
		_, exists := data["tags"]
		assert.False(t, exists, "typed nil slice should be skipped")
	})

	t.Run("MarshalNonNilMap", func(t *testing.T) {
		data := make(map[string]interface{})
		err := xun.MarshalJSONFieldsForTest(data, map[string]interface{}{
			"deps": map[string]string{"echo": "^1.0.0"},
		})
		require.NoError(t, err)
		assert.Equal(t, `{"echo":"^1.0.0"}`, data["deps"])
	})

	t.Run("MarshalEmptyMap", func(t *testing.T) {
		data := make(map[string]interface{})
		err := xun.MarshalJSONFieldsForTest(data, map[string]interface{}{
			"deps": map[string]string{},
		})
		require.NoError(t, err)
		assert.Equal(t, `{}`, data["deps"])
	})

	t.Run("MarshalSlice", func(t *testing.T) {
		data := make(map[string]interface{})
		err := xun.MarshalJSONFieldsForTest(data, map[string]interface{}{
			"tags": []string{"ai", "bot"},
		})
		require.NoError(t, err)
		assert.Equal(t, `["ai","bot"]`, data["tags"])
	})

	t.Run("MarshalPointerToStruct", func(t *testing.T) {
		data := make(map[string]interface{})
		err := xun.MarshalJSONFieldsForTest(data, map[string]interface{}{
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

		err := xun.MarshalJSONFieldsForTest(data, map[string]interface{}{
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

func TestNanoToTime(t *testing.T) {
	t.Run("RoundTrip", func(t *testing.T) {
		now := time.Now()
		nano := xun.TimeToNanoForTest(now)
		got := xun.NanoToTimeForTest(nano)
		assert.Equal(t, now.UnixNano(), got.UnixNano())
	})

	t.Run("ZeroValue", func(t *testing.T) {
		got := xun.NanoToTimeForTest(0)
		assert.True(t, got.IsZero() || got.UnixNano() == 0)
	})

	t.Run("KnownTimestamp", func(t *testing.T) {
		ts := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
		nano := xun.TimeToNanoForTest(ts)
		got := xun.NanoToTimeForTest(nano)
		assert.Equal(t, ts.UnixNano(), got.UnixNano())
	})
}

func TestTimeToNano(t *testing.T) {
	t.Run("RoundTrip", func(t *testing.T) {
		now := time.Now()
		nano := xun.TimeToNanoForTest(now)
		got := xun.NanoToTimeForTest(nano)
		assert.Equal(t, now.UnixNano(), got.UnixNano())
	})

	t.Run("ZeroValue", func(t *testing.T) {
		var zero time.Time
		nano := xun.TimeToNanoForTest(zero)
		assert.Equal(t, int64(0), nano)
	})

	t.Run("CurrentTime", func(t *testing.T) {
		now := time.Now()
		nano := xun.TimeToNanoForTest(now)
		assert.Equal(t, now.UnixNano(), nano)
	})
}
