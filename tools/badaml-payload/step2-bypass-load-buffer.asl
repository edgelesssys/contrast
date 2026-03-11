/*
    Step 2 bypass: Defeats static OperationRegion allowlist by using
    Load() to dynamically load AML from a buffer at runtime.

    The malicious OperationRegion is hidden inside a raw AML buffer
    (DBUF) which is loaded via Load(DBUF, HNDL) in _INI. The static
    scanner only sees opaque buffer data. At runtime, the ACPI
    interpreter loads the buffer as an SSDT.

    The inner SSDT uses a harmless Name() declaration so that the
    buffer bytes do not trigger the OperationRegion scanner. In a
    real attack, the inner SSDT would contain an OperationRegion
    targeting guest memory, but the static scanner cannot distinguish
    buffer data from AML instructions.

    This motivates Step 2's defense: blocking Load, LoadTable, and
    DataTableRegion opcodes entirely.
*/

DefinitionBlock ("", "SSDT", 2, "BADAML", "BYPASS02", 0x20250225)
{
    Scope (\_SB)
    {
        Device (EVIL)
        {
            Name (_HID, "MSFT0003")

            // Handle for the loaded table
            Name (HNDL, Zero)

            // Raw AML encoding of a minimal SSDT that just declares
            // a Name object. The key point is that Load() itself is
            // not blocked by the static scanner.
            //
            // Inner SSDT ASL equivalent:
            //   DefinitionBlock("","SSDT",2,"BADAML","LOADED",1) {
            //     Name(LDOK, "LOAD_BYPASS_OK")
            //   }
            Name (DBUF, Buffer()
            {
                // SSDT header (36 bytes)
                0x53, 0x53, 0x44, 0x54,   // "SSDT"
                0x39, 0x00, 0x00, 0x00,   // Length = 57
                0x02,                     // Revision
                0x00,                     // Checksum (to be fixed)
                0x42, 0x41, 0x44, 0x41, 0x4D, 0x4C,  // OemId "BADAML"
                0x4C, 0x4F, 0x41, 0x44, 0x45, 0x44,  // OemTableId
                0x20, 0x20,                           //   "LOADED  "
                0x01, 0x00, 0x00, 0x00,   // OemRevision
                0x49, 0x4E, 0x54, 0x4C,   // CreatorId "INTL"
                0x01, 0x00, 0x00, 0x00,   // CreatorRevision

                // AML body: Name(LDOK, "LOAD_BYPASS_OK")
                // NameOp
                0x08,
                // NameSeg "LDOK"
                0x4C, 0x44, 0x4F, 0x4B,
                // StringPrefix + string + null terminator
                0x0D,
                0x4C, 0x4F, 0x41, 0x44, 0x5F,  // "LOAD_"
                0x42, 0x59, 0x50, 0x41, 0x53,  // "BYPAS"
                0x53, 0x5F, 0x4F, 0x4B,        // "S_OK"
                0x00                           // null terminator
            })

            Method (_INI, 0, Serialized)
            {
                Debug = "BYPASS02: loading AML from buffer via Load()"
                Load (DBUF, HNDL)
                Debug = "BYPASS02: Load() completed"
            }
        }
    }
}
