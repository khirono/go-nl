package nl

import (
	"bytes"
	"io"
	"syscall"
)

const (
	NLA_TYPE_MASK = ^uint16(syscall.NLA_F_NESTED | syscall.NLA_F_NET_BYTEORDER)
)

type AttrLen uint16

func (l AttrLen) Align() int {
	return (int(l) + 3) &^ 3
}

type AttrHdr struct {
	Len  AttrLen
	Type uint16
}

func DecodeAttrHdr(b []byte) (AttrHdr, int, error) {
	var hdr AttrHdr
	if len(b) < 4 {
		return hdr, 0, io.ErrUnexpectedEOF
	}
	hdr.Len = AttrLen(native.Uint16(b[0:2]))
	hdr.Type = native.Uint16(b[2:4])
	return hdr, 4, nil
}

func (h AttrHdr) MaskedType() int {
	return int(h.Type & NLA_TYPE_MASK)
}

func (h AttrHdr) Nested() bool {
	return h.Type&syscall.NLA_F_NESTED != 0
}

func (h AttrHdr) NetByteorder() bool {
	return h.Type&syscall.NLA_F_NET_BYTEORDER != 0
}

type Attr struct {
	Type   uint16
	length AttrLen
	Value  Encoder
}

func (a *Attr) Len() int {
	if a.length == 0 {
		a.length = AttrLen(4)
		if a.Value != nil {
			a.length += AttrLen(a.Value.Len())
		}
	}
	return a.length.Align()
}

func (a *Attr) Encode(b []byte) (int, error) {
	n := a.Len()
	if len(b) < n {
		return 0, io.ErrShortWrite
	}
	native.PutUint16(b[0:2], uint16(a.length))
	typ := a.Type
	if a.Nested() {
		typ |= syscall.NLA_F_NESTED
	}
	native.PutUint16(b[2:4], typ)
	if a.Value != nil {
		a.Value.Encode(b[4:a.length])
	}
	return n, nil
}

func (a *Attr) Nested() bool {
	if a.Value == nil {
		return false
	}
	_, ok := a.Value.(AttrList)
	return ok
}

type AttrList []Attr

func (al AttrList) Len() int {
	n := 0
	for _, a := range al {
		n += a.Len()
	}
	return n
}

func (al AttrList) Encode(b []byte) (int, error) {
	off := 0
	for _, a := range al {
		n, err := a.Encode(b[off:])
		if err != nil {
			return off, err
		}
		off += n
	}
	return off, nil
}

type AttrU64 uint64

func DecodeAttrU64(b []byte) (uint64, int, error) {
	if len(b) < 8 {
		return 0, 0, io.ErrUnexpectedEOF
	}
	u := native.Uint64(b)
	return u, 8, nil
}

func (u AttrU64) Len() int {
	return 8
}

func (u AttrU64) Encode(b []byte) (int, error) {
	native.PutUint64(b, uint64(u))
	return 8, nil
}

type AttrU32 uint32

func DecodeAttrU32(b []byte) (uint32, int, error) {
	if len(b) < 4 {
		return 0, 0, io.ErrUnexpectedEOF
	}
	u := native.Uint32(b)
	return u, 4, nil
}

func (u AttrU32) Len() int {
	return 4
}

func (u AttrU32) Encode(b []byte) (int, error) {
	native.PutUint32(b, uint32(u))
	return 4, nil
}

type AttrU16 uint16

func DecodeAttrU16(b []byte) (uint16, int, error) {
	if len(b) < 2 {
		return 0, 0, io.ErrUnexpectedEOF
	}
	u := native.Uint16(b)
	return u, 2, nil
}

func (u AttrU16) Len() int {
	return 2
}

func (u AttrU16) Encode(b []byte) (int, error) {
	native.PutUint16(b, uint16(u))
	return 2, nil
}

type AttrU8 uint8

func DecodeAttrU8(b []byte) (uint8, int, error) {
	if len(b) < 1 {
		return 0, 0, io.ErrUnexpectedEOF
	}
	u := uint8(b[0])
	return u, 1, nil
}

func (u AttrU8) Len() int {
	return 1
}

func (u AttrU8) Encode(b []byte) (int, error) {
	b[0] = byte(u)
	return 1, nil
}

type AttrString string

func DecodeAttrString(b []byte) (string, int, error) {
	i := bytes.IndexByte(b, 0)
	if i == -1 {
		s := string(b)
		return s, len(s), nil
	}
	s := string(b[:i])
	return s, i + 1, nil
}

func (s AttrString) Len() int {
	return len(s) + 1
}

func (s AttrString) Encode(b []byte) (int, error) {
	n := copy(b, s)
	return n + 1, nil
}
