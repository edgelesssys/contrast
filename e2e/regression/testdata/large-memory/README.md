# Pod with large memory request

There are several ways for the amount of memory assigned to a UVM to
influence runtime measurement. The memory size is reflected in ACPI
tables and SMBIOS entries, for example.

This manifested in a regression where on TDX, pods with more than 32Gi
of memory would end up in an attestation failure case. For more detail
on this, see [this PR](https://github.com/edgelesssys/contrast/pull/2170).

This test launches such a pod with a large memory request.
