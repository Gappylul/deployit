package detect

import (
	"os"
	"path/filepath"
)

type Framework string

const (
	Go      Framework = "go"
	Node    Framework = "node"
	Rust    Framework = "rust"
	Python  Framework = "python"
	Custom  Framework = "custom"
	Unknown Framework = "unknown"
)

func Detect(path string) Framework {
	if exists(path, "Dockerfile") {
		return Custom
	}
	if exists(path, "go.mod") {
		return Go
	}
	if exists(path, "package.json") {
		return Node
	}
	if exists(path, "Cargo.toml") {
		return Rust
	}
	if exists(path, "requirements.txt") || exists(path, "pyproject.toml") {
		return Python
	}
	return Unknown
}

func exists(base, file string) bool {
	_, err := os.Stat(filepath.Join(base, file))
	return err == nil
}
