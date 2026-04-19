package build

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

type BuildOptions struct {
	ContextPath string
	ImageName   string
	Tag         string
	Platform    string
}

func BuildAndPush(ctx context.Context, opts BuildOptions) error {
	fullTag := fmt.Sprintf("%s:%s", opts.ImageName, opts.Tag)

	platform := opts.Platform
	if platform == "" {
		platform = "linux/arm64"
	}

	fmt.Printf("-> building %s\n", fullTag)
	build := exec.CommandContext(ctx, "docker", "build",
		"--platform", platform,
		"-t", fullTag,
		opts.ContextPath,
	)
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		return fmt.Errorf("docker build: %w", err)
	}

	fmt.Printf("-> pushing %s\n", fullTag)
	push := exec.CommandContext(ctx, "docker", "push", fullTag)
	push.Stdout = os.Stdout
	push.Stderr = os.Stderr
	if err := push.Run(); err != nil {
		return fmt.Errorf("docker push: %w", err)
	}

	return nil
}
