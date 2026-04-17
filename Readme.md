# deployit

Deploy any project to your homelab Kubernetes cluster with a single command.

```bash
deployit deploy ./my-project --host myapp.yourdomain.com
```

Detects your framework, builds an arm64 Docker image, pushes it to your container registry, deploys it to your cluster, and configures Cloudflare DNS automatically. No YAML, no Dockerfile, no dashboard clicks required.

## How it works

```
deployit deploy ./my-project
        ↓
detects framework (Go, Node, Vite, Rust, Python, or custom Dockerfile)
        ↓
generates Dockerfile if needed
        ↓
builds arm64 image → pushes to your registry
        ↓
creates WebApp custom resource on the cluster
        ↓
webapp-operator reconciles → Deployment + Service + Ingress
        ↓
Cloudflare tunnel route added + DNS CNAME created
        ↓
https://myapp.yourdomain.com is live
```

## Requirements

- Docker Desktop running locally
- kubectl configured and pointing at your cluster (~/.kube/config)
- webapp-operator installed on the cluster
- Logged into your container registry
- A Cloudflare tunnel set up for your domain

## Installation

```bash
git clone https://github.com/gappylul/deployit
cd deployit
go build -o deployit .
sudo mv deployit /usr/local/bin/deployit
```

**or**

```bash
go install github.com/gappylul/deployit@latest
```

## Setup

Store secrets in a dedicated file:

```bash
cat > ~/.secrets << 'SECRETS'
export DEPLOYIT_REGISTRY=ghcr.io/your-username
export CLOUDFLARE_TOKEN=your_token
export CLOUDFLARE_ACCOUNT_ID=your_account_id
export CLOUDFLARE_TUNNEL_ID=your_tunnel_id
export CLOUDFLARE_ZONE_ID=your_zone_id
SECRETS

chmod 600 ~/.secrets
echo 'source ~/.secrets' >> ~/.zshrc
source ~/.secrets
```

Cloudflare API token permissions required:
- Account → Cloudflare Tunnel → Edit
- Account → Account Settings → Read
- Zone → DNS → Edit (for your domain)

If Cloudflare env vars are not set, deployit will skip DNS automation and warn you rather than failing.

## Usage

### Deploy a project

```bash
deployit deploy <path> --host <hostname> [--replicas <n>] [--registry <registry>]
```

```bash
deployit deploy ./my-api --host api.yourdomain.com
deployit deploy ./my-api --host api.yourdomain.com --replicas 3
deployit deploy ./my-api --host api.yourdomain.com --registry ghcr.io/your-username
```

### Deploy with Extensions (Databases)

`deployit` can automatically provision dedicated infrastructure for your app.
> Extension instances are completely isolated per-project and come pre-configured with the necessary connection strings automatically injected into your secrets.

```bash
# Deploy with a dedicated, isolated Redis instance
deployit deploy ./my-app --host app.yourdomain.com --with redis
```

### List deployed apps

```bash
deployit list

NAME                 STATUS          READY      HOST
my-api               ● Running       2/2        api.yourdomain.com
my-frontend          ● Progressing   0/1        app.yourdomain.com
old-app              ● Error         0/1        old.yourdomain.com
```

### See status of services

Example:
```bash
deployit services goshort

STATUS REPORT: goshort

COMPUTE (Pods)                      STATE     RESTARTS/CAP
--------------                      -----     ------------
[+] goshort-7db4b9967-vvt2p         Running    0
[+] redis-goshort-d664fb97b-6f2dl   Running    0

STORAGE (Volumes)                  
[+] redis-data-goshort              Bound      500Mi
```

### Delete an app

```bash
deployit delete <name> --host <hostname>
```

```bash
deployit delete my-api --host api.yourdomain.com
✓ deleted my-api
→ cloudflare: removed api.yourdomain.com from tunnel
```

Always pass `--host` when deleting so the Cloudflare DNS record and tunnel route are cleaned up automatically. If you forget `--host` and the app is already deleted, use the `cleanup` command.

> Automatically deletes the extensions connected to the deployed app.

### Stream logs

```bash
deployit logs <name> [--tail <n>]
```

> Handles high-concurrency log streaming (up to 50 replicas) and automatically prefixes lines with the specific pod name for easier debugging.

### Manage Secrets

Manage environment variables securely without putting them in your Git history or YAML files. `deployit` creates a Kubernetes Secret and the operator automatically injects all keys as environment variables.

