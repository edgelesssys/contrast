final: prev: {
  contrastPkgs = prev.contrastPkgs.overrideScope (
    _final: prev: {
      scripts = prev.scripts.overrideScope (
        _final: prev: {
          repro-a = prev.repro-a.overrideAttrs (old: {
            text = ''
              #!/bin/bash
              echo "overridden A"
            '';
          });
        }
      );
    }
  );
}
