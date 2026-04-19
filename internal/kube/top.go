package kube

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/metrics/pkg/client/clientset/versioned"
)

func PrintTop(config *rest.Config, appName string) error {
	metricsClient, err := versioned.NewForConfig(config)
	if err != nil {
		return err
	}

	podMetrics, err := metricsClient.MetricsV1beta1().PodMetricses("default").List(context.TODO(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", appName),
	})
	if err != nil {
		return fmt.Errorf("failed to get metrics: %w", err)
	}

	if len(podMetrics.Items) == 0 {
		fmt.Printf("No metrics available for app '%s'\n", appName)
		return nil
	}

	fmt.Printf("\n--- [ RESOURCE USAGE: %s ] ---\n", appName)
	fmt.Printf("%-45s %-12s %-12s\n", "POD NAME", "CPU(cores)", "MEMORY(bytes)")

	for _, pm := range podMetrics.Items {
		var cpuTotal int64
		var memTotal int64

		for _, container := range pm.Containers {
			cpuTotal += container.Usage.Cpu().MilliValue()
			memTotal += container.Usage.Memory().Value() / (1024 * 1024)
		}

		fmt.Printf("%-45s %dm          %dMi\n", pm.Name, cpuTotal, memTotal)
	}
	return nil
}
