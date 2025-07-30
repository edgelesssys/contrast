# How to add a bare-metal instance to the CI

## Install Ubuntu LTS server

Download and install the latest Ubuntu LTS server from https://ubuntu.com/download/server.

When configuring the disk layout, ensure to use btrfs as the root filesystem.

## SNP setup

Creating AMD SEV-SNP guests via KVM is supported by kernels newer than 6.11 (see https://www.phoronix.com/news/Linux-6.11-KVM).
While we want to check occasionally if the latest version breaks
Contrast, we want to use the latest LTS version.

`mainline` is a tool to manage kernel installations on Ubuntu.
First install the `mainline` package:
```bash
sudo add-apt-repository ppa:cappelikan/ppa
sudo apt-get update
sudo apt-get install mainline pkexec
```
Now list all available kernel versions and install the latest LTS version. To
figure out what the latest long term version is, refer to https://kernel.org/.
```bash
mainline list
mainline install <latest LTS version e.g. 6.12.17>
```
Reboot the machine to boot automatically into the latest kernel and delete all old ones.
```bash
reboot
mainline uninstall-old
```

Check that SEV-SNP is enabled. If it's not then it likely needs to be
enabled in the BIOS. For those steps, either have a look in our docs
https://docs.edgeless.systems/contrast/howto/cluster-setup/bare-metal or
google for "enable AMD SEV in BIOS." Sadly, AMD changes their document
links from time to time, so we don't link it here.

Once it's enabled, verify this as follows:
```bash session
root@hetzner-ax162-snp ~ # journalctl -k -b 0 | grep -i sev
Feb 27 19:32:31 hetzner-ax162-snp kernel: SEV-SNP: RMP table physical range [0x0000000035500000 - 0x0000000075afffff]
Feb 27 19:32:31 hetzner-ax162-snp kernel: SEV-SNP: Reserving start/end of RMP table on a 2MB boundary [0x0000000035400000]
Feb 27 19:32:31 hetzner-ax162-snp kernel: SEV-SNP: Reserving start/end of RMP table on a 2MB boundary [0x0000000075a00000]
Feb 27 19:32:32 hetzner-ax162-snp kernel: ccp 0000:09:00.5: sev enabled
Feb 27 19:33:13 hetzner-ax162-snp kernel: ccp 0000:09:00.5: SEV API:1.55 build:32
Feb 27 19:33:13 hetzner-ax162-snp kernel: ccp 0000:09:00.5: SEV-SNP API:1.55 build:32
Feb 27 19:33:13 hetzner-ax162-snp kernel: kvm_amd: SEV enabled (ASIDs 100 - 1006)
Feb 27 19:33:13 hetzner-ax162-snp kernel: kvm_amd: SEV-ES enabled (ASIDs 1 - 99)
Feb 27 19:33:13 hetzner-ax162-snp kernel: kvm_amd: SEV-SNP enabled (ASIDs 1 - 99)
```

## TDX setup

Follow https://docs.edgeless.systems/contrast/howto/cluster-setup/bare-metal?vendor=intel#hardware-and-firmware-setup

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
echo \
  "deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu \
  $(. /etc/os-release && echo "${UBUNTU_CODENAME:-$VERSION_CODENAME}") stable" | \
  sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt-get update

# Download docker package
sudo apt-get install docker-ce docker-ce-cli containerd.io docker-buildx-plugin docker-compose-plugin
```

Install helm
```bash
curl -fsL https://get.helm.sh/helm-v3.17.1-linux-amd64.tar.gz | tar -C /tmp -xz linux-amd64/helm && mv /tmp/linux-amd64/helm /usr/local/bin
```

Add K3s configuration override
```bash
mkdir -p /etc/rancher/k3s
cat > /etc/rancher/k3s/config.yaml <<EOF
write-kubeconfig-mode: "0640"
write-kubeconfig-group: sudo
disable:
  - local-storage
kubelet-arg:
  - "runtime-request-timeout=5m"
EOF
```

Install K3s
```bash
curl -sfL https://get.k3s.io | INSTALL_K3S_VERSION=v1.31.5+k3s1 sh -
```
The K3s docs state:
> A kubeconfig file will be written to /etc/rancher/k3s/k3s.yaml and the kubectl installed by K3s will automatically use it.

Export the Kubeconfig for the current user for the following steps:
```bash
export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
```

Install Longhorn into K3s
```bash
helm repo add longhorn https://charts.longhorn.io
helm repo update
helm install longhorn longhorn/longhorn --namespace longhorn-system --create-namespace \
  --set defaultSettings.storageReservedPercentageForDefaultDisk=5 \
  --set persistence.defaultClassReplicaCount=1
```

## Kernel config

Follow https://docs.edgeless.systems/contrast/howto/cluster-setup/bare-metal?vendor=intel#kernel-setup.

For newer Ubuntu versions, also set

```
echo "kernel.apparmor_restrict_unprivileged_userns = 0" > /etc/sysctl.d/97-apparmor-allow-userns.conf
```

## Networking
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

## Add server as a GitHub runner

First, create another user, which the runner service will use.
```bash
useradd -s /bin/bash -m -G sudo,docker github
```

Put the K3s kubeconfig into the default dir for the user:
```bash
mkdir -p /home/github/.kube
ln -s /etc/rancher/k3s/k3s.yaml /home/github/.kube/config
```

The CI jobs build things with nix, therefore install it following the official instructions:
https://nixos.org/download/#nix-install-linux.

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
# Create file fs backend
echo "Setting up btrfs nix builder volume..."
sudo mkdir -p /mnt/nixbld
sudo truncate -s 20G /mnt/btrfs.img
sudo mkfs.btrfs -f /mnt/btrfs.img

# Create fstab entry to mount the file as btrfs
sudo echo -e "# btrfs for nix builder \n/mnt/btrfs.img /mnt/nixbld btrfs loop,defaults 0 0" > /etc/fstab
sudo mount -a

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
systemctl restart actions.runner.edgelesssys-contrast.hetzner-ax162-snp.service
```

Add the necessary tags for the runner in GitHub by navigating to
https://github.com/edgelesssys/contrast/settings/actions/runners
selecting the newly added runner and adding the labels the runner fulfills,
that's "tdx" for TDX servers and "snp" for SNP servers.

## Developer access
For developers to be able to access the K8s cluster, prepare
a kubeconfig which points to the DNS name of the server inside
the Tailscale:
```bash
CONFIG=$(cat /etc/rancher/k3s/k3s.yaml)
CONFIG="${CONFIG//default/$(hostname)$}"
CONFIG="${CONFIG//127.0.0.1/$(hostname)}"
echo "${CONFIG}" > $(hostname)-kubeconfig
```

Copy `hetzner-ax162-snp-kubeconfig` over to somewhere you are already
authenticated with GCP and push it as a secret. If the secret already
exists, only execute the `gcloud secrets versions add` command.
```bash
gcloud secrets create hetzner-ax162-snp-kubeconfig --replication-policy="automatic" --project constellation-331613
gcloud secrets versions add hetzner-ax162-snp-kubeconfig --data-file="./hetzner-ax162-snp-kubeconfig" --project constellation-331613
```

Add the secret to the secrets retrieved via `just` in
https://github.com/edgelesssys/contrast/blob/f14824f6c039e47a96cc0bbf2298bce5aa8e9844/justfile#L334

## Bare-metal runner specification
To run our e2e test with the real bare-metal runner specification, a ConfigMap named `bm-tcb-specs` is added to all e2e clusters.
Having the ConfigMap prevents using committed values in the e2e tests directly, which could otherwise lead to backporting problems.

The `bm-tcb-specs` ConfigMap wraps the [`tcb-specs.json`](../e2e/tcb-specs.json), sharing TDX and SNP bare-metal specifications.
While the ConfigMap stores both runner specifications the [patchReferenceValues()](https://github.com/edgelesssys/contrast/blob/main/e2e/internal/contrasttest/contrasttest.go#L254-L283) function will only use the platform-specific reference values for overwriting.

Add or update [`tcb-specs.json`](../e2e/tcb-specs.json) with the values from the runner you've added.
