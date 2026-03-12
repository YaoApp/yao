//go:build containerized

package sandbox_test

import (
	"fmt"
	"os"
)

func init() {
	extraNodeProviders = append(extraNodeProviders, containerizedNodes)
	extraPurgeProviders = append(extraPurgeProviders, containerizedPurge)
}

func containerizedNodes() []nodeConfig {
	host := os.Getenv("TAI_TEST_CONTAINERIZED_HOST")
	if host == "" {
		return nil
	}
	grpcPort := envPort("TAI_TEST_CONTAINERIZED_GRPC_PORT", 9200)
	return []nodeConfig{{
		Name: "containerized",
		Addr: fmt.Sprintf("tai://%s:%d", host, grpcPort),
	}}
}

func containerizedPurge() []purgeTarget {
	host := os.Getenv("TAI_TEST_CONTAINERIZED_HOST")
	if host == "" {
		return nil
	}
	grpcPort := envPort("TAI_TEST_CONTAINERIZED_GRPC_PORT", 9200)
	return []purgeTarget{{
		name: "containerized",
		addr: fmt.Sprintf("tai://%s:%d", host, grpcPort),
	}}
}
