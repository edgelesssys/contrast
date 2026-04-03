# Debugging e2e failures

## Collecting logs

### After a `just e2e` run

`just e2e` deploys a log-collector DaemonSet that streams pod logs from the
start. After the test finishes (pass or fail), download the logs:

```bash
just download-logs
```

Logs are written to `workspace/logs/`.

### After a manual deployment (`just`)

If you deployed with `just` (the default target) and want to collect logs:

```bash
just download-logs
```

This deploys the log-collector DaemonSet (if not already running), collects
host-level journal entries, and downloads everything.

### In CI

CI runs `just download-logs` automatically after every e2e test. Logs are
uploaded as GitHub Actions artifacts. To find them: go to the workflow run,
scroll to the bottom of the run summary page, and look for artifacts named
`e2e_pod_logs-<platform>-<test>` (for example, `e2e_pod_logs-Metal-QEMU-SNP-openssl`).
Alternatively you can expand the "Upload logs" step in a particular test and
get the Artifact download URL.

## Log structure

```
workspace/logs/
├── <namespace>_<pod>_<uid>/       # pod container logs
│   └── <container>/0.log
├── host/                          # host-level journal logs
│   ├── kernel.log                 # journalctl -k (SEV-ES termination, VFIO/IOMMU)
│   ├── k3s.log                    # journalctl -u k3s (k3s-specific kubelet/containerd)
│   ├── kubelet.log                # journalctl -u kubelet (non-k3s runners)
│   ├── containerd.log             # journalctl -u containerd (non-k3s runners)
│   └── kata.log                   # journalctl -t kata (QEMU lifecycle, register dumps)
└── <namespace>-k8s-events.yaml    # kubernetes events
```

Host logs are time-scoped to the namespace creation time, so they only contain
entries relevant to the test run.

## Debugging CVM failures

CVM boot failures (for example, SEV-ES termination, OVMF crashes) leave no trace in
pod logs -the guest never starts. Look at host-level logs instead:

1. **kernel.log** -look for `SEV-ES guest requested termination`, VFIO/IOMMU
   errors, or KVM failures.
2. **kata.log** -look for `detected guest crash`, QEMU launch arguments,
   register dumps, and console output (`vmconsole=` lines contain guest serial
   output).
3. **k3s.log** -look for `task is in unknown state` or containerd errors that
   indicate the CVM process died.

## Tracing a pod to its sandbox in kata.log

kata.log contains interleaved logs from all sandboxes. To find logs for a
specific pod, you need to go from runtime class to sandbox ID.

1. Find the test namespace:

```bash
ns=$(cat workspace/just.namespace)
```

2. List pods and their runtime classes:

```bash
kubectl get pods -n "$ns" -o custom-columns='NAME:.metadata.name,RUNTIME:.spec.runtimeClassName'
# NAME                              RUNTIME
# openssl-backend-85cd89c76-q6w6h   contrast-cc-metal-qemu-snp-fd4512d5
# openssl-frontend-97f6d865d-mvph4  contrast-cc-metal-qemu-snp-fd4512d5
# coordinator-0                     contrast-cc-metal-qemu-snp-fd4512d5
```

3. Get the runtime hash and list sandbox IDs. The hash is the last component
   of the runtime class name (for example, `fd4512d5` from
   `contrast-cc-metal-qemu-snp-fd4512d5`):

```bash
hash="fd4512d5"
grep "$hash" workspace/logs/host/kata.log | grep -oP 'sandbox=\K[a-f0-9]+' | sort -u
# 8c0277d1f16fbbce871d5ab4982fbc376aa7c39e3ae00615cf0722f9b0d7f9a9
# 2fbc275ee03b0fe9929f4e6ef474dd6e855c1da6cfbcd4843a535e51e8ba5441
# 548cbab9ee75a3f1...
```

   Each sandbox ID corresponds to one pod/CVM.

4. Filter kata.log for a specific sandbox:

```bash
sandbox="8c0277d1f16fbbce"
grep "$sandbox" workspace/logs/host/kata.log
```

### Fallback: Finding sandboxes by runtime class hash

If a pod is missing from the sandbox map (deleted before log collection), you
can find its sandbox ID using the runtime class hash from kata.log. The hash
is the last component of the runtime class name (for example, `d17bc85e` from
`contrast-cc-metal-qemu-snp-d17bc85e`):

```bash
grep "d17bc85e" workspace/logs/host/*/kata.log | grep -oP 'sandbox=\K[a-f0-9]+' | sort -u
```

This lists all sandbox IDs for that runtime class. Cross-reference with the
sandbox map to identify which ones are unmapped.

Note that some kata log lines (config loading, factory init, device cold plug)
don't have a sandbox ID. These are shared across all CVMs and may be relevant
for debugging startup failures.
