// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

//go:build !enterprise

package main

import (
	"context"

	"golang.org/x/sync/errgroup"
)

func registerEnterpriseServices(context.Context, *errgroup.Group, *components) {}
