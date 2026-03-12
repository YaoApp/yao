package runtime

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	apiresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
)

// K8sOption configures a K8s runtime.
type K8sOption struct {
	Namespace  string // default "default"
	KubeConfig string // path to kubeconfig file
}

type k8sSandbox struct {
	cli    kubernetes.Interface
	cfg    *rest.Config
	ns     string
	labels map[string]string
}

// NewK8s creates a Runtime backed by Kubernetes via Tai's TCP proxy.
// addr should be "host:port" pointing to Tai's K8s proxy endpoint.
// kubeConfigPath must be an absolute path or will be resolved relative to the caller's working directory.
func NewK8s(addr string, opts ...K8sOption) (Runtime, error) {
	ns := "default"
	var kubeConfigPath string
	if len(opts) > 0 {
		if opts[0].Namespace != "" {
			ns = opts[0].Namespace
		}
		if opts[0].KubeConfig != "" {
			kubeConfigPath = opts[0].KubeConfig
			if !filepath.IsAbs(kubeConfigPath) {
				abs, err := filepath.Abs(kubeConfigPath)
				if err != nil {
					return nil, fmt.Errorf("resolve kubeconfig path: %w", err)
				}
				kubeConfigPath = abs
			}
		}
	}

	if kubeConfigPath == "" {
		return nil, fmt.Errorf("kubeconfig path is required for K8s runtime")
	}

	cfg, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return nil, fmt.Errorf("build kubeconfig: %w", err)
	}

	// Override the server address to point at the Tai proxy
	if addr != "" {
		cfg.Host = "https://" + addr
		// When connecting through Tai TCP proxy, skip TLS verification
		cfg.TLSClientConfig.Insecure = true
		cfg.TLSClientConfig.CAData = nil
		cfg.TLSClientConfig.CAFile = ""
	}

	cli, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("create k8s client: %w", err)
	}

	// Verify connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err = cli.CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
	if err != nil && !errors.IsNotFound(err) {
		return nil, fmt.Errorf("k8s connectivity check: %w", err)
	}

	return &k8sSandbox{
		cli: cli,
		cfg: cfg,
		ns:  ns,
		labels: map[string]string{
			"managed-by": "yao-tai-sdk",
		},
	}, nil
}

func (s *k8sSandbox) Create(ctx context.Context, opts CreateOptions) (string, error) {
	name := opts.Name
	if name == "" {
		name = fmt.Sprintf("sandbox-%d", time.Now().UnixNano())
	}
	// K8s names must be DNS-compatible
	name = strings.ToLower(name)
	name = strings.ReplaceAll(name, "_", "-")

	envVars := make([]corev1.EnvVar, 0, len(opts.Env))
	for k, v := range opts.Env {
		envVars = append(envVars, corev1.EnvVar{Name: k, Value: v})
	}

	container := corev1.Container{
		Name:       "main",
		Image:      opts.Image,
		Args:       opts.Cmd,
		Env:        envVars,
		WorkingDir: opts.WorkingDir,
	}

	if opts.Memory > 0 || opts.CPUs > 0 {
		container.Resources = buildResources(opts.Memory, opts.CPUs)
	}

	labels := make(map[string]string)
	for k, v := range s.labels {
		labels[k] = v
	}
	labels["sandbox-name"] = name
	for k, v := range opts.Labels {
		labels[k] = v
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: s.ns,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers:    []corev1.Container{container},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}

	if opts.User != "" {
		uid, err := parseUID(opts.User)
		if err == nil {
			pod.Spec.SecurityContext = &corev1.PodSecurityContext{
				RunAsUser: &uid,
			}
		}
	}

	created, err := s.cli.CoreV1().Pods(s.ns).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("create pod: %w", err)
	}
	return created.Name, nil
}

func (s *k8sSandbox) Start(ctx context.Context, id string) error {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 60*time.Second)
		defer cancel()
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		pod, err := s.cli.CoreV1().Pods(s.ns).Get(ctx, id, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("get pod: %w", err)
		}
		if pod.Status.Phase == corev1.PodRunning || pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("pod %s did not reach Running: %w", id, ctx.Err())
		case <-ticker.C:
		}
	}
}

func (s *k8sSandbox) Stop(ctx context.Context, id string, timeout time.Duration) error {
	secs := int64(timeout.Seconds())
	return s.cli.CoreV1().Pods(s.ns).Delete(ctx, id, metav1.DeleteOptions{
		GracePeriodSeconds: &secs,
	})
}

func (s *k8sSandbox) Remove(ctx context.Context, id string, force bool) error {
	opts := metav1.DeleteOptions{}
	if force {
		zero := int64(0)
		opts.GracePeriodSeconds = &zero
	}
	err := s.cli.CoreV1().Pods(s.ns).Delete(ctx, id, opts)
	if errors.IsNotFound(err) {
		return nil
	}
	return err
}

