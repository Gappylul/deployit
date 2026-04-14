package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	corev1 "k8s.io/api/core/v1"

	"github.com/gappylul/deployit/internal/build"
	"github.com/gappylul/deployit/internal/cloudflare"
	"github.com/gappylul/deployit/internal/deploy"
	"github.com/gappylul/deployit/internal/detect"
	"github.com/gappylul/deployit/internal/dockerfile"
	"github.com/spf13/cobra"
)

var (
	host     string
	replicas int32
	registry string
	envVars  []string
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
		tag := gitShortSHA()
		fullImage := fmt.Sprintf("%s:%s", imageName, tag)

		if err := build.BuildAndPush(ctx, build.BuildOptions{
			ContextPath: path,
			ImageName:   imageName,
			Tag:         tag,
		}); err != nil {
			return fmt.Errorf("build: %w", err)
		}

		fmt.Printf("-> pushed %s\n", fullImage)

		var parsedEnv []corev1.EnvVar
		for _, e := range envVars {
			parts := strings.SplitN(e, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid env var: %s (expected KEY=VALUE)", e)
			}
			parsedEnv = append(parsedEnv, corev1.EnvVar{
				Name:  parts[0],
				Value: parts[1],
			})
		}

		fmt.Println("-> deploying to cluster")
		if err := deploy.Deploy(ctx, name, fullImage, host, replicas, parsedEnv); err != nil {
			return fmt.Errorf("deploy: %w", err)
		}

		fmt.Printf("\n✓ deployed to https://%s\n", host)

		cf, err := cloudflare.NewClient()
		if err != nil {
			fmt.Printf("⚠ skipping Cloudflare: %s\n", err)
		} else {
			if err := cf.AddHostname(host); err != nil {
				fmt.Printf("⚠ Cloudflare error: %s\n", err)
			}
		}

		return nil
	},
}

func init() {
	defaultRegistry := os.Getenv("DEPLOYIT_REGISTRY")

	deployCmd.Flags().StringVar(&host, "host", "", "hostname to deploy to (required)")
	deployCmd.Flags().Int32Var(&replicas, "replicas", 1, "number of replicas")
	deployCmd.Flags().StringVar(&registry, "registry", defaultRegistry, "image registry e.g. ghcr.io/username (or set DEPLOYIT_REGISTRY)")
	deployCmd.Flags().StringArrayVar(&envVars, "env", []string{}, "environment variables KEY=VALUE")
	deployCmd.MarkFlagRequired("host")
	if defaultRegistry == "" {
		deployCmd.MarkFlagRequired("registry")
	}
	rootCmd.AddCommand(deployCmd)
}

func gitShortSHA() string {
	out, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return "latest"
	}
	return strings.TrimSpace(string(out))
}
