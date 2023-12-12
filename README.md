# Edgeless-CoCo

## Introduction

This project is based on the [Confidential Containers project](https://github.com/confidential-containers) and Azure's CoCo AKS offering.

## Problem

One can start singe confidential containers with the CoCo project. However, how can we secure and attest whole deployments with multiple microserivces and a singular web service (e.g., [Emojivoto](https://github.com/BuoyantIO/emojivoto))?

## Solution

A [MarbleRun](https://github.com/edgelesssys/marblerun) style service mesh for confidential deployments.
It includes a coordinator which is responsible for the attestation and certificate generation for new CoCos.
Moreover, it accepts an manifest which states what kind of deployment / images are allowed.

## How to use

1. [Install Nix](https://zero-to-nix.com/concepts/nix-installer)
1. Run `nix develop .#`
1. Run `just onboard` and fill out `justfile.env`
1. Run `just create`
1. Run `just`
1. Run `just init`

## Cleanup

1. Run `just destroy`
