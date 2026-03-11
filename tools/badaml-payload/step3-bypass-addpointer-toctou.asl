/*
    Step 3 bypass: Defeats static verification by using the QEMU
    table-loader ADD_POINTER mechanism to shift an OperationRegion
    address after verification.

    The OperationRegion is declared with address 0xFED00000 (HPET),
    which passes the allowlist check. However, the attacker controls
    the table-loader command script and includes an ADD_POINTER command
    that adds a delta to the address field after verification,
    redirecting it to arbitrary guest memory.

    This is a TOCTOU (time-of-check-time-of-use) attack: the address
    is valid at check time but modified before use.

    This motivates Step 3's defense: re-running VerifyBlobData after
    all table-loader commands have been processed.

    Note: This ASL file alone is not sufficient to mount this attack.
    It requires a crafted table-loader script with the ADD_POINTER
    command targeting the OperationRegion address field. The ASL is
    provided for documentation; the actual attack vector is the
    table-loader command sequence.
*/

DefinitionBlock ("", "SSDT", 6, "BADAML", "BYPASS03", 0x20250225)
{
    Scope (\_SB)
    {
        Device (EVIL)
        {
            Name (_HID, "MSFT0003")

            // Address passes allowlist check (HPET: 0xFED00000, 0x0400)
            // but ADD_POINTER in the table-loader script will shift it
            // to target guest memory at runtime.
            OperationRegion (HPTM, SystemMemory, 0xFED00000, 0x0400)
            Field (HPTM, AnyAcc, NoLock, Preserve)
            {
                TDAT, 64,
            }

            Method (_INI, 0, Serialized)
            {
                Debug = "BYPASS03: TOCTOU -- address was shifted by ADD_POINTER"
                Local0 = TDAT
                Debug = Local0
            }
        }
    }
}
