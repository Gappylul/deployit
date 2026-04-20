package provision

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

func FindPodByPrefix(ctx context.Context, clientset *kubernetes.Clientset, prefix string) (string, error) {
	pods, err := clientset.CoreV1().Pods("default").List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", err
	}

	for _, pod := range pods.Items {
		if strings.HasPrefix(pod.Name, prefix) {
			return pod.Name, nil
		}
	}

	return "", fmt.Errorf("no pod found starting with prefix: %s", prefix)
}

func BackupPostgres(ctx context.Context, config *rest.Config, clientset *kubernetes.Clientset, appName string, out io.Writer) error {
	prefix := fmt.Sprintf("postgres-%s", appName)
	podName, err := FindPodByPrefix(ctx, clientset, prefix)
	if err != nil {
		return fmt.Errorf("finding postgres pod: %w", err)
	}

	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace("default").
		SubResource("exec")

	option := &corev1.PodExecOptions{
		Command: []string{"pg_dump", "-U", "postgres", "-d", appName, "--clean", "--if-exists"},
		Stdout:  true,
		Stderr:  true,
		TTY:     false,
	}
	req.VersionedParams(option, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return err
	}

	return exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: out,
		Stderr: os.Stderr,
	})
}

func RestorePostgres(ctx context.Context, config *rest.Config, clientset *kubernetes.Clientset, appName string, in io.Reader) error {
	prefix := fmt.Sprintf("postgres-%s", appName)
	podName, err := FindPodByPrefix(ctx, clientset, prefix)
	if err != nil {
		return fmt.Errorf("finding postgres pod: %w", err)
	}

	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace("default").
		SubResource("exec")

	option := &corev1.PodExecOptions{
		Command: []string{"psql", "-U", "postgres", "-d", appName},
		Stdin:   true,
		Stdout:  true,
		Stderr:  true,
		TTY:     false,
	}
	req.VersionedParams(option, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return err
	}

	return exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  in,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})
}

func BackupRedis(ctx context.Context, config *rest.Config, clientset *kubernetes.Clientset, appName string, out io.Writer) error {
	prefix := fmt.Sprintf("redis-%s", appName)
	podName, err := FindPodByPrefix(ctx, clientset, prefix)
	if err != nil {
		return err
	}

	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").Name(podName).Namespace("default").SubResource("exec")

	option := &corev1.PodExecOptions{
		Command: []string{"sh", "-c", "redis-cli SAVE > /dev/null && cat /data/dump.rdb"},
		Stdout:  true,
		Stderr:  true,
	}
	req.VersionedParams(option, scheme.ParameterCodec)
	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return err
	}

	return exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdout: out,
		Stderr: os.Stderr,
	})
}

func RestoreRedis(ctx context.Context, config *rest.Config, clientset *kubernetes.Clientset, appName string, in io.Reader) error {
	pvcName := fmt.Sprintf("redis-data-%s", appName)
	helperPodName := fmt.Sprintf("redis-restore-helper-%s", appName)

	helperPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: helperPodName, Namespace: "default"},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name:    "helper",
				Image:   "busybox",
				Command: []string{"sh", "-c", "sleep 3600"},
				VolumeMounts: []corev1.VolumeMount{{
					Name:      "data",
					MountPath: "/data",
				}},
			}},
			Volumes: []corev1.Volume{{
				Name: "data",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: pvcName,
					},
				},
			}},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}

	fmt.Println("-> Spinning up restore helper pod...")
	_, err := clientset.CoreV1().Pods("default").Create(ctx, helperPod, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	defer clientset.CoreV1().Pods("default").Delete(ctx, helperPodName, metav1.DeleteOptions{})

	time.Sleep(10 * time.Second)

	req := clientset.CoreV1().RESTClient().Post().
		Resource("pods").Name(helperPodName).Namespace("default").SubResource("exec")

	option := &corev1.PodExecOptions{
		Command: []string{"sh", "-c", `
        rm -rf /data/*
        cat > /data/dump.rdb
        echo "appendonly no" > /data/redis.conf
        echo "save \"\"" >> /data/redis.conf
        chown -R 999:999 /data
        chmod 666 /data/dump.rdb
        sync
    `},
		Stdin:  true,
		Stdout: true,
		Stderr: true,
	}
	req.VersionedParams(option, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return err
	}

	fmt.Println("-> Uploading RDB file to volume...")
	return exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  in,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	})
}
