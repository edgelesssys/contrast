## Contributing

### Getting started

1. [Install Nix](https://zero-to-nix.com/concepts/nix-installer)
2. Enter the development environment with

    ```sh
    nix develop .#
    ```

   Or activate [direnv](https://direnv.net/) to automatically enter the nix shell.
   It is recommended to use [nix-direnv](https://github.com/nix-community/nix-direnv).
   If your system ships outdated bash, [install direnv](https://direnv.net/docs/installation.html) via package manager.
   Additionally, you may want to add the [vscode extension](https://github.com/direnv/direnv-vscode).

   ```sh
   direnv allow
   ```

3. Execute and follow instructions of

    ```sh
    just onboard
    ```

4. Provision a CoCo enabled AKS cluster with

    ```sh
    just create
    ```

    The kubeconfig of the cluster will be automatically downloaded and merged with your default config.
    You can get the kubeconfig of the running cluster at a later time with

    ```sh
    just get-credentials
    ```

### Deploy

5. To build, containerize, push and deploy, run

    ```sh
    just
    ```

    Ensure the pushed container images are accessible to your cluster.

6. Set the manifest after the Coordinator has started with

    ```sh
    just set
    ```

### Cleanup

7. Destroy the cluster with

    ```sh
    just destroy
    ```

### Maintenance tasks

- Run code generation

    ```sh
    just codegen
    ```

- Format all code

    ```sh
    nix fmt
    ```
