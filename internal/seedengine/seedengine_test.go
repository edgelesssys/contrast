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
			wantRootCAKey:             "3081a402010104301048b5c79c516eb73020c3c19d0e5e75dcfa5523cdd795aeabcb88a210802c76208fe3bae0f7da7c7f63f5e8953358fea00706052b81040022a16403620004872f0b9edfaa1da5c0769dcf06b718496870e738adbe8d857068c23747c3b80c3a190d744b9789ebf92661dbb13ac25a48e07aed117ca21c2b8430e551be584176a0e921b43423f2abb745e8e38463a613a09fa74cee74639c27ce20cbc70c9d",
			wantTransactionSigningKey: "3081a402010104301a97756a0006510d5563eed012ceff6985f09388a518f1ce7ace37ef8a326895331e6b0f3eeed254c75d4a2d07bde6a1a00706052b81040022a16403620004f9a879010c3a4f463320865e18cdce3987e45af8cf9448fbd0bf1746509a66ace3191fbd806c07f8cd5369a87cf6e91b2b62d24da3bb13ab1daef9abd59d4dc5415873be265a6f9e729a748c9414963423a33060c8dcfd27d88108a4ed7c6539",
		},
		"successful 2": {
			secretSeed:                "1adb326866d5b1e04520d9475f6ff41d3370bec96bbb5045d8dd9d16b3c48274",
			salt:                      "0d2f1e0360d8476f836b8aa3dde3b1c58a361469a6e4f10cf9ed500a651c1c2b",
			wantPodStateSeed:          "8964f1750fa69bd4107ffb055f3e093cd93064590f6c3ad459b66c3bb19231fd",
			wantHistorySeed:           "03c95af2f666f44239a92d2cda3a14c3ad9ad776ef06fd97a8873457b9cff7f4",
			wantRootCAKey:             "3081a40201010430719e8620f2ed8b5c757c7ce8680985720b83b20c272af2f058d396dd7ae6dc76c9913fd3465bcdcae6f0d3e6384db413a00706052b81040022a16403620004b454b6147af047e3f341f0f60cc664809566d86a78cd1862871daed5234490b20410f957fe60d321acc43550600dd648e005382af6e5e9549daee1b50dc82fc44358bcfb6d72fa9b050f95b96c5a8b70a6250a307dbe43c66b322518ae268bc0",
			wantTransactionSigningKey: "3081a402010104305748d4dd5f101bfae8ff6489ebf6390de68bfc5267b8deb49d485dffad83c18f51808ebb93b27b2b9487a04a28b2e255a00706052b81040022a164036200047ffa141c50ff48223f7802e02731585754fcca44746dff5ef8df19339f108295a2bb3b42de2eb50cab599e594138abb15679bdcb8a6d3fbef0c372fc6306cffdd33f3321393be8a6f1a6989c8d5755aafd6d5a6c49058f9af8dd0beca9141c14",
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

func TestSeedEngine_DeriveMeshCAKey(t *testing.T) {
	req := require.New(t)

	// transactionHash -> want
	testCases := map[string]struct {
		meshCAKey string // hex(x509.MarshalECPrivateKey(meshCAKey))
		err       bool
	}{
		/*
			Crypto-determinism regression test cases.

			DO NOT CHANGE!
		*/
		"8d62644ef9944dbbb1a2b1a574840cbd6b09e5e7f96ac0f82a8a37271edd983b": {meshCAKey: "3081a40201010430f1247e2b66958faf9aa524a94816da813b18a035f15daf4d35cbbb60643e97b72df91a9135ea2fab7724979e913a3422a00706052b81040022a16403620004593ff9c7ecdefe21b428416e0ae81784b954b331faf3de9612b504db1e13305b834a1126fbd6fa7073081cb3b92f6464628fc51de3baf7b78037e02b1371259fa7633bc3b99588a16f24111c56f8f6db41369619204857b7d1416534c77f3c47"},
		"b838a7adb60d110d6c3c7a1dfa51b439b78386f439a092eda0d67d53cc01c02e": {meshCAKey: "3081a40201010430390804190ef6e01e34fff963a50850cd5d8345d092c413deb86c32dcdd14a1e2c7dac4317139e94842b97b335104150aa00706052b81040022a16403620004eb8541e09d18cb96c49dd66d2585597ae3f185d125ac6ff78371213d96636d74ced679370d63723eebbf1f98bb23e3cead1535cac7d595f5ff3b480dd53c7146e409956474fa4c20dc5956082d1c843343e65b827e7c91172ca5143250575591"},
		"11103d1efce19d05f5aaac2c8af405136ad91dae9f64ba25c2402100ff0e03eb": {meshCAKey: "3081a40201010430d89ede876f64c8c683fbb953ec57a841bb2c2f59a3863f0c325855da40f6da0a3babe7e64d925553d03d11c4d8caa8a4a00706052b81040022a164036200045ab49a846c2b742b85415fdfebd08272bf6e32bd1a0c605cf4b1cd25dbcce4c17f847c6dd27235a4da548134c505694048a6ecbf3864815dafca842edef5451440c103f3143e538bd5aa2a9bd8897a82923ac06a24dee3f7cb25187befbab8b8"},
		"d229c5714ca84d4e73b973636723e6cd5fe49f3c3e486732facfba61f94a10fc": {meshCAKey: "3081a402010104304b9320746578895fe917b55040fbfb65a806c9ca7757ff06d467a5eb9e9eec3f5a6ea5dda90734f5effa7e1ec14f4d12a00706052b81040022a164036200042e11ae0fc85bfde3a762f597d7bcafad8122749fc4c8ac614925b5c0060a2520541ff7c5c96d7c5bcb8784254f299e99fb263daf9204905f59cff7d94265b36f3b613e3625e5443e90d54b8d401e15b1a06bea5f6595891a23f2631633cd8bc3"},
		"91b7513a7709d2ab92d2c1fe1e431e37f0bea18165dd908b0e6386817b0c6faf": {meshCAKey: "3081a40201010430d7dc79626cbdb350d9bd7ea95f1461dc2ebf6f06d2cff9b501e4e802dd7a2ab30bb6298b509063b3237737da45564a95a00706052b81040022a164036200047831f43068f9b4acf553a19983a15efff4cc928a4be56be2e239eecbd9446eb6aacd8ac6e13a4fc1957636a9586c26b2f8b5cc92653bc148cc30d0e5f0c46351ccdb2972802af849b5d7d24785dfde1e881a907358c6e285f3593334362730de"},
		"99704c8b2a08ae9b8165a300ea05dbeae3b4c9a2096a6221aa4175bad43d53ec": {meshCAKey: "3081a40201010430f08b066f918684cb97e33449143327b651c206a8473c3d789c3091bb4c677afca0dc24d78df527523d345a75a9be4fc4a00706052b81040022a164036200041b84754f8963621c77ad7dba9a452869e96407e6646eb028ab11f46d07cac4341fb6762d500c96b6a480250e11f3ba45d664c88382d242be39632270f540dcf2f98cc66f46d0495aa713cfebb8123483e3e7667c3461f4cd65c2f8c84abbe384"},
		"f2e57529d3b92832eef960b75b2d299aadf1e373473bebf28cc105dae55c5f4e": {meshCAKey: "3081a4020101043095e15f05598731a2f1e9b54b3c3e6c914fbf532be808eaa65010fb9a6d5dc18de9f1678216c82d7113e4b9f93e74cc97a00706052b81040022a16403620004259d18d11b0fa7cb30ca190f5ba637f5c400346c95084d515d313ebe40cd699b08d0a3bf0f4b1494229e5ce5d2347be90c9af51c131f627b1789da925ff2221da3e0cee4e34460fcfc3748e976fc9f608b404e7bb52de456e8d79e4fec4f2042"},
		"e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855": {err: true},
		"": {err: true},
	}

	secretSeed, err := hex.DecodeString("ccebed634ddee7535cd593e1e200b19b780f3906d8782207fa09c59e87a07cb3")
	req.NoError(err)
	salt, err := hex.DecodeString("8c1b1225c5f6cb7eef6dbd8f77a1e1e149de031d6e3718e660a8b04c8e2b0037")
	req.NoError(err)

	se, err := New(secretSeed, salt)
	req.NoError(err)

	for transactionHashStr, want := range testCases {
		t.Run(transactionHashStr, func(t *testing.T) {
			require := require.New(t)

			var transactionHash [32]byte
			transactionHashSlice, err := hex.DecodeString(transactionHashStr)
			require.NoError(err)
			copy(transactionHash[:], transactionHashSlice)

			meshCAKey, err := se.DeriveMeshCAKey(transactionHash)

			if want.err {
				require.Error(err)
				return
			}
			require.NoError(err)

			meshCAKeyDER, err := x509.MarshalECPrivateKey(meshCAKey)
			require.NoError(err)
			require.Equal(want.meshCAKey, hex.EncodeToString(meshCAKeyDER))
		})
	}
}
