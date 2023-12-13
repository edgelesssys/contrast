# Constellation Confidential Containers Coordinator

## Developing

### Getting started

1. [Install Nix](https://zero-to-nix.com/concepts/nix-installer)
1. Enter the development environment.
    ```
    nix develop .#
    ```
2. Execute and follow instructions of
    ```sh
    just onboard
    ```
3. Provision a CoCo enables AKS cluster with
    ```sh
    just create
    ```

### Deploy

5. To build, containerize and deploy, run
    ```sh
    just
    ```
36. Set the manifest after the Coordinator has started with
    ```sh
    just init
    ```

### Cleanup

7. Destroy the cluster with
    ```sh
    just destroy
    ```
