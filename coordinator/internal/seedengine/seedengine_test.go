// Copyright 2024 Edgeless Systems GmbH
// SPDX-License-Identifier: AGPL-3.0-only

package seedengine

import (
	"crypto/x509"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSeedEngine_New(t *testing.T) {
	testCases := map[string]struct {
		secretSeed string
		salt       string

		wantPodStateSeed          string // hex encoded
		wantHistorySeed           string // hex encoded
		wantRootCAKey             string // DER, hex encoded
		wantTransactionSigningKey string // DER, hex encoded
		wantErr                   bool
	}{
		/*
			Crypto-determinism regression test cases.

			DO NOT CHANGE!
		*/
		"successful 1": {
			secretSeed:                "ccebed634ddee7535cd593e1e200b19b780f3906d8782207fa09c59e87a07cb3",
			salt:                      "8c1b1225c5f6cb7eef6dbd8f77a1e1e149de031d6e3718e660a8b04c8e2b0037",
			wantPodStateSeed:          "e8d42dc81aea4b0d0749b75004f3bb2ad35fd827e05727ac19c31e106ddc2a1f",
			wantHistorySeed:           "e0f4adb8326ed1bbf99b8291d7a90363113e2ac8ff9d030bcabe5e48b88bf0a6",
			wantRootCAKey:             "3081a40201010430b17a061a04d93454c9530d247b336a9112a5209da0d0199929484cba350d25159f2ead15d2d1334d6c908dad63ce7ee4a00706052b81040022a16403620004b30650a5c9b1653038ee779d0cef9da66f7207adf6b2a055ddbd13545734b4ababe5f1e6a062ba1694654f2b886fd6ec488ef7742af5cb8a9abd8823981c987d1868ce8708b29baea7963ae4428c7ea29c5d181006b2566dc21f34892c23d482",
			wantTransactionSigningKey: "3081a402010104305a58b771eef6bd6d2967b933ef3474e71bea849fd2b900f431dafe843d267b0b08875a95f4e442c6863663090c7c8576a00706052b81040022a16403620004f38a9990332aa58557780eff947e75c78a3486bbebce9f80d3e1f98f57b71ceaa207df91394d0eed25307d03ee460785db0afa958567089885e34ea693d861dfaa567fb34e6b3da51de25dfaf2a32aef01fb9d654f895712f1f4468281cd8ee9",
		},
		"successful 2": {
			secretSeed:                "1adb326866d5b1e04520d9475f6ff41d3370bec96bbb5045d8dd9d16b3c48274",
			salt:                      "0d2f1e0360d8476f836b8aa3dde3b1c58a361469a6e4f10cf9ed500a651c1c2b",
			wantPodStateSeed:          "8964f1750fa69bd4107ffb055f3e093cd93064590f6c3ad459b66c3bb19231fd",
			wantHistorySeed:           "03c95af2f666f44239a92d2cda3a14c3ad9ad776ef06fd97a8873457b9cff7f4",
			wantRootCAKey:             "3081a402010104303362d867cd3dfa7db3ca6fb3920aa4a5f198ac05bf2eb0190983c969a7e4f47d94c7bc061551800faaecbc321f541d20a00706052b81040022a16403620004308baf73fc4ff32dc16eaae1cf9354ee6d768b6f7636506225f05d2fada7c55beed8d7987c62815de952449359db6baf4e65b311f3c3f191fba8a17e938f4fe88423d96fc5c6edd54bcaea7e9a3047047160243e8ad1d7e0491145694c55b050",
			wantTransactionSigningKey: "3081a4020101043042ac864deb9da243469f13d6fae37576cfafbc9995bbd094b0e62873f4258099315ef7f2290f84e9163f44559f4a43a0a00706052b81040022a1640362000440d63497d354b85fd794575fe5916a581beeeee59bb63eb99dc3f44627af605040a9f50dfbd351e5b3c65f621e6fe5aedf2f170121201e5baf0e8b958bf8b0eacabe9e204bcd785fd73ae330c2ba8a321ce8aa2b73c52f606861250625cbcbe6",
		},
		"short salt": {
			secretSeed: "ccebed634ddee7535cd593e1e200b19b780f3906d8782207fa09c59e87a07cb3",
			salt:       "8c1b1225c5f6cb7eef6dbd8f77a1e1e149de031d6e3718e660a8b04c",
			wantErr:    true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			secret, err := hex.DecodeString(tc.secretSeed)
			require.NoError(err)
			salt, err := hex.DecodeString(tc.salt)
			require.NoError(err)

			se, err := New(secret, salt)

			if tc.wantErr {
				assert.Error(err)
				return
			}
			assert.NoError(err)
			assert.Equal(tc.wantPodStateSeed, hex.EncodeToString(se.podStateSeed))
			assert.Equal(tc.wantHistorySeed, hex.EncodeToString(se.historySeed))
			rootCAKey, err := x509.MarshalECPrivateKey(se.RootCAKey())
			require.NoError(err)
			assert.Equal(tc.wantRootCAKey, hex.EncodeToString(rootCAKey))
			transactionSigningKey, err := x509.MarshalECPrivateKey(se.TransactionSigningKey())
			require.NoError(err)
			assert.Equal(tc.wantTransactionSigningKey, hex.EncodeToString(transactionSigningKey))
		})
	}
}

