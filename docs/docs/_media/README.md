# Attestation graphics

Here is how you can modify and rebuild the attestation graphics.

Requirements:

* [inkscape](https://inkscape.org/)
* [goat](https://github.com/blampe/goat)

1. Edit the ASCII art
2. Render the SVG with goat: `goat -i <graphic>.txt -o <graphic>.svg
3. Export text to path with inkscape: `inkscape <graphic>.svg -o --export-plain-svg=attestation-pod.svg --export-text-to-path`
