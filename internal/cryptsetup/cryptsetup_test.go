// Copyright 2025 Edgeless Systems GmbH
// SPDX-License-Identifier: BUSL-1.1

package cryptsetup

import (
	"encoding/json"
	"testing"
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
