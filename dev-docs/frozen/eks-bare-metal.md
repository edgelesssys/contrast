# Contrast on EKS

This is based on the setup presented in https://github.com/aws-samples/howto-runtime-attestation-on-aws.

## Resources

Replace the placeholder in eks-managed-sev-snp-metal-ubuntu-template.yaml with your SSH key name.

Create cluster and node group with

```
eksctl create cluster --without-nodegroup -f eks-cluster-template.yaml
eksctl create nodegroup -f eks-managed-sev-snp-metal-ubuntu-template.yaml
```

## Prevent autoscaling of the node

This is needed as the node needs to be rebooted.

```sh
aws autoscaling suspend-processes \
    --auto-scaling-group-name eksctl-raas-nodegroup-selfmanaged-NodeGroup-c2opXigZr6N9 \
    --scaling-processes ReplaceUnhealthy
```

## Setup the node

Get ssh access, then

```sh
sudo apt update
sudo apt upgrade -y
```

```bash
sudo apt install dracut -y
sudo tee -a /etc/dracut.conf.d/20-omit-ccp.conf <<EOF
omit_drivers+=" ccp "
EOF
sudo dracut --force
sudo tee -a /etc/modprobe.d/60-ccp.conf <<EOF
options ccp init_ex_path=/SEV_metadata
EOF
```

```bash
sudo sed 's/\(GRUB_CMD.*\)"/\1 mem_encrypt=on kvm_amd.sev=1 iommu=nopt"/' -i /etc/default/grub.d/50-cloudimg-settings.cfg
grep GRUB_CMD /etc/default/grub.d/50-cloudimg-settings.cfg # To validate
sudo update-grub
sudo grep sev /boot/grub/grub.cfg # Check
```

Reboot the node

```sh
sudo reboot
```

## Get the reference values

Access the node again via ssh, then

```sh
$ sh <(curl --proto '=https' --tlsv1.2 -L https://nixos.org/nix/install) --daemon
```

As root

```sh
nix-shell -p snphost
```

Reference values and check of host configuration via

```sh
modprobe msr
snphost ok
```

```sh
lscpu | grep "Model name\|CPU family"
```

CPU family = 25 → Milan
CPU family = 26 → Genoa

## Get the kubeconfig of the cluster

```sh
aws eks update-kubeconfig \
    --region eu-central-1 \
    --name raas \
    --kubeconfig kubeconf
export KUBECONFIG=$(realpath kubeconf)
```

## Make `ebs` default storage class

```sh
kubectl patch storageclass gp2 \
  -p '{"metadata":{"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'
```

## Configure the Contrast Manifest

- Remove the non-matching SNP processor family
- Update the TCB values with the output of `snphost ok`
- Set `TSMEEnabled` to `true`

## Update settings.json

Set the correct pause image

```json
 "cluster_config": {
        "pause_container_image": "public.ecr.aws/eks-distro/kubernetes/pause:3.5",
```

## Getting the coordinator service IP

Notice EKS isn't giving a LoadBalancer IP by default but a DNS name.
The instructions from the docs won't work to get it, describe the service and use the ingress field.
