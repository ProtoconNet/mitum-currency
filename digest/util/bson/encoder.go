package bsonenc

import (
	"encoding"
	"io"
	"reflect"

	"github.com/ProtoconNet/mitum2/util"
	"github.com/ProtoconNet/mitum2/util/encoder"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/pkg/errors"
	"go.mongodb.org/mongo-driver/bson"
)

var BSONEncoderHint = hint.MustNewHint("bson-encoder-v2.0.0")

type Encoder struct {
	decoders *hint.CompatibleSet[encoder.DecodeDetail]
	pool     util.GCache[string, any]
}

func NewEncoder() *Encoder {
	return &Encoder{
		decoders: hint.NewCompatibleSet[encoder.DecodeDetail](1 << 10),
	}
}

func (*Encoder) Hint() hint.Hint {
	return BSONEncoderHint
}

func (enc *Encoder) SetPool(pool util.GCache[string, any]) *Encoder {
	enc.pool = pool

	return nil
}

func (enc *Encoder) Add(d encoder.DecodeDetail) error {
	if err := d.IsValid(nil); err != nil {
		return util.ErrInvalid.WithMessage(err, "add in bson encoder")
	}

	x := d
	if x.Decode == nil {
		x = enc.analyze(d, d.Instance)
	}

	return enc.addDecodeDetail(x)
}

func (enc *Encoder) AddHinter(hr hint.Hinter) error {
	if err := hr.Hint().IsValid(nil); err != nil {
		return util.ErrInvalid.WithMessage(err, "add in bson encoder")
	}

	return enc.addDecodeDetail(enc.analyze(encoder.DecodeDetail{Hint: hr.Hint()}, hr))
}

func (*Encoder) Marshal(v interface{}) ([]byte, error) {
	return bson.Marshal(v)
}

func (*Encoder) Unmarshal(b []byte, v interface{}) error {
	return bson.Unmarshal(b, v)
}

func (*Encoder) StreamEncoder(w io.Writer) util.StreamEncoder {
	return nil
}

func (*Encoder) StreamDecoder(r io.Reader) util.StreamDecoder {
	return nil
}

func (enc *Encoder) Decode(b []byte) (interface{}, error) {
	if isNil(b) {
		return nil, nil
	}

	ht, err := enc.guessHint(b)
	if err != nil {
		return nil, errors.WithMessage(err, "guess hint in bson decoders")
	}

	return enc.decodeWithHint(b, ht)
}

func (enc *Encoder) DecodeWithHint(b []byte, ht hint.Hint) (interface{}, error) {
	if isNil(b) {
		return nil, nil
	}

	return enc.decodeWithHint(b, ht.String())
}

func (enc *Encoder) DecodeWithHintType(b []byte, t hint.Type) (interface{}, error) {
	if isNil(b) {
		return nil, nil
	}

	ht, v, found := enc.decoders.FindBytType(t)
	if !found {
		return nil, errors.Errorf("Find decoder by type in bson decoders, %q", t)
	}

	i, err := v.Decode(b, ht)
	if err != nil {
		return nil, errors.WithMessagef(err, "Decode, %q in bson decoders", ht)
	}

	return i, nil
}

func (enc *Encoder) DecodeWithFixedHintType(s string, size int) (interface{}, error) {
	if len(s) < 1 {
		return nil, nil
	}

	e := util.StringError("Decode with fixed hint type")

	if size < 1 {
		return nil, e.Errorf("Size < 1")
	}

	i, found := enc.poolGet(s)
	if found {
		if i != nil {
			err, ok := i.(error)
			if ok {
				return nil, e.Wrap(err)
			}
		}

		return i, nil
	}

	i, err := enc.decodeWithFixedHintType(s, size)
	if err != nil {
		enc.poolSet(s, err)

		return nil, e.Wrap(err)
	}

	enc.poolSet(s, i)

	return i, nil
}

func (enc *Encoder) decodeWithFixedHintType(s string, size int) (interface{}, error) {
	e := util.StringError("Decode with fixed hint type")

	body, t, err := hint.ParseFixedTypedString(s, size)
	if err != nil {
		return nil, e.WithMessage(err, "Parse fixed typed string")
	} else if _, _, found := enc.decoders.FindBytType(t); !found {
		return nil, e.Errorf("Find decoder by fixed typed hint type, %q in bson decoders", t)
	}

	i, err := enc.DecodeWithHintType([]byte(body), t)
	if err != nil {
		return nil, e.WithMessage(err, "Decode with hint type")
	}

	return i, nil
}

func (enc *Encoder) DecodeSlice(b []byte) ([]interface{}, error) {
	if isNil(b) {
		return nil, nil
	}

	raw := bson.Raw(b)

	r, err := raw.Values()
	if err != nil {
		return nil, err
	}

	s := make([]interface{}, len(r))
	for i := range r {
		j, err := enc.Decode(r[i].Value)
		if err != nil {
			return nil, errors.Wrap(err, "Decode slice")
		}

		s[i] = j
	}

	return s, nil
}

