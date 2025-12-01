# Debugging TDX measurement mismatches

> [!CAUTION]
> This document doesn't claim correctness by any mean and shouldn't be seen
> as a source of truth. It should only serve as a helper document for internal
> debugging.

TDX uses a static MRTD and dynamic RTMRs for it's (boot) integrity measurements.

We pre-calculate expected values that we later check against when verifying workloads.

This document shows how mismatches in these measurements can be debugged.

## Retrieving the guest's event log

[Get a shell](../serial-console.md) into the pod VM.
Then, run the `tdeventlog` binary that's included in the root filesystem (see `tdx-tools` package).

<details>
<summary>Unfold in case you don't have a debug build</summary>

You can try to run `tdeventlog` from a container instead:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: tdeventlog
spec:
  template:
    metadata:
      annotations:
        "io.katacontainers.config.hypervisor.cc_init_data": H4sIAAAAAAAAA42UTU/DMAyG7/0VVS+7TYAEByQOsI0PibGqHeyAEPIar41I45I4g/LrSamEOJB2h16cp47f13b2aKwkHV/EydH0eHqURKBKMpKruovZCk5Oz5IoehbA8BIlDSlZtFODJSUemEwmUQPFG5QY+0/zaw9EkcAdOMXxpRCXWfqAsqy2ZGyG7w4tx+cXMRuHf7H8A5rQ8UyRxZyF1EGCmvZaKgyeGwTGGWkGqdEMYzlosaXPEDT3QUPtCLX4xCI1VKANqr5BXiIbWQwRq9VysffWBpEuOkevSwXT3EvLd5rR7MDXM0Rl5DhMLLG+JW6UK69ar20btHullbd5lj76P0JMCs6OdyR1St3VfrhCQIYgcvZ9C16UYU378Zt6LGdQ+CQNS9rZvAKDS3Kag5ZkaBFF5meB6jnuBzBXj9eQI/f99FO4lnVQtOfSn00LArLUoEbmz2s1PF5TR60NFFKXAwzbAzJRM5Joze2m23Irv4LiHxtxyC732KKpsEYDariNPfy7I8PY8JJsQPKI8Rv/wuL/Q9s9qd8znczNlgUAAA==
    spec:
      runtimeClassName: contrast-cc-metal-qemu-tdx-4c6bca70 # TODO: changeme
      containers:
      - name: main
        image: ghcr.io/burgerdev/tdeventlog:latest@sha256:dbeda9c95686d0f239a923b8fa3d316d075716fef142ed0d8a4eaa3a93bbcd68
        resources:
          requests:
            memory: 5000Mi
          limits:
            memory: 5000Mi
      restartPolicy: Never
