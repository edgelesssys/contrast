{
  writeShellApplication,
  scripts,
}:
writeShellApplication {
  name = "repro-b";
  runtimeInputs = [ scripts.repro-a ];
  text = ''
    repro-a
  '';
}
