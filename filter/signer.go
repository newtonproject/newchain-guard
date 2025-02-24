// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package filter

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/crypto/sha3"
)

var (
	ErrInvalidChainId      = errors.New("invalid chain id for signer")
	ErrNotEIP155           = errors.New("invalid signer type")
	ErrInvalidMsgLen       = errors.New("invalid message length, need 32 bytes")
	ErrInvalidSignatureLen = errors.New("invalid signature length")
	ErrInvalidRecoveryID   = errors.New("invalid signature recovery id")
	ErrInvalidKey          = errors.New("invalid private key")
)

var (
	secp256r1N     = elliptic.P256().Params().N
	secp256r1halfN = new(big.Int).Div(secp256r1N, big.NewInt(2))
)

// EIP155Transaction implements Signer using the EIP155 rules.
type EIP155Signer struct {
	chainId, chainIdMul *big.Int
}

func NewEIP155Signer(chainId *big.Int) EIP155Signer {
	if chainId == nil {
		chainId = new(big.Int)
	}
	return EIP155Signer{
		chainId:    chainId,
		chainIdMul: new(big.Int).Mul(chainId, big.NewInt(2)),
	}
}

var big8 = big.NewInt(8)

func (s EIP155Signer) Sender(tx *types.Transaction) (common.Address, error) {
	if !tx.Protected() {
		return common.Address{}, ErrNotEIP155
	}
	if tx.ChainId().Cmp(s.chainId) != 0 {
		return common.Address{}, ErrInvalidChainId
	}
	V, R, S := tx.RawSignatureValues()
	V = new(big.Int).Sub(V, s.chainIdMul)
	V.Sub(V, big8)
	return recoverPlain(s.Hash(tx), R, S, V, true)
}

func rlpHash(x interface{}) (h common.Hash) {
	hw := sha3.NewLegacyKeccak256()
	rlp.Encode(hw, x)
	hw.Sum(h[:0])
	return h
}

// Hash returns the hash to be signed by the sender.
// It does not uniquely identify the transaction.
func (s EIP155Signer) Hash(tx *types.Transaction) common.Hash {
	return rlpHash([]interface{}{
		tx.Nonce(),
		tx.GasPrice(),
		tx.Gas(),
		tx.To(),
		tx.Value(),
		tx.Data(),
		s.chainId, uint(0), uint(0),
	})
}

// ValidateSignatureValues verifies whether the signature values are valid with
// the given chain rules. The v value is assumed to be either 0 or 1.
func ValidateSignatureValues(v byte, r, s *big.Int, homestead bool) bool {
	if r.Cmp(common.Big1) < 0 || s.Cmp(common.Big1) < 0 {
		return false
	}
	// reject upper range of s values (ECDSA malleability)
	// see discussion in secp256k1/libsecp256k1/include/secp256k1.h
	if homestead && s.Cmp(secp256r1halfN) > 0 {
		return false
	}
	// Frontier: allow s to be in full N range
	return r.Cmp(secp256r1N) < 0 && s.Cmp(secp256r1N) < 0 && (v == 0 || v == 1)
}

// Keccak256 calculates and returns the Keccak256 hash of the input data.
func Keccak256(data ...[]byte) []byte {
	d := sha3.NewLegacyKeccak256()
	for _, b := range data {
		d.Write(b)
	}
	return d.Sum(nil)
}

func recoverPlain(sighash common.Hash, R, S, Vb *big.Int, homestead bool) (common.Address, error) {
	if Vb.BitLen() > 8 {
		return common.Address{}, types.ErrInvalidSig
	}
	V := byte(Vb.Uint64() - 27)
	if !ValidateSignatureValues(V, R, S, homestead) {
		return common.Address{}, types.ErrInvalidSig
	}
	// encode the snature in uncompressed format
	r, s := R.Bytes(), S.Bytes()
	sig := make([]byte, 65)
	copy(sig[32-len(r):32], r)
	copy(sig[64-len(s):64], s)
	sig[64] = V
	// recover the public key from the snature
	pub, err := Ecrecover(sighash[:], sig)
	if err != nil {
		return common.Address{}, err
	}
	if len(pub) == 0 || pub[0] != 4 {
		return common.Address{}, errors.New("invalid public key")
	}
	var addr common.Address
	copy(addr[:], Keccak256(pub[1:])[12:])
	return addr, nil
}

