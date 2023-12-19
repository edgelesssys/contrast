# Nunki

Nunki ([/ˈnʌŋki/](https://en.wikipedia.org/wiki/Sigma_Sagittarii)) runs confidential container deployments
on untrusted Kubernetes at scale.

Nunki is based on the [Kata Containers](https://github.com/kata-containers/kata-containers) and
[Confidential Containers](https://github.com/confidential-containers) projects. Confidential Containers are
Kubernetes pods that are executed inside a confidential micro-VM and provide strong hardware-based isolation
from the surrounding environment. This works with unmodified containers in a lift-and-shift approach.

## The Nunki Coordinator

The Nunki Coordinator is the central remote attestation component of a Nunki deployment. It's a certificate
authority and issues certificates for workload pods running inside confidential containers. The Coordinator
is configured with a *manifest*, a configuration file that holds the reference values of all other parts of
a deployment. The Coordinator ensures that your app's topology adheres to your specified manifest. It verifies
the identity and integrity of all your services and establishes secure, encrypted communication channels between
the different parts of your deployment. As your app needs to scale, the Coordinator transparently verifies new
instances and then provides them with mesh credentials.

To verify your deployment, the remote attestation of the Coordinator and its manifest offers a single remote
attestation statement for your entire deployment. Anyone can use this to verify the integrity of your distributed
app, making it easier to assure stakeholders of your app's security.

## The Nunki Initializer

Nunki provides an Initializer that handles the remote attestation on the workload side transparently and
fetches the workload certificate. The Initializer runs as init container before your workload is started.

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
