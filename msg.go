package nl

import (
	"syscall"
)

type Msg struct {
	Header Header
	Body   []byte
}

func DecodeMsg(b []byte) (*Msg, int, error) {
	m := new(Msg)
	h, n, err := DecodeHeader(b)
	if err != nil {
		return nil, 0, err
	}
	m.Header = *h
	m.Body = b[n:m.Header.Len]
	return m, int(m.Header.Len), nil
}

func DecodeMsgError(b []byte) (error, int, error) {
	e := int32(native.Uint32(b[:4]))
	if e != 0 {
		return syscall.Errno(-e), 4, nil
	}
	return nil, 4, nil
}

type Header struct {
	Len   uint32
	Type  uint16
	Flags uint16
	Seq   uint32
	Pid   uint32
}

func DecodeHeader(b []byte) (*Header, int, error) {
	h := new(Header)
	h.Len = native.Uint32(b[0:4])
	h.Type = native.Uint16(b[4:6])
	h.Flags = native.Uint16(b[6:8])
	h.Seq = native.Uint32(b[8:12])
	h.Pid = native.Uint32(b[12:16])
	return h, 16, nil
}
