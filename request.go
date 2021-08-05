package nl

import (
	"syscall"
	"unsafe"
)

type Request struct {
	Header Header
	Iovs   []syscall.Iovec
}

func NewRequest(typ, flags int) *Request {
	r := new(Request)
	r.Header.Len = syscall.SizeofNlMsghdr
	r.Header.Type = uint16(typ)
	r.Header.Flags = syscall.NLM_F_REQUEST | uint16(flags)
	r.Iovs = make([]syscall.Iovec, 1)
	r.Iovs[0].Base = (*byte)(unsafe.Pointer(&r.Header))
	r.Iovs[0].Len = syscall.SizeofNlMsghdr
	return r
}

func (r *Request) Append(e Encoder) error {
	l := e.Len()
	b := make([]byte, l)
	_, err := e.Encode(b)
	if err != nil {
		return err
	}
	r.AppendBytes(b)
	return nil
}

func (r *Request) AppendBytes(b []byte) {
	l := len(b)
	r.Header.Len += uint32(l)
	iov := syscall.Iovec{Base: &b[0], Len: uint64(l)}
	r.Iovs = append(r.Iovs, iov)
}

func (r *Request) AppendPointer(p unsafe.Pointer, length int) {
	r.Header.Len += uint32(length)
	iov := syscall.Iovec{
		Base: (*byte)(p),
		Len:  uint64(length),
	}
	r.Iovs = append(r.Iovs, iov)
}

func (r *Request) Commit(seq int) {
	r.Header.Seq = uint32(seq)
}
