// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package mount

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseBlkidCommand(t *testing.T) {
	blk := &blk{
		DevName:   "/dev/nvme1n1p1",
		UUID:      "f3e74cbb-4b6c-442b-8cd1-d12e3fc7adcd",
		BlockSize: 1,
		Type:      "ntfs",
	}

	blocks := map[string]struct {
		outFormatString string
		expError        bool
	}{
		"valid output": {
			outFormatString: "DEVNAME=%s\nUUID=%s\nBLOCK_SIZE=%d\nTYPE=%s\n",
			expError:        false,
		},
		"no equal sign": {
			outFormatString: "No-Equal-Sign\n",
			expError:        true,
		},
		"block size is string": {
			outFormatString: "BLOCK_SIZE=isString\n",
			expError:        true,
		},
	}

	for name, block := range blocks {
		t.Run(name, func(t *testing.T) {
			parsedBlock, err := parseBlkidCommand(func(formatString string) []byte {
				return []byte(fmt.Sprintf(formatString,
					blk.DevName, blk.UUID, blk.BlockSize, blk.Type))
			}(block.outFormatString))

			if block.expError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, blk, parsedBlock)

			}
		})
	}
}
