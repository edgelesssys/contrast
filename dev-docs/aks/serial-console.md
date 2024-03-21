# Obtain a serial console inside the podvm

Get a shell on the AKS node. If in doubt, use [nsenter-node.sh](https://github.com/alexei-led/nsenter/blob/master/nsenter-node.sh).
Now run the following commands to use a debug [IGVM](https://docs.google.com/presentation/d/1uWeyqtYV53Vtxd3ayYWWTLbxasYNr35a/) and enable debugging for the Kata runtime.

```sh
sed -i -e 's#^igvm = "\(.*\)"#igvm = "/opt/confidential-containers/share/kata-containers/kata-containers-igvm-debug.img"#g' /opt/confidential-containers/share/defaults/kata-containers/configuration-clh-snp.toml
sed -i -e 's/^#enable_debug = true/enable_debug = true/g' /opt/confidential-containers/share/defaults/kata-containers/configuration-clh-snp.toml
systemctl restart containerd
```

Now you need to reconnect to the host. Use the following commands to print the sandbox ids of Kata VMs.
Please note that only the pause container of every pod has the `clh.sock`.  Other containers are part of the same VM.

```shell-session
$ ctr --namespace k8s.io container ls "runtime.name==io.containerd.kata-cc.v2"
$ sandbox_id=ENTER_SANDBOX_ID_HERE
```

And attach to the serial console using `socat`. You need to type `CONNECT 1026<ENTER>` to get a shell.

```shell-session
$ cd /var/run/vc/vm/${sandbox_id}/ && socat stdin unix-connect:clh.sock
CONNECT 1026
```

Alternatively, you can use `kata-runtime exec`, which gives you proper TTY behavior including tab completion.
First ensure the `kata-monitor` is running, then connect.

```
$ kata-monitor &
$ kata-runtime exec ${sandbox_id}
```

If you are done, use the following commands to go back to a release IGVM.

```sh
sed -i -e 's#^igvm = "\(.*\)"#igvm = "/opt/confidential-containers/share/kata-containers/kata-containers-igvm.img"#g' /opt/confidential-containers/share/defaults/kata-containers/configuration-clh-snp.toml
sed -i -e 's/^enable_debug = true/#enable_debug = true/g' /opt/confidential-containers/share/defaults/kata-containers/configuration-clh-snp.toml
systemctl restart containerd
```
