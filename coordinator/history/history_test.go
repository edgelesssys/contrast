// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package history

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"

	"github.com/edgelesssys/contrast/internal/testkeys"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHistory_GetLatestAndHasLatest(t *testing.T) {
	rq := require.New(t)

	testCases := map[string]struct {
		fsContent  map[string]string
		signingKey *ecdsa.PrivateKey
		// wants for GetLatest
		wantT   LatestTransition
		wantErr bool
		// wants for HasLatest
		wantHasLatest bool
	}{
		"success": {
			fsContent: map[string]string{
				"transitions/latest": fromHex(rq, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"+"304502210081e237315253991b496bdef5516527533a2bf828bae70a068be38ed612d5b90802207067b76f0a98e72282b276379e3b4d2857a37beea012c1bb3be9902cfc2d510c"),
			},
			signingKey: testkeys.New[ecdsa.PrivateKey](t, testkeys.ECDSAP256Keys[0]),
			wantT: LatestTransition{
				TransitionHash: strToHash(rq, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"),
				signature:      []byte(fromHex(rq, "304502210081e237315253991b496bdef5516527533a2bf828bae70a068be38ed612d5b90802207067b76f0a98e72282b276379e3b4d2857a37beea012c1bb3be9902cfc2d510c")),
			},
			wantHasLatest: true,
		},
		"hash modified": {
			fsContent: map[string]string{
				"transitions/latest": fromHex(rq, "3cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"+"304502210081e237315253991b496bdef5516527533a2bf828bae70a068be38ed612d5b90802207067b76f0a98e72282b276379e3b4d2857a37beea012c1bb3be9902cfc2d510c"),
			},
			signingKey:    testkeys.New[ecdsa.PrivateKey](t, testkeys.ECDSAP256Keys[0]),
			wantErr:       true,
			wantHasLatest: true,
		},
		"signature modified": {
			fsContent: map[string]string{
				"transitions/latest": fromHex(rq, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"+"404502210081e237315253991b496bdef5516527533a2bf828bae70a068be38ed612d5b90802207067b76f0a98e72282b276379e3b4d2857a37beea012c1bb3be9902cfc2d510c"),
			},
			signingKey:    testkeys.New[ecdsa.PrivateKey](t, testkeys.ECDSAP256Keys[0]),
			wantErr:       true,
			wantHasLatest: true,
		},
		"no latest": {
			fsContent:     map[string]string{},
			signingKey:    testkeys.New[ecdsa.PrivateKey](t, testkeys.ECDSAP256Keys[0]),
			wantErr:       true,
			wantHasLatest: false,
		},
		"no signature": {
			fsContent: map[string]string{
				"transitions/latest": fromHex(rq, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"),
			},
			signingKey:    testkeys.New[ecdsa.PrivateKey](t, testkeys.ECDSAP256Keys[0]),
			wantErr:       true,
			wantHasLatest: true,
		},
	}

	t.Run("GetLatest", func(t *testing.T) {
		for name, tc := range testCases {
			t.Run(name, func(t *testing.T) {
				require := require.New(t)
				assert := assert.New(t)

				fs := afero.Afero{Fs: afero.NewMemMapFs()}
				for path, content := range tc.fsContent {
					require.NoError(fs.WriteFile(path, []byte(content), 0o644))
				}

				h := &History{
					store:   NewAferoStore(&fs),
					hashFun: sha256.New,
				}

				gotT, err := h.GetLatest(&tc.signingKey.PublicKey)

				if tc.wantErr {
					require.Error(err)
					return
				}
				require.NoError(err)
				require.NotNil(gotT)
				assert.Equal(tc.wantT, *gotT)
			})
		}
	})

	t.Run("HasLatest", func(t *testing.T) {
		for name, tc := range testCases {
			t.Run(name, func(t *testing.T) {
				require := require.New(t)

				fs := afero.Afero{Fs: afero.NewMemMapFs()}
				for path, content := range tc.fsContent {
					require.NoError(fs.WriteFile(path, []byte(content), 0o644))
				}

				h := &History{
					store:   NewAferoStore(&fs),
					hashFun: sha256.New,
				}

				got, err := h.HasLatest()

				require.NoError(err)
				require.Equal(tc.wantHasLatest, got)
			})
		}
	})
}

func TestHistory_SetLatest(t *testing.T) {
	rq := require.New(t)
	testCases := map[string]struct {
		fsContent  map[string]string
		fsRo       bool
		signingKey *ecdsa.PrivateKey
		oldT       *LatestTransition
		newT       *LatestTransition
		wantErr    bool
	}{
		"success": {
			fsContent: map[string]string{
				"transitions/latest": fromHex(rq, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824") + "+sig",
			},
			signingKey: testkeys.New[ecdsa.PrivateKey](t, testkeys.ECDSAP256Keys[0]),
			oldT: &LatestTransition{
				TransitionHash: strToHash(rq, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"),
				signature:      []byte("+sig"),
			},
			newT: &LatestTransition{
				TransitionHash: strToHash(rq, "486ea46224d1bb4fb680f34f7c9ad96a8f24ec88be73ea8e5a6c65260e9cb8a7"),
			},
		},
		"initial transition": {
			fsContent:  map[string]string{},
			signingKey: testkeys.New[ecdsa.PrivateKey](t, testkeys.ECDSAP256Keys[0]),
			oldT:       nil,
			newT: &LatestTransition{
				TransitionHash: strToHash(rq, "486ea46224d1bb4fb680f34f7c9ad96a8f24ec88be73ea8e5a6c65260e9cb8a7"),
			},
		},
		"write error": {
			fsContent: map[string]string{
				"transitions/latest": fromHex(rq, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824") + "+sig",
			},
			signingKey: testkeys.New[ecdsa.PrivateKey](t, testkeys.ECDSAP256Keys[0]),
			oldT: &LatestTransition{
				TransitionHash: strToHash(rq, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"),
				signature:      []byte("+sig"),
			},
			newT: &LatestTransition{
				TransitionHash: strToHash(rq, "486ea46224d1bb4fb680f34f7c9ad96a8f24ec88be73ea8e5a6c65260e9cb8a7"),
			},
			fsRo:    true,
			wantErr: true,
		},
		"latest not existing": {
			fsContent:  map[string]string{},
			signingKey: testkeys.New[ecdsa.PrivateKey](t, testkeys.ECDSAP256Keys[0]),
			oldT: &LatestTransition{
				TransitionHash: strToHash(rq, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"),
				signature:      []byte("+sig"),
			},
			newT: &LatestTransition{
				TransitionHash: strToHash(rq, "486ea46224d1bb4fb680f34f7c9ad96a8f24ec88be73ea8e5a6c65260e9cb8a7"),
			},
			fsRo:    true,
			wantErr: true,
		},
		"latest updated": {
			fsContent: map[string]string{
				"transitions/latest": fromHex(rq, "c3ab8ff13720e8ad9047dd39466b3c8974e592c2fa383d4a3960714caef0c4f2") + "+sig",
			},
			signingKey: testkeys.New[ecdsa.PrivateKey](t, testkeys.ECDSAP256Keys[0]),
			oldT: &LatestTransition{
				TransitionHash: strToHash(rq, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"),
				signature:      []byte("+sig"),
			},
			newT: &LatestTransition{
				TransitionHash: strToHash(rq, "486ea46224d1bb4fb680f34f7c9ad96a8f24ec88be73ea8e5a6c65260e9cb8a7"),
			},
			fsRo:    true,
			wantErr: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			fs := afero.Afero{Fs: afero.NewMemMapFs()}
			for path, content := range tc.fsContent {
				require.NoError(fs.WriteFile(path, []byte(content), 0o644))
			}
			if tc.fsRo {
				fs = afero.Afero{Fs: afero.NewReadOnlyFs(fs.Fs)}
			}

			h := &History{
				store:   NewAferoStore(&fs),
				hashFun: sha256.New,
			}

			err := h.SetLatest(tc.oldT, tc.newT, tc.signingKey)

			if tc.wantErr {
				require.Error(err)
				return
			}
			require.NoError(err)
		})
	}
}

func TestHistory_GetTransition(t *testing.T) {
	rq := require.New(t)
	testCases := map[string]struct {
		fsContent      map[string]string
		hash           string
		wantTransition Transition
		wantErr        bool
	}{
		"success": {
			fsContent: map[string]string{
				"transitions/7305db9b2abccd706c256db3d97e5ff48d677cfe4d3a5904afb7da0e3950e1e2": fromHex(
					rq, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824486ea46224d1bb4fb680f34f7c9ad96a8f24ec88be73ea8e5a6c65260e9cb8a7"),
			},
			hash: "7305db9b2abccd706c256db3d97e5ff48d677cfe4d3a5904afb7da0e3950e1e2",
			wantTransition: Transition{
				ManifestHash: strToHash(
					rq, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"),
				PreviousTransitionHash: strToHash(
					rq, "486ea46224d1bb4fb680f34f7c9ad96a8f24ec88be73ea8e5a6c65260e9cb8a7"),
			},
		},
		"not found": {
			fsContent: map[string]string{},
			hash:      "7305db9b2abccd706c256db3d97e5ff48d677cfe4d3a5904afb7da0e3950e1e2",
			wantErr:   true,
		},
		"unmarshal error": {
			fsContent: map[string]string{
				"transitions/2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824": "hello",
			},
			hash:    "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
			wantErr: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			fs := afero.Afero{Fs: afero.NewMemMapFs()}
			for path, content := range tc.fsContent {
				require.NoError(fs.WriteFile(path, []byte(content), 0o644))
			}

			h := &History{
				store:   NewAferoStore(&fs),
				hashFun: sha256.New,
			}

			gotTransition, err := h.GetTransition(strToHash(require, tc.hash))

			if tc.wantErr {
				require.Error(err)
				t.Log(err)
				return
			}
			require.NoError(err)
			require.NotNil(gotTransition)
			assert.Equal(tc.wantTransition, *gotTransition)
		})
	}
}

func TestHistory_SetTransition(t *testing.T) {
	testCases := map[string]struct {
		fsContent     map[string]string
		fsRo          bool
		transition    Transition
		wantHash      string
		wantFSContent map[string]string
		wantErr       bool
	}{
		"success": {
			fsContent: map[string]string{},
			transition: Transition{
				ManifestHash:           strToHash(require.New(t), "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"),
				PreviousTransitionHash: strToHash(require.New(t), "486ea46224d1bb4fb680f34f7c9ad96a8f24ec88be73ea8e5a6c65260e9cb8a7"),
			},
			wantHash: "7305db9b2abccd706c256db3d97e5ff48d677cfe4d3a5904afb7da0e3950e1e2",
			wantFSContent: map[string]string{
				"transitions/7305db9b2abccd706c256db3d97e5ff48d677cfe4d3a5904afb7da0e3950e1e2": "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824486ea46224d1bb4fb680f34f7c9ad96a8f24ec88be73ea8e5a6c65260e9cb8a7",
			},
		},
		"object exists": {
			fsContent: map[string]string{
				"transitions/7305db9b2abccd706c256db3d97e5ff48d677cfe4d3a5904afb7da0e3950e1e2": "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824486ea46224d1bb4fb680f34f7c9ad96a8f24ec88be73ea8e5a6c65260e9cb8a7",
			},
			transition: Transition{
				ManifestHash:           strToHash(require.New(t), "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"),
				PreviousTransitionHash: strToHash(require.New(t), "486ea46224d1bb4fb680f34f7c9ad96a8f24ec88be73ea8e5a6c65260e9cb8a7"),
			},
			wantHash: "7305db9b2abccd706c256db3d97e5ff48d677cfe4d3a5904afb7da0e3950e1e2",
			wantFSContent: map[string]string{
				"transitions/7305db9b2abccd706c256db3d97e5ff48d677cfe4d3a5904afb7da0e3950e1e2": "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824486ea46224d1bb4fb680f34f7c9ad96a8f24ec88be73ea8e5a6c65260e9cb8a7",
			},
		},
		"write error": {
			fsContent: map[string]string{},
			fsRo:      true,
			transition: Transition{
				ManifestHash:           strToHash(require.New(t), "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"),
				PreviousTransitionHash: strToHash(require.New(t), "486ea46224d1bb4fb680f34f7c9ad96a8f24ec88be73ea8e5a6c65260e9cb8a7"),
			},
			wantErr: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			fs := afero.Afero{Fs: afero.NewMemMapFs()}
			for path, content := range tc.fsContent {
				require.NoError(fs.WriteFile(path, []byte(content), 0o644))
			}
			if tc.fsRo {
				fs = afero.Afero{Fs: afero.NewReadOnlyFs(fs.Fs)}
			}

			h := &History{
				store:   NewAferoStore(&fs),
				hashFun: sha256.New,
			}

			gotHash, err := h.SetTransition(&tc.transition)

			if tc.wantErr {
				require.Error(err)
				return
			}
			require.NoError(err)

			gotFSContent := map[string]string{}
			require.NoError(fs.Walk("", func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					return nil
				}
				if info.Mode().IsRegular() {
					content, err := fs.ReadFile(path)
					require.NoError(err)
					gotFSContent[path] = hex.EncodeToString(content)
				}
				return nil
			}))
			assert.Equal(tc.wantHash, hex.EncodeToString(gotHash[:]))
			assert.Equal(tc.wantFSContent, gotFSContent)
		})
	}
}

func TestHistory_getCA(t *testing.T) {
	testCases := map[string]struct {
		fsContent map[string]string
		hash      string
		wantBytes string
		wantErr   bool
	}{
		"success": {
			fsContent: map[string]string{
				"tests/2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824": "hello",
				"tests/486ea46224d1bb4fb680f34f7c9ad96a8f24ec88be73ea8e5a6c65260e9cb8a7": "world",
			},
			hash:      "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
			wantBytes: "hello",
		},
		"not found": {
			fsContent: map[string]string{
				"tests/486ea46224d1bb4fb680f34f7c9ad96a8f24ec88be73ea8e5a6c65260e9cb8a7": "world",
			},
			hash:    "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
			wantErr: true,
		},
		"hash mismatch": {
			fsContent: map[string]string{
				"tests/2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824": "hello!",
				"tests/486ea46224d1bb4fb680f34f7c9ad96a8f24ec88be73ea8e5a6c65260e9cb8a7": "world",
			},
			hash:    "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
			wantErr: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			fs := afero.Afero{Fs: afero.NewMemMapFs()}
			for path, content := range tc.fsContent {
				require.NoError(fs.WriteFile(path, []byte(content), 0o644))
			}

			h := &History{
				store:   NewAferoStore(&fs),
				hashFun: sha256.New,
			}

			hash := strToHash(require, tc.hash)

			gotBytes, err := h.getContentaddressed("tests/%s", hash)

			if tc.wantErr {
				require.Error(err)
				return
			}
			require.NoError(err)
			require.Equal(tc.wantBytes, string(gotBytes))
		})
	}
}

func TestHistory_setCA(t *testing.T) {
	testCases := map[string]struct {
		fsContent     map[string]string
		fsRo          bool
		pathFmt       string
		data          string
		wantHash      string
		wantFSContent map[string]string
		wantErr       bool
	}{
		"success hello": {
			fsContent: map[string]string{},
			pathFmt:   "tests/%s",
			data:      "hello",
			wantHash:  "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824",
			wantFSContent: map[string]string{
				"tests/2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824": "hello",
			},
		},
		"success world": {
			fsContent: map[string]string{
				"tests/2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824": "hello",
			},
			pathFmt:  "tests/%s",
			data:     "world",
			wantHash: "486ea46224d1bb4fb680f34f7c9ad96a8f24ec88be73ea8e5a6c65260e9cb8a7",
			wantFSContent: map[string]string{
				"tests/2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824": "hello",
				"tests/486ea46224d1bb4fb680f34f7c9ad96a8f24ec88be73ea8e5a6c65260e9cb8a7": "world",
			},
		},
		"write error": {
			fsContent: map[string]string{},
			pathFmt:   "tests/%s",
			data:      "hello",
			fsRo:      true,
			wantErr:   true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			assert := assert.New(t)

			fs := afero.Afero{Fs: afero.NewMemMapFs()}
			for path, content := range tc.fsContent {
				require.NoError(fs.WriteFile(path, []byte(content), 0o644))
			}
			if tc.fsRo {
				fs = afero.Afero{Fs: afero.NewReadOnlyFs(fs.Fs)}
			}

			h := &History{
				store:   NewAferoStore(&fs),
				hashFun: sha256.New,
			}

			gotHash, err := h.setContentaddressed(tc.pathFmt, []byte(tc.data))

			if tc.wantErr {
				require.Error(err)
				return
			}
			require.NoError(err)

			gotFSContent := map[string]string{}
			require.NoError(fs.Walk("", func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					return nil
				}
				if info.Mode().IsRegular() {
					content, err := fs.ReadFile(path)
					require.NoError(err)
					gotFSContent[path] = string(content)
				}
				return nil
			}))
			assert.Equal(tc.wantFSContent, gotFSContent)
			assert.Equal(tc.wantHash, hex.EncodeToString(gotHash[:]))
		})
	}
}

func TestHistory_SetGet(t *testing.T) {
	h := &History{
		store:   &AferoStore{fs: &afero.Afero{Fs: afero.NewMemMapFs()}},
		hashFun: sha256.New,
	}

	testCases := []string{
		"hello",
		"world",
		"Nun ich verkündige dir, merk auf, und höre die Worte!" +
			"Denke nach: wird uns Athene und Vater Kronion" +
			"Gnügen; oder ist's nötig, noch andere Hilfe zu suchen?",
	}
	testFunPairs := map[string]struct {
		setFun func([]byte) ([HashSize]byte, error)
		getFun func([HashSize]byte) ([]byte, error)
	}{
		"manifest": {h.SetManifest, h.GetManifest},
		"policy":   {h.SetPolicy, h.GetPolicy},
	}

	for name, tc := range testFunPairs {
		for _, data := range testCases {
			t.Run(name+"_"+data[:5], func(t *testing.T) {
				require := require.New(t)

				hash, err := tc.setFun([]byte(data))
				require.NoError(err)

				gotData, err := tc.getFun(hash)
				require.NoError(err)
				require.Equal(data, string(gotData))
			})
		}
	}
}

func strToHash(require *require.Assertions, s string) [HashSize]byte {
	hashSlc, err := hex.DecodeString(s)
	require.NoError(err)
	require.Len(hashSlc, HashSize)
	var hash [HashSize]byte
	copy(hash[:], hashSlc)
	return hash
}

func fromHex(require *require.Assertions, s string) string {
	data, err := hex.DecodeString(s)
	require.NoError(err)
	return string(data)
}
