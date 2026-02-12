package rss

import (
	"encoding/json"
	"fmt"
)

// mapToFeed converts an arbitrary value (typically a map from Process args)
// into a Feed struct. It uses JSON marshaling/unmarshaling as a safe
// intermediate conversion, which handles nested maps, slices, and type coercion.
func mapToFeed(v interface{}) (*Feed, error) {
	if v == nil {
		return nil, fmt.Errorf("feed data is nil")
	}

	// If already a *Feed, return directly
	if feed, ok := v.(*Feed); ok {
		return feed, nil
	}

	// If it's a Feed value (not pointer), take its address
	if feed, ok := v.(Feed); ok {
		return &feed, nil
	}

	// Otherwise, marshal to JSON and unmarshal to Feed
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize feed data: %s", err.Error())
	}

	var feed Feed
	if err := json.Unmarshal(data, &feed); err != nil {
		return nil, fmt.Errorf("failed to parse feed data: %s", err.Error())
	}

	return &feed, nil
}

// mapToFetchOptions converts an arbitrary value (typically a map from Process args)
// into a FetchOptions struct. Returns default options for nil input.
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
