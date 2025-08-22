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
      "key_size":96,
      "af":{
        "type":"luks1",
        "stripes":4000,
        "hash":"sha256"
      },
      "area":{
        "type":"raw",
        "offset":"32768",
        "size":"385024",
        "encryption":"aes-xts-plain64",
        "key_size":64
      },
      "kdf":{
        "type":"argon2id",
        "time":463,
        "memory":10240,
        "cpus":1,
        "salt":"qHvCJjYEArUuKqqCv0iRRzORdl2BwWeF985bkumTyRA="
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
      "sector_size":512,
      "integrity":{
        "type":"hmac(sha256)",
        "journal_encryption":"none",
        "journal_integrity":"none"
      }
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
      "iterations":331408,
      "salt":"rhZsOc5mfml4jSWJd2u949PlTcOfNvcC28+qaWiProk=",
      "digest":"yua6YjYoX4SwBmQtNUNXwCfxqTNO9SPPkzeQYrtuapI="
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
