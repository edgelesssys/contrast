// Copyright 2026 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package snp

import "github.com/edgelesssys/contrast/internal/atls/validators"

// TODO(burgerdev): there should be tests for the SNP validator!

// Ensure that Validator implements the intended interface.
var (
	_ validators.Validator = (*Validator)(nil)
	_ validators.Validator = (*IterativeValidator)(nil)
)
