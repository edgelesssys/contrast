# Recovery

## Problem

Restart currently causes loss of

- CA root & mesh key
- Active manifest
- Manifest & policy history

## Requirements

- State needs to be securely safed
- User needs secret for recovery
- Multi-party recovery (?)
- Keep manifest/policy history
- KMS also needs recovery (and all other services?)

# State

- Kubernetes-based state in CoCo
  - Initial Samba based implementation in Azure, but without security
  - Encrypted volumes needed anyway
- Kubernetes objects
  - Simpler, less initial investment

# Secret

- Reuse workload owner key
- Return encrypted recovery secret

# KMS recovery

# Distributed Coordinator updates

- Must be agreed upon in secret sharing mode
