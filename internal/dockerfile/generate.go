package dockerfile

import (
	"github.com/gappylul/deployit/internal/detect"
)

func Generate(framework detect.Framework) string {
	switch framework {
	case detect.Go:
		return goDockerfile
	case detect.Node:
		return nodeDockerfile
	case detect.Rust:
		return rustDockerfile
	case detect.Python:
		return pythonDockerfile
	default:
		return ""
	}
}

var goDockerfile = `FROM --platform=linux/amd64 golang:1.22-alpine AS builder
WORKDIR /app
COPY go.mod go.sum* .
RUN go mod download
COPY . .
RUN GOARCH=arm64 GOOS=linux go build -o server .

FROM --platform=linux/arm64 alpine:latest
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
`

var nodeDockerfile = `FROM --platform=linux/amd64 node:20-alpine AS builder
WORKDIR /app
COPY package*.json .
RUN npm ci --production

FROM --platform=linux/arm64 node:20-alpine
WORKDIR /app
COPY --from=builder /app/node_modules ./node_modules
COPY . .
EXPOSE 8080
CMD ["node", "index.js"]
`

var rustDockerfile = `FROM --platform=linux/amd64 rust:alpine AS builder
WORKDIR /app
COPY . .
RUN CARGO_TARGET_AARCH64_UNKNOWN_LINUX_MUSL_LINKER=aarch64-linux-gnu-gcc \
    cargo build --release --target aarch64-unknown-linux-musl

FROM --platform=linux/arm64 alpine:latest
WORKDIR /app
COPY --from=builder /app/target/aarch64-unknown-linux-musl/release/app .
EXPOSE 8080
CMD ["./app"]
`

var pythonDockerfile = `FROM --platform=linux/arm64 python:3.12-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install -r requirements.txt
COPY . .
EXPOSE 8080
CMD ["python", "main.py"]
`
