package types

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/ProtoconNet/mitum-currency/v3/common"
	"math/big"
	"strings"

	"crypto/ecdsa"
	"crypto/sha256"

	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
)

// MEPublickey is the optional public key of mitum, it is based on Ether Privatekey.
type MEPublickey struct {
	k *ecdsa.PublicKey
	s string
	b []byte
	hint.BaseHinter
}

func NewMEPublickey(k *ecdsa.PublicKey) MEPublickey {
	pub := MEPublickey{
		BaseHinter: hint.NewBaseHinter(MEPublickeyHint),
		k:          k,
	}

	return pub.ensure()
}

func ParseMEPublickey(s string) (MEPublickey, error) {
	t := MEPublickeyHint.Type().String()

	switch {
	case !strings.HasSuffix(s, t):
		return MEPublickey{}, util.ErrInvalid.Errorf("unknown public key string")
	case len(s) <= len(t):
		return MEPublickey{}, util.ErrInvalid.Errorf("invalid public key string; too short")
	}

	return LoadMEPublicKey(s[:len(s)-len(t)])
}

func LoadMEPublicKey(s string) (MEPublickey, error) {
	h, err := hex.DecodeString(s)
	if err != nil {
		return MEPublickey{}, util.ErrInvalid.WithMessage(err, "load public key")
	}
	pk, err := crypto.DecompressPubkey(h)
	if err != nil {
		return MEPublickey{}, util.ErrInvalid.WithMessage(err, "unmarshal public key")
	}

	return NewMEPublickey(pk), nil
}

func (k MEPublickey) String() string {
	return k.s
}

func (k MEPublickey) Bytes() []byte {
	return k.b
}

func (k MEPublickey) IsValid([]byte) error {
	if err := k.BaseHinter.IsValid(MEPublickeyHint.Type().Bytes()); err != nil {
		return util.ErrInvalid.WithMessage(err, "wrong hint in public key")
	}

	switch {
	case k.k == nil:
		return util.ErrInvalid.Errorf("empty public key in MEPublickey")
	case len(k.s) < 1:
		return util.ErrInvalid.Errorf("empty public key string")
	case len(k.b) < 1:
		return util.ErrInvalid.Errorf("empty public key []byte")
	}

	return nil
}

func (k MEPublickey) Equal(b base.PKKey) bool {
	switch {
	case b == nil:
		return false
	default:
		return k.s == b.String()
	}
}

func (k MEPublickey) Verify(input []byte, sig base.Signature) error {
	if len(sig) < 4 {
		return common.ErrValueInvalid.Wrap(base.ErrSignatureVerification.WithStack())
	}

	if len(sig) > 256 {
		return common.ErrValueInvalid.Wrap(base.ErrSignatureVerification.WithStack())
	}

	rlength := int(binary.LittleEndian.Uint32(sig[:4]))
	if (4 + rlength) > len(sig) {
		return common.ErrValueInvalid.Wrap(base.ErrSignatureVerification.WithStack())
	}

	r := big.NewInt(0).SetBytes(sig[4 : 4+rlength])
	s := big.NewInt(0).SetBytes(sig[4+rlength:])

	h := sha256.Sum256(input)
	if !ecdsa.Verify(k.k, h[:], r, s) {
		return base.ErrSignatureVerification.WithStack()
	}

	return nil
}

func (k MEPublickey) MarshalText() ([]byte, error) {
	return []byte(k.s), nil
}

func (k *MEPublickey) UnmarshalText(b []byte) error {
	u, err := LoadMEPublicKey(string(b))
	if err != nil {
		return errors.Wrap(err, "UnmarshalText for public key")
	}

	*k = u.ensure()

	return nil
}

func (k *MEPublickey) ensure() MEPublickey {
	if k.k == nil {
		return *k
	}

	k.s = fmt.Sprintf("%s%s", hex.EncodeToString(crypto.CompressPubkey(k.k)), k.Hint().Type().String())
	k.b = []byte(k.s)

	return *k
}
