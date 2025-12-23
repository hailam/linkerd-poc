# Linkerd Service Mesh PoC

Two services communicating via Linkerd mTLS, where the receiver identifies the caller using the `l5d-client-id` header.

## How It Works

```
caller-app ──► linkerd-proxy ══mTLS══► linkerd-proxy ──► echo-service
                   │                        │
                   └── injects l5d-client-id header
```

1. Linkerd injects a sidecar proxy into each pod
2. All traffic between services goes through these proxies
3. Proxies establish mTLS connections automatically (no app code changes)
4. The receiving proxy injects `l5d-client-id` header with the caller's identity

**Header format:** `<service-account>.<namespace>.serviceaccount.identity.linkerd.cluster.local`

## Prerequisites

```bash
# Install Linkerd CLI (macOS)
brew install linkerd

# Install Linkerd on cluster
linkerd install --crds | kubectl apply -f -
linkerd install --set proxyInit.runAsRoot=true | kubectl apply -f -
linkerd check
```

> Note: `--set proxyInit.runAsRoot=true` is required for Docker runtime (OrbStack, some EKS configs).

## Deploy

```bash
# Build image
docker build -t linkerd-demo:latest ./app

# Apply namespace first, then services
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/caller-app.yaml
kubectl apply -f k8s/echo-service.yaml

# Verify pods have 2 containers (app + linkerd-proxy)
kubectl -n mesh-demo get pods
```

## Test

```bash
kubectl -n mesh-demo port-forward deploy/caller-app 8080:8080
curl -s http://localhost:8080/call-echo | jq
```

Expected output:
```json
{
  "self": { "app_name": "caller" },
  "echo_response": {
    "self": { "app_name": "echo-server" },
    "caller": {
      "l5d_client_id": "default.mesh-demo.serviceaccount.identity.linkerd.cluster.local"
    }
  }
}
```

**If `caller` is empty:** Linkerd is not installed or sidecars not injected. Run:
```bash
linkerd check
kubectl -n mesh-demo rollout restart deploy
```

## Endpoints

| Endpoint     | Description                                   |
| ------------ | --------------------------------------------- |
| `/`          | Returns self + caller identity from headers   |
| `/call-echo` | Calls echo-service, returns combined response |
| `/health`    | Health check                                  |

## Cleanup

```bash
kubectl delete namespace mesh-demo
```
