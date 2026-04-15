package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs <name>",
	Short: "Stream logs from your app",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		c := exec.Command("kubectl",
			"logs",
			"-f",
			"-l", fmt.Sprintf("app=%s", name),
			"--all-containers",
			"--tail", tailLines,
		)

		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

var tailLines string

func init() {
	logsCmd.Flags().StringVar(&tailLines, "tail", "100", "Number of lines to show from the end of the logs")
	rootCmd.AddCommand(logsCmd)
}
