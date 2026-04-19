package kube

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func PrintAppStatus(clientset *kubernetes.Clientset, appName string) error {
	fmt.Printf("\n--- [ STATUS: %s ] ---\n", appName)

	pods, err := clientset.CoreV1().Pods("default").List(context.TODO(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", appName),
	})
	if err != nil {
		return fmt.Errorf("failed to get pods: %w", err)
	}

	fmt.Println("PODS:")
	if len(pods.Items) == 0 {
		fmt.Println("  No pods found for this application.")
	} else {
		for _, p := range pods.Items {
			status := string(p.Status.Phase)
			if len(p.Status.ContainerStatuses) > 0 && p.Status.ContainerStatuses[0].State.Waiting != nil {
				status = p.Status.ContainerStatuses[0].State.Waiting.Reason
			}
			fmt.Printf("  %-40s [%s]\n", p.Name, status)
		}
	}

	fieldSelector := fmt.Sprintf("involvedObject.name=%s", appName)
	events, err := clientset.CoreV1().Events("default").List(context.TODO(), metav1.ListOptions{
		FieldSelector: fieldSelector,
	})
	if err != nil {
		return fmt.Errorf("failed to get events: %w", err)
	}

	fmt.Println("\nRECENT EVENTS:")
	if len(events.Items) == 0 {
		fmt.Println("  No events found.")
	}

	for i, e := range events.Items {
		if i > 5 {
			break
		}
		timestamp := e.LastTimestamp.Time.Format("15:04.05")
		fmt.Printf("  [%s] %-15s %s\n", timestamp, e.Reason, e.Message)
	}
	fmt.Println("")
	return nil
}
