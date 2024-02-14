{ writeShellApplication, yq-go }:

writeShellApplication {
  name = "kypatch";
  runtimeInputs = [ yq-go ];
  text = builtins.readFile ./kypatch.sh;
}
