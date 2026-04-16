package cmd

import (
	"context"
	"fmt"

	"github.com/gappylul/deployit/internal/deploy"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var servicesCmd = &cobra.Command{
	Use:   "services <app-name>",
	Short: "Show attached services and health for an app",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		appName := args[0]
		clientset, err := deploy.GetClientset()
		if err != nil {
			return err
		}

		ctx := context.TODO()
		labelSelector := fmt.Sprintf("project=%s", appName)

		pods, _ := clientset.CoreV1().Pods("default").List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
		pvcs, _ := clientset.CoreV1().PersistentVolumeClaims("default").List(ctx, metav1.ListOptions{LabelSelector: labelSelector})

		maxLen := 15
		for _, p := range pods.Items {
			l := len(p.Name) + 4
			if l > maxLen {
				maxLen = l
			}
		}
		for _, p := range pvcs.Items {
			l := len(p.Name) + 4
			if l > maxLen {
				maxLen = l
			}
		}

		colWidth := maxLen + 2

		fmt.Printf("STATUS REPORT: %s\n", appName)

		fmt.Printf("\n%-*s STATE     RESTARTS/CAP\n", colWidth, "COMPUTE (Pods)")
		fmt.Printf("%-*s -----     ------------\n", colWidth, "--------------")

		if len(pods.Items) == 0 {
			fmt.Println("(none)")
		}

		for _, pod := range pods.Items {
			indicator, state := getPodTextStatus(pod)
			restarts := 0
			if len(pod.Status.ContainerStatuses) > 0 {
				restarts = int(pod.Status.ContainerStatuses[0].RestartCount)
			}

			name := fmt.Sprintf("%s %s", indicator, pod.Name)
			fmt.Printf("%-*s %-10s %d\n", colWidth, name, state, restarts)
		}

		if len(pvcs.Items) > 0 {
			fmt.Printf("\n%-*s\n", colWidth, "STORAGE (Volumes)")
			for _, pvc := range pvcs.Items {
				indicator := "[*]"
				if pvc.Status.Phase == corev1.ClaimBound {
					indicator = "[+]"
				}
				capacity := pvc.Spec.Resources.Requests.Storage().String()
				name := fmt.Sprintf("%s %s", indicator, pvc.Name)

				fmt.Printf("%-*s %-10s %s\n", colWidth, name, pvc.Status.Phase, capacity)
			}
		}

		fmt.Println("")
		return nil
	},
}

func getPodTextStatus(pod corev1.Pod) (string, string) {
	if pod.DeletionTimestamp != nil {
		return "[*]", "Terminating"
	}
	for _, container := range pod.Status.ContainerStatuses {
		if container.State.Waiting != nil {
			return "[!]", container.State.Waiting.Reason
		}
		if container.State.Terminated != nil {
			if container.State.Terminated.ExitCode != 0 {
				return "[X]", "Error"
			}
			return "[-]", "Completed"
		}
	}
	switch pod.Status.Phase {
	case corev1.PodPending:
		return "[*]", "Pending"
	case corev1.PodRunning:
		return "[+]", "Running"
	case corev1.PodFailed:
		return "[X]", "Failed"
	default:
		return "[?]", string(pod.Status.Phase)
	}
}

func init() {
	rootCmd.AddCommand(servicesCmd)
}