```bash
# List secrets for an app
deployit secrets my-api

# Add or update secrets (merges with existing)
deployit secrets my-api DB_PASSWORD=pizza REDIS_URL=redis://10.42.0.5:6379

# Update a secret and trigger a rolling restart
deployit secrets my-api RESTART_KEY=1

# Delete a secret
deployit secrets delete my-api RESTART_KEY
```

> When you update a secret, the webapp-operator detects the change and automatically performs a rolling restart of your pods so the new values are picked up immediately.

### Clean up Cloudflare only

If you deleted an app without --host and the DNS record is orphaned:

```bash
deployit cleanup --host api.yourdomain.com
→ cloudflare: removed api.yourdomain.com from tunnel
```

This only touches Cloudflare — it does not affect the cluster.

## Framework detection

deployit detects frameworks by looking for indicator files in this order:

| File                                   | Framework                         |
|----------------------------------------|-----------------------------------|
| Dockerfile                             | Custom (uses existing Dockerfile) |
| vite.config.js / vite.config.ts        | Vite (React, Vue, etc.)           |
| go.mod                                 | Go                                |
| bun.lock (bun install --lockfile-only) | Bun                               |
| tsconfig.json                          | Node.ts                           |
| package.json                           | Node.js                           |
| Cargo.toml                             | Rust                              |
| requirements.txt / pyproject.toml      | Python                            |

If a Dockerfile already exists it is used as-is. Otherwise deployit generates one automatically targeting linux/arm64. Adding support for a new framework is two files and about 10 lines of Go.

## Commands

| Command    | Description                                                |
|------------|------------------------------------------------------------|
| deploy     | Build, push, deploy, configure DNS                         |
| list       | List all deployed apps real-time status                    |
| logs       | Stream real-time logs from your app                        |
| services   | Show attached services (like redis) for an app             |
| delete     | Delete app and clean up Cloudflare                         |
| secrets    | List, set, or update app secrets                           |
| cleanup    | Remove a hostname from Cloudflare only                     |
| completion | Generate the autocompletion script for the specified shell |

## Flags

| Flag       | Default            | Description                      |
|------------|--------------------|----------------------------------|
| --host     | required           | Hostname to deploy to            |
| --registry | $DEPLOYIT_REGISTRY | Container image registry         |
| --replicas | 1                  | Number of pod replicas           |
| --env      | none               | Environment variables            |
| --with     | none               | Add extensions (postgres, redis) |
| --tail     | 100                | Number of lines to show          |

## Architecture

deployit is built on top of webapp-operator, a Kubernetes operator that watches WebApp custom resources and manages the underlying Deployment, Service, and Ingress automatically.

When you run deployit deploy, it creates:

```yaml
apiVersion: platform.gappy.hu/v1
kind: WebApp
metadata:
  name: my-api
  namespace: default
spec:
  image: ghcr.io/your-username/my-api:git-sha
  replicas: 2
  host: api.yourdomain.com
  env:
    - name: HELLO
      value: "world" 
```

> Uses Git SHA for versioning. If the repository is 'dirty' (uncommitted changes), a timestamp is appended to force a fresh pull on the cluster.

The operator handles the rest. Deleting the WebApp cascades — all child resources are cleaned up automatically.

**Environment** 
> **Secret Injection:** The operator looks for a secret named `<app-name>-secrets`. If found, it adds an `envFrom` source to the Deployment. This means any key you add via `deployit secrets` is instantly available to your application code via standard environment variable lookups (e.g., `os.Getenv("DB_PASSWORD")`).


## Smart Persistence

When you deploy with extensions like `--with postgres/redis`, `deployit` automates the boring data stuff:

- **Durable Storage**: Automatically creates a **PersistentVolumeClaim (PVC)**.
- **SD-Card Friendly**: Uses `local-path` provisioning—it only consumes the actual bytes you write (pay-as-you-grow).
- **Crash Proof**: Forces `--appendonly yes` so your data survives power cuts or Pod restarts.
- **Auto-Wiring**: Connection strings (like `REDIS_URL`, `DATABASE_URL`) are injected directly into your app's secrets.

## Self-hosting

The recommended setup:

- Cluster: Raspberry Pi 5 running k3s
- Ingress: Traefik (k3s default)
- Operator: webapp-operator
- Remote access: Tailscale — kubectl and deployit work from anywhere
- Public access: Cloudflare Tunnel — no open ports required

Total infrastructure cost: roughly 5 euros per month in electricity.