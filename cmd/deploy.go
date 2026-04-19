package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gappylul/deployit/internal/provision"
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
	withExt  []string
	arch     string
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
		ignorePath := filepath.Join(path, ".dockerignore")
		generated := false
		if framework != detect.Custom {
			content := dockerfile.Generate(framework)
			if err := os.WriteFile(dockerfilePath, []byte(content), 0644); err != nil {
				return fmt.Errorf("write dockerfile: %w", err)
			}

			ignoreContent := dockerfile.GenerateIgnore(framework)
			if ignoreContent != "" {
				if err := os.WriteFile(ignorePath, []byte(ignoreContent), 0644); err != nil {
					return fmt.Errorf("write dockerignore: %w", err)
				}
			}

			generated = true
			fmt.Printf("-> generated Dockerfile for %s\n", framework)
		} else {
			fmt.Printf("-> using existing Dockerfile\n")
		}
		if generated {
			defer os.Remove(dockerfilePath)
			defer os.Remove(ignorePath)
		}

		imageName := fmt.Sprintf("%s/%s", registry, name)
		tag := gitShortSHA()
		fullImage := fmt.Sprintf("%s:%s", imageName, tag)

		dockerPlatform := "linux/arm64"
		if arch == "amd64" {
			dockerPlatform = "linux/amd64"
		}

		if err := build.BuildAndPush(ctx, build.BuildOptions{
			ContextPath: path,
			ImageName:   imageName,
			Tag:         tag,
			Platform:    dockerPlatform,
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

		k8sClient, err := deploy.GetClientset()
		if err != nil {
			return fmt.Errorf("failed to get k8s client: %w", err)
		}

		for _, ext := range withExt {
			switch ext {
			case "redis":
				fmt.Printf("-> provisioning attached redis for %s (updating secrets)...\n", name)
				if err := provision.EnsureRedis(ctx, k8sClient, "default", name); err != nil {
					return fmt.Errorf("redis provision failed: %w", err)
				}
			case "postgres":
				fmt.Printf("-> provisioning attached postgres for %s (updating secrets)...\n", name)
				dbURL, err := provision.EnsurePostgres(ctx, k8sClient, "default", name)
				if err != nil {
					return fmt.Errorf("postgres provision failed: %w", err)
				}
				parsedEnv = append(parsedEnv, corev1.EnvVar{
					Name:  "DATABASE_URL",
					Value: dbURL,
				})
				fmt.Println("-> injected DATABASE_URL")
			}
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
	deployCmd.Flags().StringSliceVar(&withExt, "with", []string{}, "add extensions (postgres, redis)")
	deployCmd.Flags().StringVar(&arch, "arch", "arm64", "target architecture (arm64 or amd64)")
	deployCmd.MarkFlagRequired("host")
	if defaultRegistry == "" {
		deployCmd.MarkFlagRequired("registry")
	}
	rootCmd.AddCommand(deployCmd)
}

func gitShortSHA() string {
	shaRaw, err := exec.Command("git", "rev-parse", "--short", "HEAD").Output()
	if err != nil {
		return "latest"
	}
	sha := strings.TrimSpace(string(shaRaw))

	status, _ := exec.Command("git", "status", "--porcelain").Output()

	if len(status) > 0 {
		return fmt.Sprintf("%s-dirty-%d", sha, time.Now().Unix())
	}
	return sha
}
