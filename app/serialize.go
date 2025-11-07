package app

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/bvisness/flowshell/trace"
	"github.com/bvisness/flowshell/util"
)

type Serializer struct {
	Buf     *bytes.Buffer
	Encode  bool
	Version int
	Errs    []error
}

type Serializable interface {
	Serialize(s *Serializer) bool
}

func NewEncoder(version int) *Serializer {
	s := Serializer{
		Buf:     &bytes.Buffer{},
		Encode:  true,
		Version: version,
	}
	SInt(&s, &s.Version)
	return &s
}

func NewDecoder(buf []byte) *Serializer {
	s := Serializer{
		Buf:    bytes.NewBuffer(buf),
		Encode: false,
	}
	SInt(&s, &s.Version)
	return &s
}

func (s *Serializer) Bytes() []byte {
	if !s.Encode {
		panic("cannot call Serializer.Bytes() unless in Encode mode")
	}
	return s.Buf.Bytes()
}

func (s *Serializer) Ok() bool {
	return len(s.Errs) == 0
}

func (s *Serializer) Error(err error) bool {
	s.Errs = append(s.Errs, SerializeError{
		Err:   err,
		Stack: trace.Trace()[1:],
	})
	return false
}

func SBool(s *Serializer, b *bool) bool {
	if !s.Ok() {
		return false
	}

	if s.Encode {
		err := s.Buf.WriteByte(util.Tern[byte](*b, 0x01, 0x00))
		util.Assert(err == nil, "the documentation lied :(")
	} else {
		x, err := s.Buf.ReadByte()
		if err != nil {
			return s.Error(err)
		}
		*b = x > 0
	}
	return true
}

func SInt[T ~int | ~int32 | ~int64](s *Serializer, n *T) bool {
	if !s.Ok() {
		return false
	}

	if s.Encode {
		// Why couldn't they just have binary.WriteVarint again...?
		// https://github.com/golang/go/issues/29010
		var b [binary.MaxVarintLen64]byte
		nBytes := binary.PutVarint(b[:], int64(*n))
		if _, err := s.Buf.Write(b[:nBytes]); err != nil {
			return s.Error(err)
		}
	} else {
		x, err := binary.ReadVarint(s.Buf)
		if err != nil {
			return s.Error(err)
		}
		*n = T(x)
	}
	return true
}

func SUint[T ~uint | ~uint32 | ~uint64](s *Serializer, n *T) bool {
	if !s.Ok() {
		return false
	}

	if s.Encode {
		var b [binary.MaxVarintLen64]byte
		nBytes := binary.PutUvarint(b[:], uint64(*n))
		if _, err := s.Buf.Write(b[:nBytes]); err != nil {
			return s.Error(err)
		}
	} else {
		x, err := binary.ReadUvarint(s.Buf)
		if err != nil {
			return s.Error(err)
		}
		*n = T(x)
	}
	return true
}

func SFloat[T ~float32 | ~float64](s *Serializer, n *T) bool {
	if !s.Ok() {
		return false
	}

	if s.Encode {
		err := binary.Write(s.Buf, binary.LittleEndian, *n)
		if err != nil {
			return s.Error(err)
		}
	} else {
		err := binary.Read(s.Buf, binary.LittleEndian, n)
		if err != nil {
			return s.Error(err)
		}
	}
	return true
}

func SStr[T ~string](s *Serializer, str *T) bool {
	if !s.Ok() {
		return false
	}

	strlen := len(*str)
	if ok := SInt(s, &strlen); !ok {
		return false
	}

	if s.Encode {
		if _, err := s.Buf.Write([]byte(*str)); err != nil {
			return s.Error(err)
		}
	} else {
		res := make([]byte, strlen)
		if nRead, err := s.Buf.Read(res[:]); err != nil {
			return s.Error(err)
		} else if nRead < strlen {
			return s.Error(io.EOF)
		}
		*str = T(res)
	}
	return true
}

func (s *Serializer) ReadStr() (string, bool) {
	util.Assert(!s.Encode)
	var res string
	if ok := SStr(s, &res); !ok {
		return "", false
	}
	return res, true
}

func (s *Serializer) WriteStr(str string) bool {
	util.Assert(s.Encode)
	return SStr(s, &str)
}

func SThing[T any, PT PSerializable[T]](s *Serializer, v PT) bool {
	if !s.Ok() {
		return false
	}
	return v.Serialize(s)
}

func SMaybeThing[T any, PT PSerializable[T]](s *Serializer, v **T) bool {
	if !s.Ok() {
		return false
	}

	exists := *v != nil
	if ok := SBool(s, &exists); !ok {
		return false
	}
	if exists {
		var newThing T
		if ok := SThing(s, PT(&newThing)); !ok {
			return false
		}
		*v = &newThing
	}
	return true
}

func SFixed[T any](s *Serializer, v *T) bool {
	if !s.Ok() {
		return false
	}

	if s.Encode {
		if err := binary.Write(s.Buf, binary.LittleEndian, *v); err != nil {
			return s.Error(err)
		}
	} else {
		if err := binary.Read(s.Buf, binary.LittleEndian, v); err != nil {
			return s.Error(err)
		}
	}
	return true
}

func SMaybeFixed[T any](s *Serializer, v **T) bool {
	if !s.Ok() {
		return false
	}

	exists := v != nil
	if ok := SBool(s, &exists); !ok {
		return false
	}
	if exists {
		return SFixed(s, *v)
	}
	return true
}

func SSlice[T any, PT PSerializable[T]](s *Serializer, slice *[]T) bool {
	if !s.Ok() {
		return false
	}

	n := len(*slice)
	if ok := SInt(s, &n); !ok {
		return false
	}

	if !s.Encode {
		if n == 0 {
			*slice = nil
		} else {
			*slice = make([]T, n)
		}
	}
	for i := range n {
		if ok := SThing(s, PT(&(*slice)[i])); !ok {
			return false
		}
	}
	return true
}

// ------------------------------------
// Errors

type SerializeError struct {
	Err   error
	Stack trace.CallStack
}

func (e SerializeError) Error() string {
	return e.Err.Error()
}

func (e SerializeError) Unwrap() error {
	return e.Err
}

// --------------------------------------
// Type utilities

type PSerializable[T any] interface {
	*T
	Serializable
}

type PP[T any] interface {
	**T
}
