package sdkhelper

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/crypto/ripemd160"
)

// ConvertAndEncode converts from a base64 encoded byte string to base32 encoded byte string and then to bech32.
func ConvertAndEncode(hrp string, data []byte) (string, error) {
	converted, err := ConvertBits(data, 8, 5, true)
	if err != nil {
		return "", fmt.Errorf("encoding bech32 failed: %w", err)
	}

	return Encode(hrp, converted)
}

// DecodeAndConvert decodes a bech32 encoded string and converts to base64 encoded bytes.
func DecodeAndConvert(bech string) (string, []byte, error) {
	hrp, data, err := Decode(bech, 1023)
	if err != nil {
		return "", nil, fmt.Errorf("decoding bech32 failed: %w", err)
	}

	converted, err := ConvertBits(data, 5, 8, false)
	if err != nil {
		return "", nil, fmt.Errorf("decoding bech32 failed: %w", err)
	}

	return hrp, converted, nil
}

const Secp256k1 = "/cosmos.crypto.secp256k1.PubKey"
const TendermintSecp256k1 = "tendermint/PubKeySecp256k1"
const Ed25519 = "/cosmos.crypto.ed25519.PubKey"
const Bn254 = "/cosmos.crypto.bn254.PubKey"

func MakeProposerAddress(keyType string, decodedPubkey []byte) (string, error) {
	switch keyType {
	case Secp256k1:
		return strings.ToUpper(hex.EncodeToString(sumTruncatedForSecp256k1(decodedPubkey))), nil
	case Ed25519:
		return strings.ToUpper(hex.EncodeToString(sumTruncated(decodedPubkey))), nil
	case Bn254:
		return strings.ToUpper(hex.EncodeToString(sumTruncated(decodedPubkey))), nil
	}
	// return "", and error unsupported key
	return "", errors.Errorf("unsupported key type: %s", keyType)
}

// ref; cosmos-sdk/crypto/keys/secp256k1
// ref; https://github1s.com/cosmos/cosmos-sdk/blob/main/crypto/keys/secp256k1/secp256k1.go#L170-L171
func sumTruncatedForSecp256k1(bz []byte) []byte {
	const TruncatedSize = 20
	sha := sha256.Sum256(bz)
	hasherRIPEMD160 := ripemd160.New()
	hasherRIPEMD160.Write(sha[:]) // does not error
	return hasherRIPEMD160.Sum(nil)[:TruncatedSize]
}

// ref; https://github1s.com/cosmos/cosmos-sdk/blob/main/crypto/keys/ed25519/ed25519.go#L166-L167
// ref; https://github.com/cometbft/cometbft/blob/main/crypto/tmhash/hash.go#L102
func sumTruncated(bz []byte) []byte {
	const TruncatedSize = 20
	hash := sha256.Sum256(bz)
	return hash[:TruncatedSize]
}

func MakeBLSPubkey(cometbftPubKey string) (string, error) {
	decodedKey, err := base64.StdEncoding.DecodeString(cometbftPubKey)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("0x%x", decodedKey), nil
}

func ProposerAddressFromPublicKey(pubKey string) (string, error) {
	decodedKey, err := base64.StdEncoding.DecodeString(pubKey)
	if err != nil {
		errors.Wrap(err, "failed to make a consensus key from staking validator public key")
	}

	return MakeProposerAddress(Ed25519, decodedKey)
}

func IsProposerAddress(str string) bool {
	for _, r := range str {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F') {
			return false
		}
	}
	return true
}
