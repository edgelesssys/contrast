  DefinitionBlock ("", "SSDT", 6, "BADAML", "BADAML", 0x20240306)
  {
      Scope (\_SB)
      {
          Device (FAKE)
          {
              Name (_HID, "MSFT0003")

              Method (_INI, 0, Serialized)
              {
                  Debug = "BAD AML _INI STARTED"

                  /* Attempt a SystemMemory read - this should be blocked by the sandbox */
                  OperationRegion (TEST, SystemMemory, 0x10000000, 0x08)
                  Field (TEST, AnyAcc, NoLock, Preserve)
                  {
                      TDAT, 8
                  }
                  Local0 = TDAT

                  Debug = "BAD AML MEMORY ACCESS COMPLETED"
                  Debug = Local0
              }
          }
      }
  }
