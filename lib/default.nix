final: _prev: {
  by-name = pkgs: path:
    let
      entries = builtins.readDir path;
      filenames = builtins.attrNames entries;
      callPackage = final.callPackageWith (pkgs // self);
      maybeNameValuePair = filename:
        if entries.${filename} == "directory"
        then [
          {
            name = filename;
            value = callPackage (path + "/${filename}/package.nix") { };
          }
        ]
        else [ ];
      nameValuePairs = builtins.concatMap maybeNameValuePair filenames;
      self = builtins.listToAttrs nameValuePairs;
    in
    self;
}
