# Advanced debugging

In some cases, additional information from within a pod VM is necessary to diagnose issues.
Contrast can configure the pod VM for interactive debugging via initdata that's passed to the guest.
The guest then enables an SSH-based debug service reachable from within the pod sandbox.

![Debug shell architecture](../_media/debugshell.drawio.svg)

:::danger

Enabling debug access is inherently insecure.
Never use this option on production workloads or with sensitive data involved.

:::

Enabling the debug shell requires two independent changes:

1. A workload-policy change (the kata-agent must accept `exec`, `read-stream`, and `write-stream` requests), opted into by editing `settings.json` before running `contrast generate`.
2. A pod-spec change (the `contrast-debug-shell` sidecar container and debug initdata), opted into via the `--insecure-enable-debug-shell-access` flag on `contrast generate`.

## Configure the policy

Edit your `settings.json` so the generated policy permits interactive exec into the pod VM.
The defaults shipped with Contrast releases are deliberately restrictive. For debug shell access, the following three keys under `request_defaults` must be loosened:

```json
{
    ...
    "request_defaults": {
        ...
        "ExecProcessRequest": {
            "allowed_commands": [],
            "regex": [ ".*" ]
        },
        ...
        "ReadStreamRequest": true,
        ...
        "WriteStreamRequest": true
    }
}
```

- `ExecProcessRequest.regex` allows the kata-agent to spawn matching commands when `kubectl exec` is called. `".*"` allows any command, which is the broadest setting.
- `ReadStreamRequest: true` lets the agent stream stdout/stderr back. Without this, `kubectl exec` and `kubectl logs` produce no output.
- `WriteStreamRequest: true` lets the agent accept stdin. Without this, the shell prompt prints but every keystroke is dropped (the symptom looks like a hung TTY).

## Generate with the debug shell flag

```sh
contrast generate --insecure-enable-debug-shell-access
```

This does two things:

- Injects the `contrast-debug-shell` sidecar container into every workload that uses a `contrast-cc` runtime class.
- Adds the following keys to initdata, which the guest reads to enable the SSH debug service:

  ```toml
  [data]
  'contrast.insecure-debug' = 'true'
  ```

Enabling debug features via initdata is covered by the measurements of a pod VM and is therefore detectable via remote attestation.

Deploy the regenerated resources before continuing.

## Collect pod VM logs

The `contrast-debug-shell` container exposes the pod VM's journal as Kubernetes container logs, so they can be collected with the usual tooling:

```sh
kubectl logs <pod-name> -c contrast-debug-shell
```

This only requires `ReadStreamRequest: true` in `settings.json` (the default).

## Open an interactive shell in the pod VM

```sh
kubectl exec -it <pod-name> -c contrast-debug-shell -- debugshell
```

You should see output like:

```
Warning: Permanently added '[localhost]:2222' (RSA) to the list of known hosts.

[root@nixos:/]#
```
