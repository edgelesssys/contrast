# Obtain a serial console inside the podvm

Set `debug ? true` in `packages/{kata,microsoft}/runtime-class-files/package.nix` and run `just`.

Get a shell on the AKS node. If in doubt, use [nsenter-node.sh](https://github.com/alexei-led/nsenter/blob/master/nsenter-node.sh).

Use the following commands to print the sandbox ids of Kata VMs.
Please note that only the pause container of every pod has the `clh.sock`.  Other containers are part of the same VM.

Set the name of the pod you want to access:

```sh
podName=<coordinator-0>
```

Run the following command to get the sandbox ID:

```sh
sandbox_id=$(crictl pods -o json | jq -r ".items[] | select(.metadata.name == \"${podName}\" and .state == \"SANDBOX_READY\") | .id")
```

You can use `kata-runtime exec`, which gives you proper TTY behavior including tab completion.
First ensure the `kata-monitor` is running, then connect.

```sh
kata-monitor &
kata-runtime exec ${sandbox_id}
```

You might need to point `kata-runtime` to the config, for example

```sh
kata-runtime --config /opt/edgeless/contrast-cc-31695b720e385ba6ecbc4db97ae8ce28/etc/configuration-clh-snp.toml exec ${sandbox_id}
```

Alternatively, you can attach to the serial console using `socat`. You need to type `CONNECT 1026<ENTER>` to get a shell.

```sh
(cd /var/run/vc/vm/${sandbox_id}/ && socat stdin unix-connect:clh.sock)
CONNECT 1026
```
