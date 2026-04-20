package deploy

import (
	"context"
	"fmt"

	platformv1 "github.com/gappylul/webapp-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetConfig() (*rest.Config, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules, nil,
	).ClientConfig()
}

func GetClientset() (*kubernetes.Clientset, error) {
	config, err := GetConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func ScaleDeployment(ctx context.Context, clientset *kubernetes.Clientset, name string, replicas int32) error {
	scale, err := clientset.AppsV1().Deployments("default").GetScale(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	scale.Spec.Replicas = replicas
	_, err = clientset.AppsV1().Deployments("default").UpdateScale(ctx, name, scale, metav1.UpdateOptions{})
	return err
}

func Deploy(ctx context.Context, name, image, host string, replicas int32, env []corev1.EnvVar) error {
	config, err := GetConfig()
	if err != nil {
		return fmt.Errorf("load kubeconfig: %w", err)
	}

	scheme := runtime.NewScheme()
	if err := platformv1.AddToScheme(scheme); err != nil {
		return fmt.Errorf("add scheme: %w", err)
	}

	k8s, err := client.New(config, client.Options{Scheme: scheme})
	if err != nil {
		return fmt.Errorf("k8s client: %w", err)
	}

	webapp := &platformv1.WebApp{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: platformv1.WebAppSpec{
			Image:    image,
			Replicas: &replicas,
			Host:     host,
			Env:      env,
		},
	}

	err = k8s.Create(ctx, webapp)
	if err != nil {
		existing := &platformv1.WebApp{}
		if getErr := k8s.Get(ctx, client.ObjectKey{Name: name, Namespace: "default"}, existing); getErr != nil {
			return fmt.Errorf("create webapp: %w", err)
		}
		existing.Spec = webapp.Spec
		if err := k8s.Update(ctx, existing); err != nil {
			return fmt.Errorf("update webapp: %w", err)
		}
		fmt.Printf("-> updated WebApp %s\n", name)
		return nil
	}

	fmt.Printf("-> created WebApp %s\n", name)
	return nil
}
