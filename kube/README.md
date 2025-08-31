# Kubernetes Resources for Verificat

## External Secrets Operator with Localstack

### About

In order to use AWS Secret Manager (ASM) values inside Kubernetes, there must be middleware. External Secrets Operator (ESO) is an option that can use an arbitrary external store to fill Kubernetes Secrets.

This guide sets up a docker `pullImageSecret` that can be used to deploy from *GitHub Packages* (aka *GitHub Container Registry* or `ghcr(.io)`. It uses the following components:

- LocalStack using
  - AWS SecretsManager
  - AWS IAM
- Orbstack running Kubernetes

### Prerequisites

The following is assumed by this guide:

- The user can authenticate with AWS SSO (via `aws sso login`)
- The user has set up `~/.aws/config` using profiles
- There is a local docker engine (Orbstack or Docker Desktop)
- There is a local kubernetes cluster (Orbstack or DD)
- The AWS CLI is installed (i.e.: `aws_cli`, or just `aws`)
- LocalStack is installed (requires GitHub auth to setup)
- Helm is required to install the ESO Chart (`brew install helm`)
- The user has `kubectl` and `kubectx` installed (Orbstack should auto-populate `~/.kube/config`)

### Local Walkthrough

> [!Note]
>
> This walkthrough uses Orbstack as both Docker engine and Kubernetes cluster.
> Docker Desktop should work similarly.

#### Orbstack with Kubernetes

1. Run Orbstack: `orb start`
2. Start up Kubernetes: `orb start k8s`
3. Configure Orbstack to bridge container IP addresses to macOS (*required for using LocalStack DNS from Kube*):

- In **OrbStack Desktop** go to **Settings > Network** and toggle on **"Allow access to container domains & IPs"**

#### LocalStack Setup

> [!Note]
>
> This assumes LocalStack is installed properly.

1. Run LocalStack: `localstack start -d --network ls`
2. Get the DNS for LocalStack's container (see [Access via endpoint URL](https://docs.localstack.cloud/references/network-troubleshooting/endpoint-url/#from-your-container)), usually something like `192.168.97.2`.

- `docker inspect localstack-main | jq -r '.[0].NetworkSettings.Networks | to_entries | .[].value.IPAddress'`

3. Set `dnsConfig.nameservers` in `values.yaml` with the Localstack container IP

#### Configure AWS Resources in LocalStack

> [!Note]
>
> LocalStack has a few ways of being run to interact with AWS Resources.
>
> The method here is to use the original `aws_cli` along with the IAM prefix: `aws --prefix localstack`

##### AWS SSO

Due to the way we use profiles in conjunction with SSO, the easiest setup for localstack is to add a profile that will still authenticate via SSO and allow you to use a localstack profile. The following should be added to your `~/.aws/config`:

```
[profile localstack]
role_arn = arn:aws:iam::197533337274:role/devops
source_profile = default
endpoint_url = http://localhost:4566
```

A few more things to make sure it's working correctly:

1. To keep the `aws_cli` commands simple, set the AWS region with: `export AWS_REGION=us-west-2`
2. Authenticate to AWS: `aws sso login`
3. There is an oddity with LocalStack and our SSO, usually this extra step must be done:
   1. Confirm you have test account access: `aws --profile test sts get-caller-identity`
   2. You should see an ARN like `arn:aws:sts::197533337274:assumed-role/devops/botocore-session-1740684713`
   3. Do something with the test account, an easy one is: `aws --profile test s3 ls`
4. Now confirm `aws_cli` is working with LocalStack:
   1. `aws --profile localstack sts get-caller-identity`
   2. You should see an ARN like `arn:aws:iam::000000000000:root`

##### AWS IAM

> [!Important]
>
> Make sure you are using the correct Kube cluster, for Orbstack it is named: `orbstack`

1. If you haven't already, **switch `kubectl` context now**: `kubectx orbstack`
2. Create a new AWS IAM Principal
   1. Add a user to get the `ACCESSKEY`: aws --profile localstack iam create-user --user-name <USER>`
   2. Add a key to get the `SECRETKEY`: aws --profile localstack iam create-access-key --user-name <USER>`
3. Create a Kubernetes Secret for AWS SecretsManager access from ESO
   1. `echo ACCESSKEY > ./lstack-access-key`
   2. `echo SECRETKEY > ./lstack-secret-access-key`
4. `kubectl create secret generic awssm-secret --from-file=./lstack-access-key --from-file=./lstack-secret-access-key`

##### AWS SecretsManager

Create the AWS SM secret to be sync'd

1. Encode the full JSON used for `imagePullSecrets`: `echo '{ "auths": { "ghcr.io": { "username": "DieselDevEx", "password": "ghp_<token>" } } }' | base64`
2. `aws --profile localstack secretsmanager create-secret --name "k8s/application/github/ghcr-pullimage" --description "Base64 encoded docker imagePullSecret JSON for ReadOnly GitHub Package Access" --secret-string "<base64-encoded-json>"`

##### Install External Secret Operator (ESO)

> [!Note]
>
> This set of configurations follow the [Getting Started walkthrough](https://external-secrets.io/latest/introduction/getting-started/) for ESO.
>
> See also: [Cluster External Secret](https://external-secrets.io/v0.7.0/api/clusterexternalsecret/)

**Make sure you're using the correct Kube cluster: `kubectx orbstack`**

1. Configure `helm` with the ESO repository: `helm repo add external-secrets https://charts.external-secrets.io`
2. Install ESO:
   1. The `values.yaml` file contains the environment variable settings that point ESO to LocalStack instead of real AWS remote endpoints.
   2. Values should also include `dnsConfig.nameservers` updated with the LocalStack IP you found above.
   3. `helm install external-secrets external-secrets/external-secrets -n external-secrets --create-namespace --values ./values.yaml`
3. Install the **ClusterStore**: `kubectl apply -f eso-clusterstore.yaml`
4. Define the **Cluster External Secret**: `kubectl apply -f eso-clustersecret-pullimage.yaml`

##### Validate

Check the status of all ESO objects:

- `kubectl get SecretStores,ClusterSecretStores,ExternalSecrets,ClusterExternalSecrets --all-namespaces`

You should see an answer like this, where **READY=True**:

```
NAMESPACE   NAME                                                      AGE   STATUS   CAPABILITIES   READY
            clustersecretstore.external-secrets.io/eso-clusterstore   18h   Valid    ReadWrite      True

NAMESPACE   NAME                                                         STORE              REFRESH INTERVAL   READY
            clusterexternalsecret.external-secrets.io/docker-pullimage   eso-clusterstore   1m                 True
```

To **verify**, install Verificat and run the smoke test (see below, **Verificat Manifests**).

## Verificat Manifests

1. Create the namespace: `verificat-namespace.yaml`
   *This file can be used to create the namespace before deploying non-Cluster ESO objects.*
2. Edit `verificat-app.yaml` and set `GH_TOKEN` to the PAT needed for Verificat to work.
   *This is the same PAT used in the `pullImageSecret`.*
3. Deploy Verificat: `kubectl apply -f verificat-app.yaml`
4. Determine the `EXTERNAL_IP` for `verificat`: `kubectl -n verificat get svc`
5. Smoketest Verificat:
   1. Switch to the `verificat` repo root
   2. Export these environment variables:
      1. `export GH_TOKEN=ghp_0987654321qwerty`
      2. `export BACKSTAGE="https://backstage.rainbowq.co"`
      3. `export PORT=4330`
   3. Alternatively, set them in `./.env` and run: `set -a; source .env`
   4. Run: `./server/testdata/smoketest.sh http://EXTERNAL_IP`

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

## Uninstall

There isn't anything too fancy about removal.

1. Use `kubectl apply -f verificat-app.yaml` to remove `verificat`.
2. Use `kubect delete` with the resources you found in the **Validate** step.
3. Remove ESO completely with: `helm delete external-secrets --namespace external-secrets`

If you get stuck installing or uninstalling, there is an option in Orbstack to reset Kubernetes. This will erase everything and get you to the default kubernetes installation.

> [!Important]
>
> Shutting down Orbstack without uninstalling or killing anything should not hurt the deployments. Putting your laptop asleep without shutting down anything should be fine, too.
>
> Resources created by LocalStack should persist between restarts. If you need to restart it, run the start command again to reload the cache (you may get a warning that `localstack-main` is already running): `localstack start -d --network ls`
