# Nunki

Nunki ([/ˈnʌŋki/](https://en.wikipedia.org/wiki/Sigma_Sagittarii)) runs confidential container deployments
on untrusted Kubernetes at scale.

Nunki is based on the [Kata Containers](https://github.com/kata-containers/kata-containers) and
[Confidential Containers](https://github.com/confidential-containers) projects. Confidential Containers are Kubernetes pods that are executed inside a confidential micro-VM and provide strong hardware-based isolation
from the surrounding environment.

## Contributing

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

5. To build, containerize and deploy, run

    ```sh
    just
    ```

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
