# Constellation Confidential Containers Coordinator

## Developing

### Getting started

1. [Install Nix](https://zero-to-nix.com/concepts/nix-installer)
2. Enter the development environment.

    ```sh
    nix develop .#
    ```

3. Execute and follow instructions of

    ```sh
    just onboard
    ```

4. Provision a CoCo enables AKS cluster with

    ```sh
    just create
    ```

### Deploy

1. To build, containerize and deploy, run

    ```sh
    just
    ```

2. Set the manifest after the Coordinator has started with

    ```sh
    just init
    ```

### Cleanup

1. Destroy the cluster with

    ```sh
    just destroy
    ```