// Ecrecover returns the uncompressed public key that created the given signature.
func Ecrecover(hash, sig []byte) ([]byte, error) {
	if len(hash) != 32 {
		return nil, ErrInvalidMsgLen
	}
	if err := checkSignature(sig); err != nil {
		return nil, err
	}
	recId := int64(sig[len(sig)-1])
	pubKey, err := ecRecovery2(hash, sig[:len(sig)-1], recId)
	if pubKey == nil {
		return nil, err
	}
	bk := elliptic.Marshal(crypto.S256(), pubKey.X, pubKey.Y)
	return bk, nil

}

func checkSignature(sig []byte) error {
	if len(sig) != 65 {
		return ErrInvalidSignatureLen
	}
	if sig[64] >= 4 {
		return ErrInvalidRecoveryID
	}
	return nil
}

// DecompressPubkey parses a public key in the 33-byte compressed format.
func decompressPubkey2(x *big.Int, yBit byte) (*ecdsa.PublicKey, error) {
	if (yBit != 0x02) && (yBit != 0x03) {
		return nil, fmt.Errorf("invalid yBit")
	}
	if x == nil {
		return nil, fmt.Errorf("invalid x")
	}

	xx := new(big.Int).Mul(x, x)
	xxa := new(big.Int).Sub(xx, big.NewInt(3))
	yy := new(big.Int).Mul(xxa, x)
	yy.Add(yy, elliptic.P256().Params().B)
	yy.Mod(yy, elliptic.P256().Params().P)

	y1 := new(big.Int).ModSqrt(yy, elliptic.P256().Params().P)
	if y1 == nil {
		return nil, fmt.Errorf("can not revcovery public key")
	}

	getY2 := func(y1 *big.Int) *big.Int {
		y2 := new(big.Int).Neg(y1)
		y2.Mod(y2, elliptic.P256().Params().P)
		return y2
	}

	y := new(big.Int)

	if yBit == 0x02 {
		if y1.Bit(0) == 0 {
			y = y1
		} else {
			y = getY2(y1)
		}
	} else {
		if y1.Bit(0) == 1 {
			y = y1
		} else {
			y = getY2(y1)
		}
	}

	return &ecdsa.PublicKey{X: x, Y: y, Curve: elliptic.P256()}, nil
}
func ecRecovery2(messageHash []byte, sig []byte, recId int64) (*ecdsa.PublicKey, error) {
	if recId < 0 || recId > 3 {
		return nil, fmt.Errorf("invalid value of v")
	}

	sigLen := len(sig)
	r := new(big.Int).SetBytes(sig[:(sigLen / 2)])
	s := new(big.Int).SetBytes(sig[(sigLen / 2):])
	if r.Cmp(secp256r1N) > 0 {
		return nil, fmt.Errorf("r can not big then n")
	}
	if s.Cmp(secp256r1N) > 0 {
		return nil, fmt.Errorf("s can not big then half of n")
	}

	p256 := elliptic.P256()
	n := p256.Params().N
	i := new(big.Int).SetInt64(recId / 2)
	x := new(big.Int).Add(r, i.Mul(i, n))

	prime := p256.Params().P
	if x.Cmp(prime) > 0 {
		return nil, fmt.Errorf("x can not big then q")
	}
	yBit := byte(0x02)
	if recId%2 == 0 {
		yBit = 0x02
	} else {
		yBit = 0x03
	}
	R, err := decompressPubkey2(x, yBit)
	if err != nil {
		return nil, err
	}

	r1, r2 := p256.ScalarMult(R.X, R.Y, n.Bytes())
	zero := new(big.Int)
	if !((r1.Cmp(zero) == 0) && (r2.Cmp(zero) == 0)) {
		return nil, fmt.Errorf("nR != point at infinity")
	}

	e := new(big.Int).SetBytes(messageHash)
	eInv := new(big.Int).SetInt64(0)
	eInv.Sub(eInv, e)
	eInv.Mod(eInv, n)

	rInv := new(big.Int).Set(r)
	rInv.ModInverse(rInv, n)

	srInv := new(big.Int).Set(rInv)
	srInv.Mul(srInv, s)
	srInv.Mod(srInv, n)

	eInvrInv := new(big.Int).Mul(rInv, eInv)
	eInvrInv.Mod(eInvrInv, n)

	krx, kry := p256.ScalarMult(R.X, R.Y, srInv.Bytes())
	kgx, kgy := p256.ScalarBaseMult(eInvrInv.Bytes())
	kx, ky := p256.Add(krx, kry, kgx, kgy)
	if (kx.Cmp(zero) == 0) && (ky.Cmp(zero) == 0) {
		return nil, fmt.Errorf("public key can not be zero")
	}
	rkey := ecdsa.PublicKey{Curve: p256, X: kx, Y: ky}
	return &rkey, nil

}
