# Obtain a serial console inside the pod VM

Set `debugRuntime ? true` in `packages/by-name/contrast/node-installer-image/package.nix`.

Then, run `just`.

Get a shell on the Kubernetes node. If in doubt, use [nsenter-node.sh](https://github.com/alexei-led/nsenter/blob/master/nsenter-node.sh).

## Bare-Metal

Copy `packages/kata-debug-shell/kata-debug-shell.sh` to the host and run it, specifying the `<namespace/pod-name>` as the only argument.


```sh
./kata-debug-shell.sh openssl-suffix/coordinator-0
```
