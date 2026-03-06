# Obtain a serial console inside the pod VM

There are two way to obtain serial console: via `justfile.env` and via `withDebug` option.
Prefer the `justfile.env` way when ever possible.

To enable access via serial console, set `set=debug` in your `justfile.env`.
This is only working if you want to debug the `base` set.

For other sets, set `withDebug ? true` in `packages/by-name/contrast/node-installer-image/package.nix`.

Then, run `just`.

Get a shell on the Kubernetes node. If in doubt, use [nsenter-node.sh](https://github.com/alexei-led/nsenter/blob/master/nsenter-node.sh).

## Bare-Metal

Copy `packages/kata-debug-shell/kata-debug-shell.sh` to the host and run it, specifying the `<namespace/pod-name>` as the only argument.


```sh
./kata-debug-shell.sh openssl-suffix/coordinator-0
```
