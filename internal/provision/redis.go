package provision

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

func EnsureRedis(ctx context.Context, clientset *kubernetes.Clientset, namespace, appName string) error {
	redisName := fmt.Sprintf("redis-%s", appName)
	secretName := fmt.Sprintf("%s-secrets", appName)
	redisURL := fmt.Sprintf("redis://%s:6379", redisName)

	labels := map[string]string{
		"app":        "redis",
		"managed-by": "deployit",
		"project":    appName,
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
	_, err := clientset.CoreV1().Services(namespace).Create(ctx, svc, metav1.CreateOptions{})
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
					Containers: []corev1.Container{{
						Name:  "redis",
						Image: "redis:alpine",
						Ports: []corev1.ContainerPort{{ContainerPort: 6379}},
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
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: namespace,
				Labels:    map[string]string{"app": appName},
			},
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
