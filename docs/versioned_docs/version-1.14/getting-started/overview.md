# Overview

This tutorial shows you how to make a Kubernetes deployment confidential using Contrast.
We'll start with from a non-confidential deployment of a simple application.

## Workflow

In this tutorial, we’ll use the [emojivoto app](#emojivoto-app) as an example and walk through the steps needed to make a Kubernetes deployment confidential.
You can either follow along or apply the same steps to your own application.

To make your app confidential with Contrast, follow these steps:

1. [Install the Contrast CLI](../howto/install-cli.md):
   Download and install the Contrast CLI, which is used to manage your deployments and generate necessary configurations.

2. [Install and set up](../howto/cluster-setup/bare-metal.md):
   Prepare your infrastructure and Kubernetes cluster for confidential computing use.

3. [Deploy your workload](./deployment):
   Update your application’s deployment to run with Contrast:

   1. **Adjust deployment files:** Modify your Kubernetes resources to integrate Contrast.
   2. **Deploy the Contrast runtime:** Run your workloads inside Confidential Virtual Machines (CVMs) by adding the Contrast runtime.
   3. **Add the Contrast Coordinator:** Include the Coordinator to verify and enforce the confidential and trusted state of your application.
   4. **Generate initdata annotations and manifest:** Use the Contrast CLI to generate a manifest that defines the expected secure state of your deployment.
   5. **Deploy your application:** Apply the updated deployment files to launch your app.
   6. **Set the manifest:** Define the trusted reference state that the Coordinator will enforce.

4. [Verify deployment](./deployment.md#7-verify-deployment):
   Confirm that your application is running securely and that workload integrity is being enforced.

5. [Securely connect to the app](./deployment.md#8-connect-securely-to-the-frontend):
   Establish a secure connection backed by confidential computing hardware—eliminating the need for users to place trust in you as the service provider.

## Emojivoto app

![Screenshot of the Emojivoto UI](../_media/emoijvoto.png)

We use the [emojivoto app](https://github.com/BuoyantIO/emojivoto) as example.
It's a microservice application where users vote for their favorite emoji, and votes are shown on a leaderboard.

The app includes:

- A web frontend (`web`)
- A gRPC backend that lists emojis (`emoji`)
- A gRPC backend for handling votes and leaderboard logic (`voting`)
- A `vote-bot` that simulates users by sending votes to the frontend

![Emojivoto components topology](https://raw.githubusercontent.com/BuoyantIO/emojivoto/e490d5789086e75933a474b22f9723fbfa0b29ba/assets/emojivoto-topology.png)

Emojivoto is a fun example, but it still handles data that could be considered sensitive.

Contrast protects emojivoto in two key ways:

1. It shields the entire app from the underlying infrastructure.
2. It can be configured to block data access even from the app's administrator.

In this setup, users can be sure that their votes stay private.
