package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gappylul/deployit/internal/build"
	"github.com/gappylul/deployit/internal/deploy"
	"github.com/gappylul/deployit/internal/detect"
	"github.com/gappylul/deployit/internal/dockerfile"
	"github.com/spf13/cobra"
)

var (
	host     string
	replicas int32
	registry string
)

var deployCmd = &cobra.Command{
	Use:   "deploy <path>",
	Short: "Deploy a project to your homelab",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		path := args[0]
		name := filepath.Base(path)

		framework := detect.Detect(path)
		if framework == detect.Unknown {
			return fmt.Errorf("could not detect framework in %s", path)
		}
		fmt.Printf("-> detected: %s\n", framework)

		dockerfilePath := filepath.Join(path, "Dockerfile")
		generated := false
		if framework != detect.Custom {
			content := dockerfile.Generate(framework)
			if err := os.WriteFile(dockerfilePath, []byte(content), 0644); err != nil {
				return fmt.Errorf("write dockerfile: %w", err)
			}
			generated = true
			fmt.Printf("-> generated Dockerfile for %s\n", framework)
		} else {
			fmt.Printf("-> using existing Dockerfile\n")
		}
		if generated {
			defer os.Remove(dockerfilePath)
		}

		imageName := fmt.Sprintf("%s/%s", registry, name)
		tag := "latest"

		if err := build.BuildAndPush(ctx, build.BuildOptions{
			ContextPath: path,
			ImageName:   imageName,
			Tag:         tag,
		}); err != nil {
			return fmt.Errorf("build: %w", err)
		}

		fmt.Printf("-> pushed %s:%s\n", imageName, tag)

		imageName = fmt.Sprintf("%s/%s", registry, name)
		fmt.Println("-> deploying to cluster")
		if err := deploy.Deploy(ctx, name, imageName+":latest", host, replicas); err != nil {
			return fmt.Errorf("deploy: %w", err)
		}

		fmt.Printf("\n✓ deployed to http://%s\n", host)
		return nil
	},
}

func init() {
	defaultRegistry := os.Getenv("DEPLOYIT_REGISTRY")

	deployCmd.Flags().StringVar(&host, "host", "", "hostname to deploy to (required)")
	deployCmd.Flags().Int32Var(&replicas, "replicas", 1, "number of replicas")
	deployCmd.Flags().StringVar(&registry, "registry", defaultRegistry, "image registry e.g. ghcr.io/username (or set DEPLOYIT_REGISTRY)")
	deployCmd.MarkFlagRequired("host")
	if defaultRegistry == "" {
		deployCmd.MarkFlagRequired("registry")
	}
	rootCmd.AddCommand(deployCmd)
}
