package provision

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

func EnsureRedis(ctx context.Context, clientset *kubernetes.Clientset, namespace, appName string) error {
	redisName := fmt.Sprintf("redis-%s", appName)
	pvcName := fmt.Sprintf("redis-data-%s", appName)
	secretName := fmt.Sprintf("%s-secrets", appName)
	redisURL := fmt.Sprintf("redis://%s:6379", redisName)

	labels := map[string]string{
		"app":        "redis",
		"managed-by": "deployit",
		"project":    appName,
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("500Mi"),
				},
			},
		},
	}

	_, err := clientset.CoreV1().PersistentVolumeClaims(namespace).Create(ctx, pvc, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create pvc: %w", err)
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: redisName, Namespace: namespace, Labels: labels},
		Spec: corev1.ServiceSpec{
			Selector: labels,
			Ports: []corev1.ServicePort{{
				Port:       6379,
				TargetPort: intstr.FromInt32(6379),
			}},
		},
	}

	_, err = clientset.CoreV1().Services(namespace).Create(ctx, svc, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	deploy := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: redisName, Namespace: namespace, Labels: labels},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "redis-storage",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: pvcName,
								},
							},
						},
					},
					Containers: []corev1.Container{{
						Name:    "redis",
						Image:   "docker.io/library/redis:alpine",
						Command: []string{"sh", "-c"},
						Args: []string{`
						  if [ -f /data/redis.conf ]; then
							echo "Starting with restore override..."
							cp /data/redis.conf /tmp/redis.conf
							rm /data/redis.conf 
							exec redis-server /tmp/redis.conf
						  else
							exec redis-server --appendonly yes
						  fi
					   `},
						Ports: []corev1.ContainerPort{{ContainerPort: 6379}},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "redis-storage",
								MountPath: "/data",
							},
						},
					}},
				},
			},
		},
	}

	_, err = clientset.AppsV1().Deployments(namespace).Create(ctx, deploy, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	secret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		newSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: secretName, Namespace: namespace},
			Data: map[string][]byte{
				"REDIS_URL": []byte(redisURL),
			},
		}
		_, err = clientset.CoreV1().Secrets(namespace).Create(ctx, newSecret, metav1.CreateOptions{})
	} else if err == nil {
		if secret.Data == nil {
			secret.Data = make(map[string][]byte)
		}
		secret.Data["REDIS_URL"] = []byte(redisURL)
		_, err = clientset.CoreV1().Secrets(namespace).Update(ctx, secret, metav1.UpdateOptions{})
	}

	return err
}