func (s *k8sSandbox) Exec(ctx context.Context, id string, cmd []string, opts ExecOptions) (*ExecResult, error) {
	execCmd := cmd
	if opts.WorkDir != "" || len(opts.Env) > 0 {
		var prefix string
		for k, v := range opts.Env {
			prefix += fmt.Sprintf("export %s=%q; ", k, v)
		}
		cdPart := ""
		if opts.WorkDir != "" {
			cdPart = fmt.Sprintf("cd %s && ", opts.WorkDir)
		}
		execCmd = []string{"sh", "-c", cdPart + prefix + strings.Join(cmd, " ")}
	}

	req := s.cli.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(id).
		Namespace(s.ns).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: "main",
			Command:   execCmd,
			Stdout:    true,
			Stderr:    true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(s.cfg, "POST", req.URL())
	if err != nil {
		return nil, fmt.Errorf("create executor: %w", err)
	}

	var stdout, stderr bytes.Buffer
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
	})

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(interface{ ExitStatus() int }); ok {
			exitCode = exitErr.ExitStatus()
			err = nil
		} else {
			return nil, fmt.Errorf("exec stream: %w", err)
		}
	}

	return &ExecResult{
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}, nil
}

func (s *k8sSandbox) ExecStream(ctx context.Context, id string, cmd []string, opts ExecOptions) (*StreamHandle, error) {
	execCmd := cmd
	if opts.WorkDir != "" || len(opts.Env) > 0 {
		var prefix string
		for k, v := range opts.Env {
			prefix += fmt.Sprintf("export %s=%q; ", k, v)
		}
		cdPart := ""
		if opts.WorkDir != "" {
			cdPart = fmt.Sprintf("cd %s && ", opts.WorkDir)
		}
		execCmd = []string{"sh", "-c", cdPart + prefix + strings.Join(cmd, " ")}
	}

	req := s.cli.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(id).
		Namespace(s.ns).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: "main",
			Command:   execCmd,
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(s.cfg, "POST", req.URL())
	if err != nil {
		return nil, fmt.Errorf("create executor: %w", err)
	}

	stdinR, stdinW := io.Pipe()
	stdoutR, stdoutW := io.Pipe()
	stderrR, stderrW := io.Pipe()

	execCtx, cancel := context.WithCancel(ctx)
	done := make(chan error, 1)
	var exitCode int

	go func() {
		err := exec.StreamWithContext(execCtx, remotecommand.StreamOptions{
			Stdin:  stdinR,
			Stdout: stdoutW,
			Stderr: stderrW,
		})
		if err != nil {
			if exitErr, ok := err.(interface{ ExitStatus() int }); ok {
				exitCode = exitErr.ExitStatus()
				err = nil
			}
		}
		stdoutW.Close()
		stderrW.Close()
		done <- err
	}()

	return &StreamHandle{
		Stdin:  stdinW,
		Stdout: stdoutR,
		Stderr: stderrR,
		Wait: func() (int, error) {
			err := <-done
			return exitCode, err
		},
		Cancel: func() {
			cancel()
			stdinR.Close()
		},
	}, nil
}

func (s *k8sSandbox) Inspect(ctx context.Context, id string) (*ContainerInfo, error) {
	pod, err := s.cli.CoreV1().Pods(s.ns).Get(ctx, id, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return &ContainerInfo{
		ID:     string(pod.UID),
		Name:   pod.Name,
		Image:  pod.Spec.Containers[0].Image,
		Status: string(pod.Status.Phase),
		IP:     pod.Status.PodIP,
		Labels: pod.Labels,
	}, nil
}

func (s *k8sSandbox) List(ctx context.Context, opts ListOptions) ([]ContainerInfo, error) {
	merged := make(map[string]string)
	for k, v := range s.labels {
		merged[k] = v
	}
	for k, v := range opts.Labels {
		merged[k] = v
	}
	var parts []string
	for k, v := range merged {
		parts = append(parts, k+"="+v)
	}
	labelSelector := strings.Join(parts, ",")

	pods, err := s.cli.CoreV1().Pods(s.ns).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}

	result := make([]ContainerInfo, 0, len(pods.Items))
	for _, pod := range pods.Items {
		ci := ContainerInfo{
			ID:     string(pod.UID),
			Name:   pod.Name,
			Status: string(pod.Status.Phase),
			IP:     pod.Status.PodIP,
			Labels: pod.Labels,
		}
		if len(pod.Spec.Containers) > 0 {
			ci.Image = pod.Spec.Containers[0].Image
		}
		result = append(result, ci)
	}
	return result, nil
}

func (s *k8sSandbox) Close() error {
	return nil // REST client doesn't need explicit close
}

// parseUID extracts a numeric UID from a user string like "1000" or "1000:1000".
func parseUID(user string) (int64, error) {
	parts := strings.SplitN(user, ":", 2)
	var uid int64
	_, err := fmt.Sscanf(parts[0], "%d", &uid)
	return uid, err
}

func buildResources(memory int64, cpus float64) corev1.ResourceRequirements {
	limits := corev1.ResourceList{}
	if memory > 0 {
		limits[corev1.ResourceMemory] = *apiresource.NewQuantity(memory, apiresource.BinarySI)
	}
	if cpus > 0 {
		limits[corev1.ResourceCPU] = *apiresource.NewMilliQuantity(int64(cpus*1000), apiresource.DecimalSI)
	}
	return corev1.ResourceRequirements{Limits: limits}
}
