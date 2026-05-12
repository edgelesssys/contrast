# Setup to retrieve secrets from GCP using Workload Identity Federation (WIF)

Some CI jobs need to access the credentials (kubeconfig) for the testing clusters which are stored as GCP secrets.
The steps here describe how to set this access up using Workload Identity Federation (WIF).

## Prerequisites

```bash
# Some gcloud commands (for example, `iam service-accounts create`) reject the project
# number and require the project ID. The SA email also uses the ID, not the
# number. Use the ID consistently.
export PROJECT_ID="constellation-331613"
export SA_NAME="contrast-ci-darwin"
export SA_EMAIL="${SA_NAME}@${PROJECT_ID}.iam.gserviceaccount.com"
export POOL="contrast-github"
export PROVIDER="github"
export GH_REPO="edgelesssys/contrast"
```

`gcloud auth login`

The steps below require `roles/secretmanager.admin`, `roles/iam.workloadIdentityPoolAdmin`, and `roles/iam.serviceAccountAdmin` on the project (or `roles/owner`).

> [!NOTE]
> Members of `devs@edgeless.systems` inherit `roles/resourcemanager.projectIamAdmin` on `constellation-331613`, which is enough to self-elevate the three required roles for the duration of the setup.
> Revoke them at the end (see [Revoke operator elevation](#revoke-operator-elevation)).

```bash
ELEVATED_ROLES=(
  roles/secretmanager.admin
  roles/iam.workloadIdentityPoolAdmin
  roles/iam.serviceAccountAdmin
)

# elevate (allow ~30s for IAM to propagate before running the steps below)
for ROLE in "${ELEVATED_ROLES[@]}"; do
  gcloud projects add-iam-policy-binding "${PROJECT_ID}" \
    --member="user:$(gcloud config get-value account)" \
    --role="${ROLE}"
done
```

## 1. Create a Service Account (SA) for the Contrast CI jobs

This is the identity the GitHub CI jobs will be assuming.

```bash
gcloud iam service-accounts create "${SA_NAME}" --project "${PROJECT_ID}" --display-name "GH Actions: contrast darwin e2e"
```

## 2. Grant the SA access to the necessary credentials (kubeconfig)

`discovery-kubeconf` is intentionally spelled without the trailing `ig`, that matches the secret as it exists in GCP.
When adding a new bare-metal cluster kubeconfig, also grant the SA access here.

```bash
for SECRET in palutena-kubeconfig discovery-kubeconf olimar-kubeconfig dgx-007-kubeconfig; do
gcloud secrets add-iam-policy-binding "${SECRET}" \
  --project "${PROJECT_ID}" \
  --member "serviceAccount:${SA_EMAIL}" \
  --role "roles/secretmanager.secretAccessor"
done
```

## 3. Create the workload identity pool

This is a logical container in GCP for "external identities we trust" and groups related external identity providers.
WIF requires a pool to attach providers to.
We name the pool after the upstream identity source (`contrast-github`) which makes the trust boundary obvious.

```bash
gcloud iam workload-identity-pools create "${POOL}" \
  --project "${PROJECT_ID}" \
  --location "global" \
  --display-name "GitHub OIDC pool"
```

## 4. Create the OIDC provider in the pool

This tells GCP "trust OIDC tokens issued by GitHub and translate certain fields of those tokens into attributes you can use in IAM policies".

```bash
gcloud iam workload-identity-pools providers create-oidc "${PROVIDER}" \
  --project "${PROJECT_ID}" \
  --location "global" \
  --workload-identity-pool "${POOL}" \
  --display-name "GitHub" \
  --issuer-uri "https://token.actions.githubusercontent.com" \
  --attribute-mapping "google.subject=assertion.sub,attribute.repository=assertion.repository,attribute.repository_owner=assertion.repository_owner,attribute.ref=assertion.ref,attribute.event_name=assertion.event_name" \
  --attribute-condition "assertion.repository == '${GH_REPO}' && assertion.ref == 'refs/heads/main' && assertion.event_name in ['schedule', 'workflow_dispatch']"
```

Roughly how this works:

1. The workflow initiates an auth flow by sending a token signed by GitHub to GCP.
2. GCP fetches GitHub's public keys and verifies the token's signature (through the provider we created).
3. After validation succeeds, GCP parses the token, runs the attribute condition above, and maps GCP-side attributes from claims in the token.
4. GCP returns a short-lived federated token to the workflow.
   This isn't yet usable to read secrets, it just represents "an authenticated GitHub workflow from edgelesssys/contrast for this event."
5. The workflow exchanges the federated token via the IAM Credentials API for a short-lived access token impersonating the SA, and that access token is what actually reads the secrets.

### Why the condition is this strict

The only steady-state consumer is `release_nightly.yml`, which runs on `schedule` (always the default branch per GitHub) and on `workflow_dispatch`.
Restricting to `refs/heads/main` closes "any push to any branch by anyone with write access" as an abuse vector and makes a malicious `workflow_dispatch` from a feature branch ineffective.
`push` is intentionally absent because no consumer needs it.
`pull_request` is intentionally absent because PRs from forks must not be able to mint a token for our SA.

When iterating on a feature branch, you can temporarily widen the condition by adding a second clause that pins both the feature ref AND its trigger event, leaving the main steady-state clause untouched.
Use `gcloud iam workload-identity-pools providers update-oidc` to apply, and revert before merging.

```bash
gcloud iam workload-identity-pools providers update-oidc "${PROVIDER}" \
  --project="${PROJECT_ID}" \
  --location=global \
  --workload-identity-pool="${POOL}" \
  --attribute-condition="assertion.repository == '${GH_REPO}' && (
    (assertion.ref == 'refs/heads/main' && assertion.event_name in ['schedule', 'workflow_dispatch'])
    ||
    (assertion.ref == 'refs/heads/sse/e2e-release-darwin' && assertion.event_name == 'push')
  )"
```

Pinning both `ref` and `event_name` per clause prevents the feature branch from being usable through any trigger other than the one its workflow declares (for example, an attacker who manages to land a `workflow_dispatch` job on the branch can't mint a token for our SA, because that event_name isn't allowed alongside this ref).

## 5. Allow GH workflows in the repository to impersonate the SA

```bash
POOL_ID="$(gcloud iam workload-identity-pools describe "${POOL}" \
  --project "${PROJECT_ID}" --location global --format='value(name)')"

gcloud iam service-accounts add-iam-policy-binding "${SA_EMAIL}" \
  --project "${PROJECT_ID}" \
  --role "roles/iam.workloadIdentityUser" \
  --member "principalSet://iam.googleapis.com/${POOL_ID}/attribute.repository/${GH_REPO}"
```

This binding expresses "anyone who came in via the `contrast-github` pool AND whose token's repository claim was `edgelesssys/contrast` may impersonate `contrast-ci-darwin`."
The token-mint condition in step 4 already restricts which workflows can produce a federated token, so this binding is the second of two layers (token mint, then SA impersonation).

## 6. Configure the GitHub repository variables

> [!NOTE]
> None of these values is a secret.

```bash
echo "  GCP_WIF_PROVIDER = $(gcloud iam workload-identity-pools providers describe "${PROVIDER}" \
    --project "${PROJECT_ID}" --location global \
    --workload-identity-pool "${POOL}" \
    --format='value(name)')"
echo "  GCP_SA_EMAIL     = ${SA_EMAIL}"
```

Add them as variables (not secrets) in the GitHub repository.
The first is used to fetch the federated token, the second to impersonate the SA.

## 7. Wire up the workflow

In the job that needs the kubeconfig, set `permissions: id-token: write` and call `google-github-actions/auth`:

```yaml
permissions:
  id-token: write
  contents: read
steps:
  - uses: google-github-actions/auth@v2
    with:
      workload_identity_provider: ${{ vars.GCP_WIF_PROVIDER }}
      service_account: ${{ vars.GCP_SA_EMAIL }}
  - run: |
      gcloud secrets versions access latest --secret palutena-kubeconfig > kubeconfig
```

The `auth` step is the verification: if federation or impersonation isn't configured properly, it fails the job loudly with a 403 from GCP.

## Revoke operator elevation

Once all setup steps above have run successfully, drop the temporary admin roles granted in Prerequisites:

```bash
for ROLE in "${ELEVATED_ROLES[@]}"; do
  gcloud projects remove-iam-policy-binding "${PROJECT_ID}" \
    --member="user:$(gcloud config get-value account)" \
    --role="${ROLE}"
done
```

This leaves the operator with only their group-inherited permissions.
The SA bindings, pool, and provider remain in place.

## Revocation and rotation

To remove the Service Account's ability to be impersonated without touching the pool or provider:

```bash
gcloud iam service-accounts remove-iam-policy-binding "${SA_EMAIL}" \
  --project "${PROJECT_ID}" \
  --role "roles/iam.workloadIdentityUser" \
  --member "principalSet://iam.googleapis.com/${POOL_ID}/attribute.repository/${GH_REPO}"
```

To rotate the SA, create a new one, repeat steps 1, 2, and 5 with the new name, update `GCP_SA_EMAIL` in the repository variables, then delete the old SA.
The pool and provider don't need to change.

For audit: Cloud Audit Logs record `SetIamPolicy` on the SA and `AccessSecretVersion` on each kubeconfig secret.
