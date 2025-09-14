# Kubernetes Resources for Verificat

## About

This guide sets up a docker `pullImageSecret` that can be used to deploy from *GitHub Packages* (aka *GitHub Container Registry* or `ghcr(.io)`. It uses Orbstack running Kubernetes.

### Prerequisites

The following is assumed by this guide:

- There is a local docker engine (Orbstack or Docker Desktop)
- There is a local kubernetes cluster (Orbstack or DD)
- Helm is required 

### Orbstack with Kubernetes

1. Run Orbstack: `orb start`
2. Start up Kubernetes: `orb start k8s`
3. Configure Orbstack to bridge container IP addresses to macOS

- In **OrbStack Desktop** go to **Settings > Network** and toggle on **"Allow access to container domains & IPs"**

## Prepare Secrets

1. Encode the full JSON: `echo '{ "auths": { "ghcr.io": { "username": "GITHUB_USER", "password": "GITHUB_TOKEN" } } }' | base64`
2. Add this to `./docker-secret.yaml`
3. Encode the PAT only: `echo $GITHUB_TOKEN | base64`
4. Add this as `GH_TOKEN` to `./verificat-app.yaml`

## Verificat Manifests

This assumes `traefik` is installed and operational.

1. Create the namespaces: `kubectl apply -f verificat-namespace.yaml`
1. Deploy Traefik Ingress: `kubectl apply -f verificat-ingress.yaml`
1. Deploy the secret: `kubectl apply -f docker-secret.yaml`
1. Deploy the App: `kubectl apply -f verificat-app.yaml`
1. Determine the `CLUSTER-IP` for `verificat`: `kubectl -n verificat get service/verificat`
1. Add a line to `/etc/hosts` with the entry
   - e.g.: `192.168.194.191 verificat.rainbowq.net`
 
This should make <http://verificat.rainbowq.net> available, you will see a lime-green screen with **Verificat | Production Readiness Scores** at the top. 

Smoketest Verificat:
1. Switch to the `verificat` repo root
1. Export these environment variables:
  1. `export GH_TOKEN=ghp_0987654321qwerty`
  1. `export BACKSTAGE="https://backstage.rainbowq.co"`
  1. `export PORT=4330`
1. Alternatively, set them in `./.env` and run: `set -a; source .env`
1. Run: `./server/testdata/smoketest.sh http://verificat.rainbowq.net`

A successful smoketest looks like this:

```verificat-smoketest
>>> ./server/testdata/smoketest.sh http://198.19.249.2

::: Running smoketest for Verificat at http://198.19.249.2 :::

Source .env for EnvVars... loaded:
 GH_TOKEN=<REDACTED>
 BACKSTAGE=https://backstage.rainbowq.co
 PORT=4330

Healthz endpoint... ok
Almanac download...       40 bytes
Admin service check... {
[...]
}
```
