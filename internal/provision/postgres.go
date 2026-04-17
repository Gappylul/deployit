package provision

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

func EnsurePostgres(ctx context.Context, clientset *kubernetes.Clientset, ns, appName string) (string, error) {
	name := fmt.Sprintf("postgres-%s", appName)
	secretName := fmt.Sprintf("postgres-auth-%s", appName)
	pvcName := fmt.Sprintf("postgres-data-%s", appName)

	password, err := getOrCreatePostgresSecret(ctx, clientset, ns, appName, secretName)
	if err != nil {
		return "", err
	}

	ensurePostgresPVC(ctx, clientset, ns, appName, pvcName)

	ensurePostgresDeployment(ctx, clientset, ns, appName, name, secretName, pvcName)

	ensurePostgresService(ctx, clientset, ns, appName, name)

	return fmt.Sprintf("postgres://postgres:%s@%s:5432/%s?sslmode=disable", password, name, appName), nil
}

func getOrCreatePostgresSecret(ctx context.Context, clientset *kubernetes.Clientset, ns, appName, secretName string) (string, error) {
	secret, err := clientset.CoreV1().Secrets(ns).Get(ctx, secretName, metav1.GetOptions{})
	if err == nil {
		return string(secret.Data["password"]), nil
	}

	b := make([]byte, 12)
	_, err = rand.Read(b)
	if err != nil {
		return "", err
	}
	pw := hex.EncodeToString(b)

	_, err = clientset.CoreV1().Secrets(ns).Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:   secretName,
			Labels: map[string]string{"project": appName, "managed-by": "deployit"},
		},
		StringData: map[string]string{
			"password": pw,
			"user":     "postgres",
			"database": appName,
		},
	}, metav1.CreateOptions{})

	return pw, err
}

func ensurePostgresPVC(ctx context.Context, clientset *kubernetes.Clientset, ns, appName, pvcName string) {
	_, err := clientset.CoreV1().PersistentVolumeClaims(ns).Get(ctx, pvcName, metav1.GetOptions{})
	if err != nil {
		_, _ = clientset.CoreV1().PersistentVolumeClaims(ns).Create(ctx, &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:   pvcName,
				Labels: map[string]string{"project": appName},
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.VolumeResourceRequirements{
					Requests: corev1.ResourceList{corev1.ResourceStorage: resource.MustParse("1Gi")},
				},
			},
		}, metav1.CreateOptions{})
	}
}

func ensurePostgresDeployment(ctx context.Context, clientset *kubernetes.Clientset, ns, appName, name, secretName, pvcName string) {
	labels := map[string]string{"project": appName, "app": "postgres"}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: name, Labels: labels},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "postgres",
							Image: "postgres:16-alpine",
							Env: []corev1.EnvVar{
								{
									Name: "POSTGRES_PASSWORD",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
											Key:                  "password",
										},
									},
								},
								{Name: "POSTGRES_USER", Value: "postgres"},
								{Name: "POSTGRES_DB", Value: appName},
							},
							Ports: []corev1.ContainerPort{{ContainerPort: 5432}},
							VolumeMounts: []corev1.VolumeMount{
								{Name: "data", MountPath: "/var/lib/postgresql/data"},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse("256Mi"),
									corev1.ResourceCPU:    resource.MustParse("100m"),
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "data",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{ClaimName: pvcName},
							},
						},
					},
				},
			},
		},
	}

	_, err := clientset.AppsV1().Deployments(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		_, _ = clientset.AppsV1().Deployments(ns).Create(ctx, deployment, metav1.CreateOptions{})
	} else {
		_, _ = clientset.AppsV1().Deployments(ns).Update(ctx, deployment, metav1.UpdateOptions{})
	}
}

func ensurePostgresService(ctx context.Context, clientset *kubernetes.Clientset, ns, appName, name string) {
	_, err := clientset.CoreV1().Services(ns).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		_, _ = clientset.CoreV1().Services(ns).Create(ctx, &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:   name,
				Labels: map[string]string{"project": appName},
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{"project": appName, "app": "postgres"},
				Ports: []corev1.ServicePort{
					{Port: 5432, TargetPort: intstr.FromInt32(5432)},
				},
			},
		}, metav1.CreateOptions{})
	}
}
