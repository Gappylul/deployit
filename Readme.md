# deployit

Deploy any project to your homelab Kubernetes cluster with a single command.

```bash
deployit deploy ./my-project --host myapp.yourdomain.com
```

Detects your framework, builds an arm64 Docker image, pushes it to your container registry, and deploys it to your cluster automatically. No YAML, no Dockerfile required.

## How it works

```
deployit deploy ./my-project
        ↓
detects framework (Go, Node, Rust, Python, or custom Dockerfile)
        ↓
generates Dockerfile if needed
        ↓
builds arm64 image → pushes to your registry
        ↓
creates WebApp custom resource on the cluster
        ↓
webapp-operator reconciles → Deployment + Service + Ingress
        ↓
✓ deployed to https://myapp.yourdomain.com
```

## Requirements

- [Docker Desktop](https://www.docker.com/products/docker-desktop/) running locally
- `kubectl` configured and pointing at your cluster (`~/.kube/config`)
- [webapp-operator](https://github.com/gappylul/webapp-operator) installed on the cluster
- Logged into your container registry:
  ```bash
  echo $PAT | docker login ghcr.io -u YOUR_USERNAME --password-stdin
  ```

## Installation

```bash
git clone https://github.com/gappylul/deployit
cd deployit
go build -o deployit .
mv deployit /usr/local/bin/deployit
```

## Setup

Set your registry once so you don't have to pass it every time:

```bash
export DEPLOYIT_REGISTRY=ghcr.io/your-username
# add to ~/.zshrc or ~/.bashrc to make it permanent
```

## Usage

### Deploy a project

```bash
deployit deploy <path> --host <hostname> [--replicas <n>] [--registry <registry>]
```

```bash
# uses DEPLOYIT_REGISTRY env var
deployit deploy ./my-api --host api.yourdomain.com

# explicit registry
deployit deploy ./my-api --host api.yourdomain.com --registry ghcr.io/your-username

# deploy with 3 replicas
deployit deploy ./my-api --host api.yourdomain.com --replicas 3
```

### List deployed apps

```bash
deployit list
```

```
NAME                 REPLICAS   HOST
my-api               2          api.yourdomain.com
my-frontend          1          app.yourdomain.com
```

### Delete an app

```bash
deployit delete <name>
```

```bash
deployit delete my-api
✓ deleted my-api
```

## Framework detection

deployit detects frameworks by looking for these files in order:

| File | Framework |
|---|---|
| `Dockerfile` | Custom (uses existing Dockerfile) |
| `go.mod` | Go |
| `package.json` | Node.js |
| `Cargo.toml` | Rust |
| `requirements.txt` / `pyproject.toml` | Python |

If a `Dockerfile` already exists in the project, it is used as-is. Otherwise deployit generates one automatically targeting `linux/arm64`.

## Flags

| Flag | Default | Description |
|---|---|---|
| `--host` | required | Hostname to deploy to |
| `--registry` | `$DEPLOYIT_REGISTRY` | Container image registry |
| `--replicas` | `1` | Number of pod replicas |

## Architecture

deployit is built on top of [webapp-operator](https://github.com/gappylul/webapp-operator) — a Kubernetes operator that watches `WebApp` custom resources and manages the underlying Deployment, Service, and Ingress automatically.

When you run `deployit deploy`, it creates a `WebApp` resource:

```yaml
apiVersion: platform.gappy.hu/v1
kind: WebApp
metadata:
  name: my-api
  namespace: default
spec:
  image: ghcr.io/your-username/my-api:latest
  replicas: 2
  host: api.yourdomain.com
```

The operator handles the rest. Deleting the `WebApp` cascades — all child resources are cleaned up automatically.

## Self-hosting

deployit is designed for self-hosted Kubernetes clusters. It works with any cluster that has:

- Traefik as the ingress controller (k3s default)
- [webapp-operator](https://github.com/gappylul/webapp-operator) installed
- A container registry your cluster can pull from

The easiest setup is a Raspberry Pi 5 running [k3s](https://k3s.io) with a [Cloudflare Tunnel](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/) for external access.