func TestSeedEngine_DerivePodSecret(t *testing.T) {
	require := require.New(t)

	// policyHash -> want
	testCases := map[string]struct {
		podSecret string
		err       bool
	}{
		/*
			Crypto-determinism regression test cases.

			DO NOT CHANGE!
		*/
		"8d62644ef9944dbbb1a2b1a574840cbd6b09e5e7f96ac0f82a8a37271edd983b": {podSecret: "27a9ce52ad64f131d7e44c655d4ab0b0ab41b38a538615d2b28f88cbfeac2c70"},
		"b838a7adb60d110d6c3c7a1dfa51b439b78386f439a092eda0d67d53cc01c02e": {podSecret: "257172cbb64f1681f25168d46f361aa512c08c11c21ef6ad0b7d8b46ad29d443"},
		"11103d1efce19d05f5aaac2c8af405136ad91dae9f64ba25c2402100ff0e03eb": {podSecret: "425b229b7f327ca82ee39941cce26ea84e6a78aef3358c0c98b76515129dac32"},
		"d229c5714ca84d4e73b973636723e6cd5fe49f3c3e486732facfba61f94a10fc": {podSecret: "9e743b32c2fb0a9d791ba4cbd51445478d118ea88c4a0953576ed1ef4c1e353f"},
		"91b7513a7709d2ab92d2c1fe1e431e37f0bea18165dd908b0e6386817b0c6faf": {podSecret: "86343cf90cecf6a1582465d50c33a6ef38dea6ca95e1424dc0bca37d5c8e076f"},
		"99704c8b2a08ae9b8165a300ea05dbeae3b4c9a2096a6221aa4175bad43d53ec": {podSecret: "4006cbada495cb8f95e67f1b55466d63d94ca321789090bb80f01ae6c19ce8bf"},
		"f2e57529d3b92832eef960b75b2d299aadf1e373473bebf28cc105dae55c5f4e": {podSecret: "66d4fd6a3bfeac05490a29e6e3c4191cb2400a1949d3b4bc726a08d12415eeb5"},
		"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855": {err: true},
		"": {err: true},
	}

	secretSeed, err := hex.DecodeString("ccebed634ddee7535cd593e1e200b19b780f3906d8782207fa09c59e87a07cb3")
	require.NoError(err)
	salt, err := hex.DecodeString("8c1b1225c5f6cb7eef6dbd8f77a1e1e149de031d6e3718e660a8b04c8e2b0037")
	require.NoError(err)

	se, err := New(secretSeed, salt)
	require.NoError(err)

	for policyHashStr, want := range testCases {
		t.Run(policyHashStr, func(t *testing.T) {
			assert := assert.New(t)

			var policyHash [32]byte
			policyHashSlice, err := hex.DecodeString(policyHashStr)
			require.NoError(err)
			copy(policyHash[:], policyHashSlice)

			podSecret, err := se.DerivePodSecret(policyHash)

			if want.err {
				require.Error(err)
				return
			}
			assert.NoError(err)

			assert.Equal(want.podSecret, hex.EncodeToString(podSecret))
		})
	}
}
