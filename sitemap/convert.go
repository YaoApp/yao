package sitemap

import (
	"encoding/json"
	"fmt"
)

// mapToURLs converts an arbitrary value (typically []interface{} from Process args)
// into a []URL slice. It uses JSON marshaling/unmarshaling as a safe intermediate
// conversion, which handles nested maps, slices, and type coercion.
func mapToURLs(v interface{}) ([]URL, error) {
	if v == nil {
		return nil, fmt.Errorf("urls data is nil")
	}

	// If already []URL, return directly
	if urls, ok := v.([]URL); ok {
		return urls, nil
	}

	// Otherwise, marshal to JSON and unmarshal to []URL
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize urls data: %s", err.Error())
	}

	var urls []URL
	if err := json.Unmarshal(data, &urls); err != nil {
		return nil, fmt.Errorf("failed to parse urls data: %s", err.Error())
	}

	return urls, nil
}

// mapToBuildOptions converts an arbitrary value (typically map[string]interface{})
// into a BuildOptions struct.
func mapToBuildOptions(v interface{}) (*BuildOptions, error) {
	if v == nil {
		return nil, fmt.Errorf("build options is nil")
	}

	if opts, ok := v.(*BuildOptions); ok {
		return opts, nil
	}
	if opts, ok := v.(BuildOptions); ok {
		return &opts, nil
	}

	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize build options: %s", err.Error())
	}

	var opts BuildOptions
	if err := json.Unmarshal(data, &opts); err != nil {
		return nil, fmt.Errorf("failed to parse build options: %s", err.Error())
	}

	return &opts, nil
}

// mapToDiscoverOptions converts an arbitrary value into a DiscoverOptions struct.
func mapToDiscoverOptions(v interface{}) (*DiscoverOptions, error) {
	if v == nil {
		return &DiscoverOptions{}, nil
	}

	if opts, ok := v.(*DiscoverOptions); ok {
		return opts, nil
	}
	if opts, ok := v.(DiscoverOptions); ok {
		return &opts, nil
	}

	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize discover options: %s", err.Error())
	}

	var opts DiscoverOptions
	if err := json.Unmarshal(data, &opts); err != nil {
		return nil, fmt.Errorf("failed to parse discover options: %s", err.Error())
	}

	return &opts, nil
}

// mapToFetchOptions converts an arbitrary value into a FetchOptions struct.
func mapToFetchOptions(v interface{}) (*FetchOptions, error) {
	if v == nil {
		return &FetchOptions{}, nil
	}

	if opts, ok := v.(*FetchOptions); ok {
		return opts, nil
	}
	if opts, ok := v.(FetchOptions); ok {
		return &opts, nil
	}

	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize fetch options: %s", err.Error())
	}

	var opts FetchOptions
	if err := json.Unmarshal(data, &opts); err != nil {
		return nil, fmt.Errorf("failed to parse fetch options: %s", err.Error())
	}

	return &opts, nil
}
