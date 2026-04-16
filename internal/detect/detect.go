package detect

import (
	"os"
	"path/filepath"
)

type Framework string

const (
	Go      Framework = "go"
	NodeJs  Framework = "nodeJs"
	NodeTs  Framework = "nodeTs"
	Bun     Framework = "bun"
	Vite    Framework = "vite"
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
	if exists(path, "vite.config.js") {
		return Vite
	}
	if exists(path, "bun.lock") {
		return Bun
	}
	if exists(path, "tsconfig.json") {
		return NodeTs
	}
	if exists(path, "package.json") {
		return NodeJs
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
