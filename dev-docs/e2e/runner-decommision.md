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

## Rotate kubeconfig

*only applicable if the runner was part of a multi-node cluster*

1. Create a new kubeconfig.
