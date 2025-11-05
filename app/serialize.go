package app

import (
	"bytes"
	"encoding/binary"
	"io"

	"github.com/bvisness/flowshell/util"
)

type Serializer struct {
	Buf     *bytes.Buffer
	Encode  bool
	Version int
	Err     error
}

type Serializable[T any] interface {
	Serialize(s *Serializer, v *T) error
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

func (s *Serializer) Error(err error) error {
	if s.Err != nil {
		return s.Err
	}

	s.Err = err
	return s.Err
}

func SBool(s *Serializer, b *bool) error {
	if s.Err != nil {
		return s.Err
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
	return nil
}

func SInt[T ~int | ~int32 | ~int64](s *Serializer, n *T) error {
	if s.Err != nil {
		return s.Err
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
	return nil
}

func SUint[T ~uint | ~uint32 | ~uint64](s *Serializer, n *T) error {
	if s.Err != nil {
		return s.Err
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
	return nil
}

func SFloat[T ~float32 | ~float64](s *Serializer, n *T) error {
	if s.Err != nil {
		return s.Err
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
	return nil
}

func SStr[T ~string](s *Serializer, str *T) error {
	if s.Err != nil {
		return s.Err
	}

	strlen := len(*str)
	if err := SInt(s, &strlen); err != nil {
		return s.Error(err)
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
	return nil
}

func (s *Serializer) ReadStr() (string, error) {
	util.Assert(!s.Encode)
	var res string
	if err := SStr(s, &res); err != nil {
		return "", err
	}
	return res, nil
}

func (s *Serializer) WriteStr(str string) error {
	util.Assert(s.Encode)
	return SStr(s, &str)
}

func SThing[T Serializable[T]](s *Serializer, v *T) error {
	var zero T
	return T.Serialize(zero, s, v)
}

func SMaybeThing[T Serializable[T]](s *Serializer, v **T) error {
	exists := *v != nil
	SBool(s, &exists)
	if exists {
		var newThing T
		SThing(s, &newThing)
		*v = &newThing
	}
	return s.Err
}

func SFixed[T any](s *Serializer, v *T) error {
	if s.Err != nil {
		return s.Err
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
	return nil
}

func SMaybeFixed[T any](s *Serializer, v **T) error {
	exists := v != nil
	SBool(s, &exists)
	if exists {
		SFixed(s, *v)
	}
	return s.Err
}

func SSlice[T Serializable[T]](s *Serializer, slice *[]T) error {
	n := len(*slice)
	if err := SInt(s, &n); err != nil {
		return s.Error(err)
	}

	if !s.Encode {
		if n == 0 {
			*slice = nil
		} else {
			*slice = make([]T, n)
		}
	}
	for i := range n {
		if err := SThing(s, &(*slice)[i]); err != nil {
			return s.Error(err)
		}
	}
	return s.Err
}