func (enc *Encoder) DecodeMap(b []byte) (map[string]interface{}, error) {
	if isNil(b) {
		return nil, nil
	}

	var r map[string]bson.Raw
	if err := bson.Unmarshal(b, &r); err != nil {
		return nil, errors.Wrap(err, "Decode map")
	}

	s := map[string]interface{}{}
	for k, v := range r {
		var i interface{}
		_, err := enc.guessHint(v)
		if err != nil {
			err := enc.Unmarshal(v, i)
			if err != nil {
				return nil, err
			}
			s[k] = i
		} else {
			j, err := enc.Decode(v)
			if err != nil {
				return nil, err
			}
			s[k] = j
		}
	}

	return s, nil
}

func (enc *Encoder) addDecodeDetail(d encoder.DecodeDetail) error {
	if err := enc.decoders.Add(d.Hint, d); err != nil {
		return util.ErrInvalid.WithMessage(err, "add DecodeDetail in bson encoder")
	}

	return nil
}

func (enc *Encoder) decodeWithHint(b []byte, s string) (interface{}, error) {
	ht, d, found, err := enc.decoders.FindByString(s)
	if err != nil {
		return nil, util.ErrNotFound.WithMessage(err, "find decoder by hint, %q", s)
	}

	if !found {
		return nil,
			util.ErrNotFound.Errorf("find decoder by hint, %q in bson decoders", ht)
	}

	i, err := d.Decode(b, ht)
	if err != nil {
		return nil, errors.WithMessagef(err, "Decode, %q in bson decoders", ht)
	}

	return i, nil
}

func (*Encoder) guessHint(b []byte) (string, error) {
	e := util.StringError("guess hint")

	var head HintedHead
	if err := bson.Unmarshal(b, &head); err != nil {
		return head.H, e.WithMessage(err, "hint not found in head")
	}

	return head.H, nil
}

func (enc *Encoder) analyze(d encoder.DecodeDetail, v interface{}) encoder.DecodeDetail {
	orig := reflect.ValueOf(v)
	ptr, elem := encoder.Ptr(orig)

	tointerface := func(i interface{}) interface{} {
		return reflect.ValueOf(i).Elem().Interface()
	}

	if orig.Type().Kind() == reflect.Ptr {
		tointerface = func(i interface{}) interface{} {
			return i
		}
	}

	switch ptr.Interface().(type) {
	case BSONDecodable:
		d.Desc = "BSONDecodable"
		d.Decode = func(b []byte, _ hint.Hint) (interface{}, error) {
			e := util.StringError("DecodeBSON")
			i := reflect.New(elem.Type()).Interface()

			if err := i.(BSONDecodable).DecodeBSON(b, enc); err != nil { //nolint:forcetypeassert //...
				return nil, e.Wrap(err)
			}

			return tointerface(i), nil
		}
	case bson.Unmarshaler:
		d.Desc = "BSONUnmarshaler"
		d.Decode = func(b []byte, _ hint.Hint) (interface{}, error) {
			e := util.StringError("UnmarshalBSON")
			i := reflect.New(elem.Type()).Interface()

			if err := i.(bson.Unmarshaler).UnmarshalBSON(b); err != nil { //nolint:forcetypeassert //...
				return nil, e.Wrap(err)
			}

			return reflect.ValueOf(i).Elem().Interface(), nil
		}
	case encoding.TextUnmarshaler:
		d.Desc = "TextUnmarshaler"
		d.Decode = func(b []byte, _ hint.Hint) (interface{}, error) {
			e := util.StringError("UnmarshalText")
			i := reflect.New(elem.Type()).Interface()

			if err := i.(encoding.TextUnmarshaler).UnmarshalText(b); err != nil { //nolint:forcetypeassert //...
				return nil, e.Wrap(err)
			}

			return tointerface(i), nil
		}
	default:
		d.Desc = "native"
		d.Decode = func(b []byte, _ hint.Hint) (interface{}, error) {
			e := util.StringError("native UnmarshalBSON")
			i := reflect.New(elem.Type()).Interface()

			if err := bson.Unmarshal(b, i); err != nil {
				return nil, e.Wrap(err)
			}

			return tointerface(i), nil
		}
	}

	return enc.analyzeExtensible(
		encoder.AnalyzeSetHinter(d, orig.Interface()),
		ptr,
	)
}

func (enc *Encoder) poolGet(s string) (interface{}, bool) {
	if enc.pool == nil {
		return nil, false
	}

	return enc.pool.Get(s)
}

func (enc *Encoder) poolSet(s string, v interface{}) {
	if enc.pool == nil {
		return
	}

	enc.pool.Set(s, v, 0)
}

func (*Encoder) analyzeExtensible(d encoder.DecodeDetail, ptr reflect.Value) encoder.DecodeDetail {
	p := d.Decode

	d.Decode = func(b []byte, ht hint.Hint) (interface{}, error) {
		i, err := p(b, ht)
		if err != nil {
			return i, err
		}

		if i == nil {
			return i, nil
		}

		return i, nil
	}

	return d
}

func isNil(b []byte) bool {
	return len(b) < 1
}