```

</details>

If you can't get the tool running inside the guest,
the `/sys/firmware/acpi/tables/data/CCEL` and `/sys/firmware/acpi/tables/CCEL` files can also be dumped
by other means and transferred to a machine where they can then be parsed with `tdeventlog`.

## Understanding the event log

The event log will consist of multiple entries looking like so:

```
==== TDX Event Log Entry - 17 [0x83AA76BC] ====
RTMR              : 2
Type              : 0x6 (EV_EVENT_TAG)
Length            : 87
Algorithms ID     : 12 (TPM_ALG_SHA384)
Digest[0] : efa84d42b931a7454dc770eeeca0d476ac613f432b650515fc26cff088cf206c856c276f8acf435e98560c14fd2e0c67
RAW DATA: ----------------------------------------------
83AA76BC  03 00 00 00 06 00 00 00 01 00 00 00 0C 00 EF A8  ................
83AA76CC  4D 42 B9 31 A7 45 4D C7 70 EE EC A0 D4 76 AC 61  MB.1.EM.p....v.a
83AA76DC  3F 43 2B 65 05 15 FC 26 CF F0 88 CF 20 6C 85 6C  ?C+e...&.... l.l
83AA76EC  27 6F 8A CF 43 5E 98 56 0C 14 FD 2E 0C 67 15 00  'o..C^.V.....g..
83AA76FC  00 00 EC 22 3B 8F 0D 00 00 00 4C 69 6E 75 78 20  ...";.....Linux
83AA770C  69 6E 69 74 72 64 00                             initrd.
RAW DATA: ----------------------------------------------
```

`RTMR` specifies the RTMR (out of `RTMR {0,1,2,3}`) the measurement has been made into. Thus,
if you want to debug only a specific register, it makes sense to `grep` for this line.
While the `Type` might be of value to see what component actually makes the measurement, it will be
considered out-of-scope for this document. `Length` and `Algorithms ID` should be self-explanatory.

`Digest[0]` is the SHA384 of the raw measured contents. In the above example, `efa84d...` corresponds to
`sha384sum initrd.zst`.

`RAW DATA` is the raw data blob for the measurement event, containing the aforementioned information as
well as the informational string (`Linux initrd`, in this case) associated with the event. Note that this
can be misleading, as for some events measured by OVMF, the informational string is actually equal to the
measured data (the input for `sha384sum`) - however, this isn't the case for all measurements.

## Locating RTMR mismatches

Usually, the error given by the coordinator, CLI, etc. will already show you which RTMR mismatched.

To narrow it down further, it's recommended to add debug statements to the [`hashAndExtend`](https://github.com/edgelesssys/contrast/blob/a73691e17492b37469e32c7e800c4c0f7a955545/tools/tdx-measure/rtmr/rtmr.go#L45)
function of the measurement precalculator to see a log corresponding to the `Digest[0]` values in the
event log. Then, one can diff these against the digests in the event log for the RTMR in question to see
which event causes the mismatch.

Finding the mismatch then is a matter of code search and reversing which component might have done which
measurement.

The [TDX Virtual Firmware documentation](https://cdrdv2.intel.com/v1/dl/getContent/733585) gives an abstract
overview of what components of the boot chain are generally reflected in the specific registers, but this is
likely not sufficient to find the exact location where things go wrong.

GitHub code search against the informational string of the event seems to be a good general pathway to find
out about what the measurement is exactly.

Below is an incomplete list of which component measures into which RTMRs:

### RTMR 0

Measured into by the Firmware(?).

Contains the firmware itself, secure boot EFI variables, ACPI configuration and the `EFI_LOAD_OPTION`
passed by the VMM.

For events that have `QEMU FW CFG` in their data, you can enable the corresponding sysfs interface in the kernel:

```
CONFIG_FW_CFG_SYSFS=y
```

Then inspect `/sys/firmware/fw_cfg` for the actual values used by QEMU.

### RTMR 1

Measured into by the Firmware.

Contains a measurement of the loaded EFI application (the kernel, for example), and raw hashes of the aforementioned
informational strings.

### RTMR 2

Measured into by GRUB or the Linux [EFI stub](https://elixir.bootlin.com/linux/v6.11.8/source/drivers/firmware/efi/libstub/efi-stub-helper.c),
as it would do with TPM PCR 8/9.

This contains a measurement for the kernel command line and the initrd, in that order.

### RTMR 3

Reserved, all-0 at the moment of writing.

## MRTD

The MRTD is a special case, since it's a static measurement. It can only be populated from the TDX module *while*
a TD (TDX VM) is being created. That is, in the time span between VM creation and VM launch.

The writing process of the MRTD in a just-*created* TD can be thought of as follows:

- For every section in the TDVF, QEMU issues a `KVM_TDX_INIT_MEM_REGION` ioctl [^1] to the kernel's KVM module [^2].
- For every page / chunk in the section, the kernel issues `TDH_MEM_PAGE_ADD` and `TDH_MR_EXTEND` SEAM calls to the
  TDX module, respectively [^3].
- The TDX module handles these SEAM calls by extending writing the received values to the SHA384 context of the MRTD
  register. [^4]

This is the logic that [`tdx-measure`](../../tools/tdx-measure/) replicates for pre- calculation of the MRTD values.

The `TDH_MEM_PAGE_ADD` calls are used for every section of the TDVF, but only write the GPA (guest physical address)
of the page to the MRTD, not any actual content.
The `TDH_MR_EXTEND` calls are only made for sections marked with a specific bit, but measure the actual page contents
into the MRTD too.

### Locating MRTD mismatches

Since debugging or re-building the TDX module isn't supported by Intel, the best way to debug mismatching MRTD values
is to trace the kernel bindings for the TDX module functions that influence the MRTD.

This `bpftrace` command can be used on the TDX host to generate an "event log" for the MRTD in `mrtd.log`:

```sh
bpftrace -e '
kprobe:tdh_mng_init { printf("TDH_MNG_INIT\n"); }
kprobe:tdh_mem_page_add { printf("TDH_MEM_PAGE_ADD gpa=0x%llx\n", arg1); }
kprobe:tdh_mr_extend { printf("TDH_MR_EXTEND gpa=0x%llx\n", arg1); }
kprobe:tdh_mr_finalize { printf("TDH_MR_FINALIZE\n"); }
' > mrtd.log
```

Note that this, to keep the event log at a manageable size, omits the actual data measured in the `TDH_MR_EXTEND` calls.
To debug a mismatch there, it's best to compare the full-on TDVF blob.

The generated event log has the exact same format as the one produced by the `tdx-measure` tool in the `eventlogs/mrtd.log`
directory of its build output, allowing for easy comparison.

[^1]: <https://github.com/qemu/qemu/blob/de074358e99b8eb5076d3efa267e44c292c90e3e/target/i386/kvm/tdx.c#L359>
[^2]: <https://github.com/torvalds/linux/blob/ac3fd01e4c1efce8f2c054cdeb2ddd2fc0fb150d/arch/x86/kvm/vmx/tdx.c#L3324>
[^3]: <https://github.com/torvalds/linux/blob/ac3fd01e4c1efce8f2c054cdeb2ddd2fc0fb150d/arch/x86/kvm/vmx/tdx.c#L3209-L3228>
[^4]: <https://github.com/intel/confidential-computing.tdx.tdx-module/blob/887ef77fe8a010811226d81f4da14e9b1329e744/src/vmm_dispatcher/api_calls/tdh_mem_page_add.c#L219>,
      <https://github.com/intel/confidential-computing.tdx.tdx-module/blob/887ef77fe8a010811226d81f4da14e9b1329e744/src/vmm_dispatcher/api_calls/tdh_mr_extend.c#L170-L185>
