//go:build k8s

package sandbox_test

import (
	"fmt"
	"os"

	"github.com/yaoapp/yao/tai"
	"github.com/yaoapp/yao/tai/types"
)

func init() {
	extraNodeProviders = append(extraNodeProviders, k8sNodes)
	extraHostExecProviders = append(extraHostExecProviders, k8sHostExec)
	extraPurgeProviders = append(extraPurgeProviders, k8sPurge)
}

func k8sNodes() []nodeConfig {
	host := os.Getenv("TAI_TEST_K8S_HOST")
	kubeconfig := os.Getenv("TAI_TEST_KUBECONFIG")
	if host == "" || kubeconfig == "" {
		return nil
	}
	grpcPort := envPort("TAI_TEST_K8S_GRPC_PORT", envPort("TAI_TEST_GRPC_PORT", 19100))
	dialOps := []tai.DialOption{
		tai.WithDialRuntime(types.K8s),
		tai.WithDialKubeConfig(kubeconfig),
	}
	if ns := os.Getenv("TAI_TEST_K8S_NAMESPACE"); ns != "" {
		dialOps = append(dialOps, tai.WithDialNamespace(ns))
	}
	return []nodeConfig{{
		Name:    "k8s",
		Addr:    fmt.Sprintf("tai://%s:%d", host, grpcPort),
		DialOps: dialOps,
	}}
}

func k8sHostExec() []hostExecTarget {
	host := os.Getenv("TAI_TEST_K8S_HOST")
	if host == "" {
		return nil
	}
	grpcPort := envPort("TAI_TEST_K8S_GRPC_PORT", envPort("TAI_TEST_GRPC_PORT", 19100))
	return []hostExecTarget{{Name: "k8s", Addr: fmt.Sprintf("%s:%d", host, grpcPort)}}
}

func k8sPurge() []purgeTarget {
	host := os.Getenv("TAI_TEST_K8S_HOST")
	kubeconfig := os.Getenv("TAI_TEST_KUBECONFIG")
	if host == "" || kubeconfig == "" {
		return nil
	}
	grpcPort := envPort("TAI_TEST_K8S_GRPC_PORT", envPort("TAI_TEST_GRPC_PORT", 19100))
	dialOps := []tai.DialOption{
		tai.WithDialRuntime(types.K8s),
		tai.WithDialKubeConfig(kubeconfig),
	}
	if ns := os.Getenv("TAI_TEST_K8S_NAMESPACE"); ns != "" {
		dialOps = append(dialOps, tai.WithDialNamespace(ns))
	}
	return []purgeTarget{{
		name:    "k8s",
		addr:    fmt.Sprintf("tai://%s:%d", host, grpcPort),
		dialOps: dialOps,
	}}
}
