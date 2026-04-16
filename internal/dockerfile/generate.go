package dockerfile

import (
	"github.com/gappylul/deployit/internal/detect"
)

func Generate(framework detect.Framework) string {
	switch framework {
	case detect.Go:
		return goDockerfile
	case detect.Vite:
		return viteDockerfile
	case detect.Bun:
		return bunDockerfile
	case detect.NodeTs:
		return nodeTsDockerfile
	case detect.NodeJs:
		return nodeJsDockerfile
	case detect.Rust:
		return rustDockerfile
	case detect.Python:
		return pythonDockerfile
	default:
		return ""
	}
}

var goDockerfile = `FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN GOARCH=arm64 GOOS=linux go build -o server .

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
`

var nodeJsDockerfile = `FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json .
RUN npm install --omit=dev

FROM node:20-alpine
WORKDIR /app
COPY --from=builder /app/node_modules ./node_modules
COPY . .
EXPOSE 8080
CMD ["node", "src/index.js"]
`

var nodeTsDockerfile = `FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build

FROM node:20-alpine
WORKDIR /app
COPY --from=builder /app/package*.json ./
RUN npm install --omit=dev
COPY --from=builder /app/dist ./dist 
EXPOSE 8080
CMD ["node", "dist/index.js"]
`

var bunDockerfile = `FROM oven/bun:alpine
WORKDIR /app
COPY package.json bun.lockb* ./
RUN bun install --frozen-lockfile
COPY . .
EXPOSE 8080
CMD ["bun", "run", "src/index.ts"]
`

var viteDockerfile = `FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build

FROM node:20-alpine
RUN npm install -g sirv-cli
WORKDIR /app
COPY --from=builder /app/dist .
EXPOSE 8080
CMD ["sirv", ".", "--single", "--port", "8080", "--host", "0.0.0.0"]
`

var nodeDockerIgnoreContent = `node_modules
.git
npm-debug.log
Dockerfile
.dockerignore
`

var rustDockerfile = `FROM rust:alpine AS builder
WORKDIR /app
COPY . .
RUN CARGO_TARGET_AARCH64_UNKNOWN_LINUX_MUSL_LINKER=aarch64-linux-gnu-gcc \
    cargo build --release --target aarch64-unknown-linux-musl

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/target/aarch64-unknown-linux-musl/release/app .
EXPOSE 8080
CMD ["./app"]
`

var pythonDockerfile = `FROM python:3.12-slim
WORKDIR /app
COPY requirements.txt .
RUN pip install -r requirements.txt
COPY . .
EXPOSE 8080
CMD ["python", "main.py"]
`

func GenerateIgnore(framework detect.Framework) string {
	switch framework {
	case detect.Vite:
	case detect.NodeJs:
	case detect.NodeTs:
	case detect.Bun:
		return nodeDockerIgnoreContent
	}
	return ""
}
