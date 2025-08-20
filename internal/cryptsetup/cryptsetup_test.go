// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package cryptsetup

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const cryptsetupHeaderDump = `{
  "keyslots":{
    "0":{
      "type":"luks2",
      "key_size":64,
      "af":{
        "type":"luks1",
        "stripes":4000,
        "hash":"sha256"
      },
      "area":{
        "type":"raw",
        "offset":"32768",
        "size":"258048",
        "encryption":"aes-xts-plain64",
        "key_size":64
      },
      "kdf":{
        "type":"argon2id",
        "time":6,
        "memory":1048576,
        "cpus":4,
        "salt":"OtouKvOWb4Wxandy0LdhiG7bQiltnT78/eweb8WqiJo="
      }
    }
  },
  "tokens":{},
  "segments":{
    "0":{
      "type":"crypt",
      "offset":"16777216",
      "size":"dynamic",
      "iv_tweak":"0",
      "encryption":"aes-xts-plain64",
      "sector_size":4096
    }
  },
  "digests":{
    "0":{
      "type":"pbkdf2",
      "keyslots":[
        "0"
      ],
      "segments":[
        "0"
      ],
      "hash":"sha256",
      "iterations":673026,
      "salt":"C5XuT3zUZ4rMPHI/98HnYi+YQ2TDF4Onk79z+rdZYZE=",
      "digest":"gzHnbLzZpMe0tMMeP52UgBIjpQDgyP6J6y4St6sX+0M="
    }
  },
  "config":{
    "json_size":"12288",
    "keyslots_size":"16744448"
  }
}`

func TestVerifyHeader(t *testing.T) {
	var metadata cryptsetupMetadata
	if err := json.Unmarshal([]byte(cryptsetupHeaderDump), &metadata); err != nil {
		t.Fatalf("Failed to decode header JSON: %v", err)
	}

	// Verify the header
	if err := (&Device{}).verifyHeader(metadata); err != nil {
		t.Errorf("Header verification failed: %v", err)
	}
}

func TestVerifyBinaryHeader(t *testing.T) {
	testCases := map[string]struct {
		header  []byte
		wantErr bool
	}{
		"valid LUKS2 header": {
			header: []byte{
				0x4c, 0x55, 0x4b, 0x53, 0xba, 0xbe,
				0x00, 0x02, 0x00, 0x00, 0x00, 0x00,
			},
		},
		"unexpected secondary LUKS header magic": {
			header: []byte{
				0x53, 0x4b, 0x55, 0x4c, 0xba, 0xbe,
				0x00, 0x02, 0x00, 0x00, 0x00, 0x00,
			},
			wantErr: true,
		},
		"unexpected version": {
			header: []byte{
				0x4c, 0x55, 0x4b, 0x53, 0xba, 0xbe,
				0x00, 0x04, 0x00, 0x00, 0x00, 0x00,
			},
			wantErr: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			headerPath := t.TempDir() + "/header"
			require.NoError(os.WriteFile(headerPath, tc.header, 0o600))

			dev := &Device{headerPath: headerPath}
			err := dev.verifyBinaryHeader()
			if tc.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)
		})
	}
}
