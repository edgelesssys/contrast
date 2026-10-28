# Overview

This tutorial shows you how to make a Kubernetes deployment confidential using Contrast.
We'll start with from a non-confidential deployment of a simple application.

## Workflow

In this tutorial, we'll use the [emojivoto app](#emojivoto-app) as an example and walk through the steps needed to make a Kubernetes deployment confidential.
You can either follow along or apply the same steps to your own application.

To make your app confidential with Contrast, follow these steps:

1. [Install the Contrast CLI.](../howto/install-cli.md)
2. [Set up your Kubernetes cluster and install Contrast into it.](../howto/cluster-setup/bare-metal.md)
3. [Deploy your workload.](./deployment)
4. [Verify the integrity of your deployment.](./deployment.md#7-verify-deployment)
5. [Connect securely to your app.](./deployment.md#8-connect-securely-to-the-frontend)

## Emojivoto app

![Screenshot of the Emojivoto UI](../_media/emoijvoto.png)

We use the [emojivoto app](https://github.com/BuoyantIO/emojivoto) as example.
It's a microservice application where users vote for their favorite emoji, and votes are shown on a leaderboard.

The app includes:

- A web frontend (`web`)
- A gRPC backend that lists emojis (`emoji`)
- A gRPC backend for handling votes and leaderboard logic (`voting`)
- A `vote-bot` that simulates users by sending votes to the frontend

![Emojivoto components topology](../_media/contrast_emojivoto-topology.drawio.svg)

Emojivoto is a fun example, but it still handles data that could be considered sensitive.

Contrast protects emojivoto in two key ways:

1. It shields the entire app from the underlying infrastructure.
2. It can be configured to block data access even from the app's administrator.

In this setup, users can be sure that their votes stay private.
