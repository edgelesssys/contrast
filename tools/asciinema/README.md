# Screencast / Asciinema

[Asciinema](https://github.com/asciinema/asciinema) is used to automatically generate
terminal session recordings for our documentation. To fully automate this we use scripts
that utilize [expect](https://manpages.debian.org/testing/expect/expect.1.en.html) to interface with different
CLI tools, and run them inside a [container](docker/Dockerfile).

## Usage

```sh
./generate-screencasts.sh
```

This will:

+ build the container
+ run the expect based scripts
+ copy recordings into the recordings directory

To replay the output you can use `asciinema play recordings/flow.cast`.
