package runtime

import "context"

// k8sImage is a no-op Image for K8s mode.
// Image pulling is handled by kubelet based on imagePullPolicy and imagePullSecrets.
type k8sImage struct{}

func NewK8sImage() Image { return &k8sImage{} }

func (k *k8sImage) Exists(_ context.Context, _ string) (bool, error) {
	return true, nil
}

func (k *k8sImage) Inspect(_ context.Context, _ string) (*ImageMeta, error) {
	return nil, nil
}

func (k *k8sImage) Pull(_ context.Context, _ string, _ PullOptions) (<-chan PullProgress, error) {
	return nil, nil
}

func (k *k8sImage) Remove(_ context.Context, _ string, _ bool) error {
	return nil
}

func (k *k8sImage) List(_ context.Context) ([]ImageInfo, error) {
	return nil, nil
}
