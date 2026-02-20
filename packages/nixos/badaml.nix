{
  lib,
  config,
  pkgs,
  ...
}:

let
  cfg = config.contrast.badaml;

  # 64KB of repeated 0xDEADBEEF pattern (16384 repetitions).
  # We create a larger file so we can search the full initrd in less steps.
  deadbeef = pkgs.runCommandLocal "deadbeef.bin" { } ''
    for i in $(seq 1 16384); do
      printf '\xDE\xAD\xBE\xEF'
    done > "$out"
    chmod 0444 "$out"
  '';
in

{
  options.contrast.badaml = {
    enable = lib.mkEnableOption "Enable BadAML attack target";
  };

  config = lib.mkIf cfg.enable {
    # The attack modifies the initrd in memory. If the initrd is compressed,
    # it will be compressed in memory at the time the AML payload is executed.
    # To make the attack work with reasonable effort, we disable compression
    # and use an unencrypted initrd.
    boot.initrd.compressor = "cat";
    boot.initrd.systemd = {
      storePaths = [
        {
          # Place the target file in the initrd.
          source = deadbeef;
          target = "/deadbeef.bin";
        }
      ];
      # To easily check the attack was successful, we copy the target file to /run,
      # which is available in both initramfs and the final system.
      services.copy-deadbeef-into-sysroot = {
        description = "Expose deadbeef blob via /run";
        wantedBy = [ "initrd-root-fs.target" ];
        before = [ "initrd-switch-root.target" ];
        serviceConfig = {
          Type = "oneshot";
          ExecStart = "${pkgs.coreutils}/bin/install -D -m0644 /deadbeef.bin /run/deadbeef.bin";
        };
      };
    };
  };
}
