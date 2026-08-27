# CI runner decommissioning

## Remove runner from GitHub

1. Go to the repository settings.
2. Click on "Actions" in the left sidebar.
3. Click on "Runners" in the left sidebar.
4. Find the runner you want to remove and click on it.
5. Click on "Remove" and confirm the action.

## Rotate CI secrets

These are shared between different CI runners, so need to be rotated:

- `CONTRAST_GHCR_READ`

This can be done via GitHub UI by the `edgelessci` account.

After regenerating the token, update it in the Google cloud project:

```
echo -n "<token>" | gcloud secrets versions add ghcr-read-token --project=796962942582 --data-file=-
```

Post a message in the Teams channel to alert devs to re-run

```
just get-ghcr-read-token
```

## Revoke the kubeconfig

*only applicable if the runner was part of a multi-node cluster*

The kubeconfig sat on the machine, so treat it as compromised the moment the machine leaves our
control. Its token is tied to the API key that minted it, so delete that key. Nothing else in the
cluster is affected.

A replacement runner needs a new key and a kubeconfig generated for it, see
[bare-metal runner setup](./bare-metal-runner.md#developer-access).
