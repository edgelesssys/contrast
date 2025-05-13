## Contributing

Thank you for getting involved! Before you start, please familiarize yourself with the [documentation](https://docs.edgeless.systems/contrast).

Please follow our [Code of Conduct](CODE_OF_CONDUCT.md) when interacting with this project.

### Opening an issue, starting a discussion, asking a question

This project uses the GitHub issue tracker. Check the existing issues before submitting to avoid duplicates.

If you have a broader topic to discuss or a question, please [open a discussion](https://github.com/edgelesssys/contrast/discussions) instead.

### Pull requests

Contrast is licensed under the [AGPLv3](LICENSE.AGPL), with some parts (`enterprise` directory) being licensed under [BUSL](enterprise/LICENSE).
When contributing, you also need to agree to our [Contributor License Agreement](https://cla-assistant.io/edgelesssys/contrast).

PRs that aren't labeled `no changelog` need a PR description that's meaningful to the end user, as they will be included in the release notes.
All other impacts imply that the PR has a visible impact on the end user.

### Git conventions

Commit history on PRs should be clean and meaningful.
Before submitting a PR, make sure to cleanup you history, for example using interactive rebase.

Most commits will use a prefix to indicate the area of change: `<prefix>: <title>`.
Common titles are the top-level directory your change was made in (for example `docs`), a subdirectory (`e2e/reproducibility`), or the nix attribute (path) that was changed (`kata.kata-runtime`).
Further, there are some meta-prefixes like `ci` and `treewide` used.
Use a meaningful commit title that's precise, short and descriptive for your change.
You can find many guides and examples online.
Commit message bodies aren't required, but recommended on PRs with many non-trivial commits.

Wherever possible, each commit should be functional (it should build, have tests and CI passing).
It's okay to deviate from this rule in situations where a non-functional commit improves review-ability a lot.
In general, commit history is to document the change and ease the review process.
Make sure file moves are detected by git, do them in a separate commit where needed.
If you are renaming things and it causes changes in many lines, do it in a separate commit.

When addressing review feedback, be sure to it cleanly with regard to your existing commits.
For smaller PRs, you can directly amend the commit and force-push the update (remember to always use `--force-with-lease`).
GitHub will have a button to show the diff of your last force-push.
Push a rebase separate from addressed changes.
If you need to push more often, this method might fail.
In this case, and on larger PRs (with many commits, reviewers or changes to address), you should use [fixup commits](https://blog.sebastian-daschner.com/entries/git-commit-fixup-autosquash) instead.
Make sure to target the right commit with the fixup commit.
After the PR has been approved, you can then rebase and `--autosquash` your fixup commits.
In some situations, tools like [git-absorb](https://github.com/tummychow/git-absorb) might come in handy (but beware of fixup commit flooding).
If you are struggling with your git history during the review process, please ask a more experienced contributor for help.

### Development setup

1. [Install Nix](https://zero-to-nix.com/concepts/nix-installer)

2. (Optional) configure Nix to allow use of extra substituters, and profit from our
    cachix remote cache. To allow using additional substituters from the `flake.nix`,
    add your user name (or the wheel group) as trusted-user in your nix config.

    On NixOS (in your config):

    ```nix
    nix.settings.trusted-users = [ "root" "@wheel" ];
    ```

    On other systems (in `/etc/nix/nix.conf`):

    ```
    trusted-users = root @wheel
    ```

    See Nix manual section on [substituters](https://nixos.org/manual/nix/stable/command-ref/conf-file.html#conf-substituters)
    and [trusted-users](https://nixos.org/manual/nix/stable/command-ref/conf-file.html#conf-trusted-users) for details and
    consequences.

    When first using a Nix command on the Contrast flake, accept the additional substituters permanently when prompted.

3. Enter the development environment with

    ```sh
    nix develop .#
    ```

   Or activate [`direnv`](https://direnv.net/) to automatically enter the nix shell.
   It's recommended to use [`nix-direnv`](https://github.com/nix-community/nix-direnv).
   If your system ships outdated bash (MacOS), [install `direnv`](https://direnv.net/docs/installation.html) via package manager.

   ```sh
   direnv allow
   ```

   Additionally, you may want to add the [VSCode extension](https://github.com/direnv/direnv-vscode).

4. Execute and follow instructions of

    ```sh
    just onboard
    ```

5. Provision a CoCo enabled AKS cluster with

    ```sh
    just create
    ```

    The kubeconfig of the cluster will be automatically downloaded and merged with your default config.
    You can get the kubeconfig of the running cluster at a later time with

    ```sh
    just get-credentials
    ```

### Deploy

The usual developer flow is available as a single target to execute:

```sh
just [default <deployment-name>]
```

This will build, containerize and push all relevant components.
Ensure the pushed container images are accessible to your cluster.
The manifest will the be generated (`contrast generate`).

Further the flow will deploy the selected deployment and wait for components to come up.
The manifest will automatically be set (`contrast set`) and the Coordinator will be verified
(`contrast verify`). The flow will also wait for the workload to get ready.

This target is idempotent and will delete an existing deployment before re-deploying.

All steps can be executed as separate targets. To list all available targets and their description, run

```sh
just --list
```

### Cleanup

- Destroy the cluster with

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
    just fmt
    ```
