# How to add a bare-metal instance to the CI

## Install Ubuntu LTS server

**This step is only relevant when we set up the host OS.**

Download and install the latest Ubuntu LTS server from https://ubuntu.com/download/server.

When configuring the disk layout, ensure to use btrfs as the root filesystem.

## SNP setup

Creating AMD SEV-SNP guests via KVM is supported by kernels newer than 6.11 (see https://www.phoronix.com/news/Linux-6.11-KVM). If the kernel is older than that, update it, for example via

```bash
sudo apt install linux-generic-hwe-24.04
```

Check that SEV-SNP is enabled. If it's not then it likely needs to be
enabled in the BIOS. For those steps, either have a look in our docs
https://docs.edgeless.systems/contrast/howto/cluster-setup/bare-metal or
google for "enable AMD SEV in BIOS." Sadly, AMD changes their document
links from time to time, so we don't link it here.

Once it's enabled, verify is using the `snphost` tool:

```bash
sudo snphost ok
```

## TDX setup

Follow <https://docs.edgeless.systems/contrast/howto/cluster-setup/bare-metal?vendor=intel#hardware-and-firmware-setup>, but pay attention to the following:

- The passwords chosen during PCCS configuration are only important for the setup phase.
  Pick a random one (`head -c8 /dev/urandom | base64`) for both user and admin, and keep it for the platform registration step.
- Don't reboot after DCAP installation, but after TDX module update.
- Take the PCS API key from `/opt/intel/sgx-dcap-pccs/config/default.json` of an existing machine.
- Use the *Online, manual, single platform, PCCS-based Indirect Registration* flow to register the platform.
  The correct invocation of the `PCKIDRetrievalTool` looks like this:

  ```bash
  PCKIDRetrievalTool -url https://localhost:8081 -user_token $PASSWORD_FROM_ABOVE -use_secure_cert false
  ```

- After running the `PCKIDRetrievalTool`, there should be a CSV file in the current directory that contains the platform manifest.
  Extract the PIID with the following command (it will be needed later for the [bare metal runner specification](#bare-metal-runner-specification) step).

  ```bash
  cat pckid_retrieval.csv | sed 's/,/\n/g' | tail -n1 | head -c160 | tail -c 32; echo
  ```

## Install required packages

Install `docker` so that the docker login step in the CI succeeds.
On Ubuntu, add it to the apt repositories (see https://docs.docker.com/engine/install/ubuntu/#install-using-the-repository).

```bash
# Add Docker's official GPG key:
sudo apt-get update
sudo apt-get install ca-certificates curl
sudo install -m 0755 -d /etc/apt/keyrings
sudo curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc
sudo chmod a+r /etc/apt/keyrings/docker.asc

# Add the repository to Apt sources:
sudo tee /etc/apt/sources.list.d/docker.sources > /dev/null <<EOF
Types: deb
URIs: https://download.docker.com/linux/ubuntu
Suites: $(. /etc/os-release && echo "${UBUNTU_CODENAME:-$VERSION_CODENAME}")
Components: stable
Architectures: $(dpkg --print-architecture)
Signed-By: /etc/apt/keyrings/docker.asc
EOF
sudo apt-get update

# Download docker package
sudo apt-get install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
```

> [!WARNING]
> On a non-k3s node, `dpkg` offers to replace `/etc/containerd/config.toml`, which the
> node-installer owns (`nodeinstaller/internal/targetconfig/targetconfig.go`). Answer `N`. The
> packaged version drops the CRI registry config and every runtime handler from under a running
> kubelet. k3s nodes are unaffected, k3s ships its own containerd.

## Kubernetes setup

These steps depend on the Kubernetes distribution used for this runner.

### Remove old runner (if applicable)

> [!IMPORTANT]
> Each testing cluster must have exactly one node labelled `ci.contrast.edgeless.systems/main-runner=true`.

If there's already a runner in the cluster, it needs to be removed before the new runner can be added.
Two runners break the assumption that e2e tests never run concurrently, since the node-installer
restarts containerd. Two `main-runner` nodes break the `hostpath` CSI driver, whose volumes are
node-local.

Remove the label `ci.contrast.edgeless.systems/main-runner=true` from all nodes in the cluster before continuing.

### K3s

Add K3s configuration override

<!-- NOTE:
If you change something here, make sure to change packages/cleanup-bare-metal.sh, too!
-->

```bash
mkdir -p /etc/rancher/k3s
cat > /etc/rancher/k3s/config.yaml <<EOF
write-kubeconfig-mode: "0640"
write-kubeconfig-group: sudo
disable:
  - local-storage
kubelet-arg:
  - "runtime-request-timeout=5m"
node-label:
  - ci.contrast.edgeless.systems/main-runner=true
embedded-registry: true
EOF
cat > /etc/rancher/k3s/registries.yaml <<EOF
mirrors:
  "*":
EOF
```

**IMPORTANT**: If the node is going to run a different Kubernetes distribution, make sure to apply these overrides in some other way.

Install K3s

```bash
curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION=v1.34.1+k3s1 sh -
```

The K3s docs state:
> A kubeconfig file will be written to /etc/rancher/k3s/k3s.yaml and the kubectl installed by K3s will automatically use it.

Export the Kubeconfig for the current user for the following steps:

```bash
export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
```

### Scaleway Kosmos

Follow Scaleway's documentation for joining the node to an existing Kosmos node pool: <https://www.scaleway.com/en/docs/kubernetes/how-to/edit-kosmos-cluster/#how-to-configure-external-nodes-to-join-the-cluster>.

Edit `/etc/systemd/system/kubelet.service` and apply the following modifications.

- Add the flag `--runtime-request-timeout=5m`.
- Append `ci.contrast.edgeless.systems/main-runner=true` to the already existing `--node-labels` flag.

Restart the Kubelet with `systemctl restart kubelet`.

Install `kubectl`:

```bash
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
mv kubectl /usr/local/bin/
chmod a+x /usr/local/bin/kubectl
kubectl version
```

Download the kubeconfig for the cluster as described under [Developer access](#developer-access),
and save it to `/home/github/.kube/config`.

## Kubernetes resources

Install the `hostpath` CSI driver:

```bash
nix build .#base.csi-driver-host-path
kubectl apply -k result
```

Its volumes are node-local, so when a runner is replaced, `collateral-proxy` is stranded on the
old node and needs re-provisioning:

```bash
just collateral-proxy-redeploy
```

## Kernel config

Follow https://docs.edgeless.systems/contrast/howto/cluster-setup/bare-metal#kernel-setup.

On AMD machines, enable the `msr` module:

```bash
cat > /etc/modules-load.d/snphost.conf << EOF
# msr module is required for sanity checks done by snphost
msr
EOF
systemctl restart systemd-modules-load.service
```

Ubuntu 23.10 and newer restrict unprivileged user namespaces via AppArmor, which breaks the
`image-podvm-gpu` build with `unshare: write failed /proc/self/uid_map: Operation not permitted`.
Exempt the binary that needs it:

```bash
sudo tee /etc/apparmor.d/nix-unshare > /dev/null <<'EOF'
abi <abi/5.0>,
include <tunables/global>

profile nix-unshare /nix/store/*-util-linux-*/bin/unshare flags=(unconfined) {
  userns,
  @{exec_path} mr,

  include if exists <local/nix-unshare>
}
EOF
sudo apparmor_parser -r /etc/apparmor.d/nix-unshare
```

## Networking

**This step only applies to runners owned by Edgeless Systems, not bare metal cloud machines.**

Add the device to the Tailscale network.
For this you have to have admin privileges, if you don't see the
overview of machines when visiting
https://login.tailscale.com/admin/machines, notify another engineer.

On this page, click "Add device." In the settings add the "ssh-access"
label. This is needed since all engineers have the "devs"
role, which allows them to ssh into all devices that have the "ssh-access"
tag.
Follow the other instructions on the Tailscale website to add the device
and execute the given script on the machine.
After the installation, execute:

```bash
sudo tailscale up --ssh
```

Add a firewall for incoming connections if the server is reachable via
a public IP, like on Hetzner:

```bash
ufw status
ufw app list
ufw allow 22
ufw allow OpenSSH
ufw show added
ufw enable
```

## Installing dependencies

Install Nix using the official instructions:
https://nixos.org/download/#nix-install-linux.

## Add server as a GitHub runner

First, create another user, which the runner service will use.

```bash
useradd -s /bin/bash -m -G sudo,docker github
```

Put the K3s kubeconfig into the default dir for the user (the symlink shown below is for k3s only):

```bash
mkdir -p /home/github/.kube
ln -s /etc/rancher/k3s/k3s.yaml /home/github/.kube/config
```

Customize the Nix configuration for flakes, the GitHub runner and Cachix:

```bash
cat > /etc/nix/nix.conf <<EOF
extra-experimental-features = nix-command flakes
auto-optimise-store = true
build-users-group = nixbld
bash-prompt-prefix = (nix:$name)\040
max-jobs = auto

# Trust the Github runner and all admins.
trusted-users = [ github @sudo ]
# Allow overriding the trusted substituters from flake config to enable Cachix.
accept-flake-config = true
EOF
systemctl restart nix-daemon
```

Check what filesystem the server has:
```bash
findmnt /
```

If it's anything other than a btrfs, setup a btrfs builder volume.
The instructions are taken from https://github.com/edgelesssys/contrast/blob/a62af98f2df761116109310a6af4adcb66e758c0/.github/actions/setup_nix/action.yml#L35.

```bash
# not installed when the root filesystem isn't btrfs
sudo apt-get install -y btrfs-progs

# Create file fs backend
echo "Setting up btrfs nix builder volume..."
sudo mkdir -p /mnt/nixbld
sudo fallocate -l 50G /mnt/btrfs.img
sudo mkfs.btrfs -f /mnt/btrfs.img

# Create fstab entry to mount the file as btrfs
sudo tee -a /etc/fstab > /dev/null <<'EOF'
# btrfs for nix builder
/mnt/btrfs.img /mnt/nixbld btrfs loop,defaults 0 0
EOF
sudo systemctl daemon-reload
sudo mount -a

# `mount -a` skips a malformed entry silently. --mountpoint reports nothing unless
# /mnt/nixbld is itself a mount, rather than falling back to an ancestor.
findmnt --mountpoint /mnt/nixbld

# Use the btrfs for nix builds
echo "build-dir = /mnt/nixbld" | sudo tee -a /etc/nix/nix.conf
sudo systemctl restart nix-daemon
```

Moreover, the e2e tests expect reference values for the CC-technology
(TDX/SNP) to be present in a configmap inside the cluster.
Follow the steps in the [chapter below](#bare-metal-runner-specification).

Execute the commands under https://github.com/edgelesssys/contrast/settings/actions/runners/new for "Download" and "Configure" as
the `github` user in their home directory.

During the configuration step, always press ENTER to use the default
settings. Don't execute `run.sh`, instead configure the runner
to start as a service. The instruction are taken from
https://docs.github.com/en/actions/hosting-your-own-runners/managing-self-hosted-runners/configuring-the-self-hosted-runner-application-as-a-service
```bash
sudo ./svc.sh install github
sudo ./svc.sh start
sudo ./svc.sh status
```

Verify that the PATH in `/home/github/actions-runner/.path` contains
the nix paths. The installer snapshots your PATH variable during
installation. If the paths don't exist, then copy over your PATH into
`/home/github/actions-runner/.path` and restart the service via:
```bash
systemctl restart actions.runner.edgelesssys-contrast.$(hostname).service
```

Add the necessary tags for the runner in GitHub by navigating to
https://github.com/edgelesssys/contrast/settings/actions/runners
selecting the newly added runner and adding the labels the runner fulfills,
that's `tdx` for TDX servers and `snp` for SNP servers (or `tdx-gpu` and `snp-gpu`, respectively).

## Developer access

For developers to be able to access the K8s cluster, prepare a kubeconfig and save it as
`${RUNNER_NAME}-kubeconfig`.

### K3s kubeconfig

Point it at the DNS name of the server inside the Tailscale:

```bash
CONFIG=$(cat /etc/rancher/k3s/k3s.yaml)
CONFIG="${CONFIG//default/$(hostname)}"
CONFIG="${CONFIG//127.0.0.1/$(hostname)}"
echo "${CONFIG}" > $(hostname)-kubeconfig
```

### Scaleway Kosmos kubeconfig

In the Scaleway portal, create an API key dedicated to this machine, owned by the application that
has Kubernetes access, then generate a Kubernetes configuration for that key. The token is tied to
the key, which is what makes it revocable when the machine is decommissioned.

It must carry a static token: CI and `just get-credentials` run where no `scw` is configured, and
recent `scw k8s kubeconfig get` emits an `exec` credential instead. Check both before uploading:

```bash
kubectl --kubeconfig "${RUNNER_NAME}-kubeconfig" config view -o json | jq -e '.users[0].user | has("token")'
KUBECONFIG="${RUNNER_NAME}-kubeconfig" kubectl get nodes
```

### Push it to GCP

Copy the config over to somewhere you are already
authenticated with GCP and push it as a secret. If the secret already
exists, only execute the `gcloud secrets versions add` command.
Set `RUNNER_NAME` to the hostname of the runner to be added.

```bash
gcloud secrets create ${RUNNER_NAME}-kubeconfig --replication-policy="automatic" --project constellation-331613
gcloud secrets versions add ${RUNNER_NAME}-kubeconfig --data-file="./${RUNNER_NAME}-kubeconfig" --project constellation-331613
```

Add the secret to the secrets retrieved via `just` in
https://github.com/edgelesssys/contrast/blob/f14824f6c039e47a96cc0bbf2298bce5aa8e9844/justfile#L334

## Bare-metal runner specification

To run our e2e test with the real bare-metal runner specification, a ConfigMap named `bm-tcb-specs` is added to all e2e clusters.
Having the ConfigMap prevents using committed values in the e2e tests directly, which could otherwise lead to backporting problems.

The `bm-tcb-specs` ConfigMap wraps the [`<host>/manifest.json`](../e2e), containing a JSON Patch file for the TDX or SNP bare-metal specifications for the configured host.
Add a file [`dev-docs/e2e/<host>/manifest.json`](../e2e) with the values for the runner you've added.
If the runner is using k3s and the embedded mirror registry, add a corresponding configuration file at `dev-docs/e2e/<host>/contrast-imagepuller.toml`.
Push the branch and run the `update_bm_tcb_specs` workflow on that branch.

These values will need to be updated after a host firmware upgrade or a TDX module change.
Updating a TDX host's firmware regenerates the platform's SGX provisioning keys, which changes the PIID, so `AllowedPIIDs` goes stale and the platform has to be re-registered with Intel before it can produce quotes again.
Replacing the TDX module changes `MrSeam`.

## Test run

First, prepare the missing Kubernetes resources by running the `bm_maintenance` workflow from `main`.
Then, start an appropriate e2e test for the platform (`openssl` or `gpu`).
