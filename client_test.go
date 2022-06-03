package nl

import (
	"syscall"
	"testing"
	"unsafe"
)

type MockConn struct {
	fd  int
	seq int
}

func (c *MockConn) Fd() int {
	return c.fd
}

func (c *MockConn) Close() {
	syscall.Close(c.fd)
}

func (c *MockConn) Read(b []byte) (int, error) {
	return syscall.Read(c.fd, b)
}

func (c *MockConn) Write(b []byte) (int, error) {
	return syscall.Write(c.fd, b)
}

func (c *MockConn) Writev(iovs []syscall.Iovec) (int, error) {
	r0, _, e1 := syscall.Syscall(syscall.SYS_WRITEV, uintptr(c.fd), uintptr(unsafe.Pointer(&iovs[0])), uintptr(len(iovs)))
	n := int(r0)
	if e1 != 0 {
		return n, syscall.Errno(e1)
	}
	return n, nil
}

func (c *MockConn) TakeSeq() int {
	seq := c.seq
	c.seq++
	return seq
}

func TestClient(t *testing.T) {
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_DGRAM, 0)
	go func() {
		conn := &MockConn{fd: fds[1], seq: 1}
		defer conn.Close()
		buf := make([]byte, 1024)
		_, err := conn.Read(buf)
		if err != nil {
			t.Error(err)
			return
		}
		for i := 0; i < 2; i++ {
			h := Header{
				Len:  syscall.SizeofNlMsghdr + 4,
				Type: uint16(syscall.NLMSG_DONE),
				Seq:  1,
				Pid:  1,
			}
			status := int32(0)
			iovs := []syscall.Iovec{
				{
					Base: (*byte)(unsafe.Pointer(&h)),
					Len:  syscall.SizeofNlMsghdr,
				},
				{
					Base: (*byte)(unsafe.Pointer(&status)),
					Len:  4,
				},
			}
			if _, err := conn.Writev(iovs); err != nil {
				t.Error(err)
			}
		}
	}()
	conn := &MockConn{fd: fds[0], seq: 1}
	defer conn.Close()
	mux, err := NewMux()
	if err != nil {
		t.Fatal(err)
	}
	defer mux.Close()
	go mux.Serve()
	c := NewClient(conn, mux)
	req := NewRequest(0, 0)
	if _, err := c.Do(req); err != nil {
		t.Fatal(err)
	}
}
