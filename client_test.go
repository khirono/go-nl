package nl

import (
	"io"
	"sync"
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

func replyStatus(conn Conner, seq, pid uint32, errno int) error {
	var typ uint16
	if errno == 0 {
		typ = syscall.NLMSG_DONE
	} else {
		typ = syscall.NLMSG_ERROR
	}
	h := Header{
		Len:  syscall.SizeofNlMsghdr + 4,
		Type: typ,
		Seq:  seq,
		Pid:  pid,
	}
	status := int32(-errno)
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
	_, err := conn.Writev(iovs)
	return err
}

func replyPong(conn Conner, seq, pid, ident uint32) error {
	h := Header{
		Len:  syscall.SizeofNlMsghdr + 4,
		Type: uint16(20),
		Seq:  seq,
		Pid:  pid,
	}
	iovs := []syscall.Iovec{
		{
			Base: (*byte)(unsafe.Pointer(&h)),
			Len:  syscall.SizeofNlMsghdr,
		},
		{
			Base: (*byte)(unsafe.Pointer(&ident)),
			Len:  4,
		},
	}
	_, err := conn.Writev(iovs)
	return err
}

func TestClient_TooManyDone(t *testing.T) {
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
			err := replyStatus(conn, 1, 1, 0)
			if err != nil {
				t.Error(err)
			}
		}
	}()
	conn := &MockConn{fd: fds[0], seq: 1}
	defer conn.Close()
	var wg sync.WaitGroup
	mux, err := NewMux()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		mux.Close()
		wg.Wait()
	}()
	wg.Add(1)
	go func() {
		mux.Serve()
		wg.Done()
	}()
	c := NewClient(conn, mux)
	req := NewRequest(0, 0)
	if _, err := c.Do(req); err != nil {
		t.Fatal(err)
	}
}

func TestClient_PingPong(t *testing.T) {
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_DGRAM, 0)
	go func() {
		ident := uint32(123)
		conn := &MockConn{fd: fds[1], seq: 1}
		defer conn.Close()
		for {
			buf := make([]byte, 1024)
			n, err := conn.Read(buf)
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Error(err)
				return
			}
			reqmsg, _, err := DecodeMsg(buf[:n])
			if err != nil {
				t.Error(err)
				return
			}
			err = replyPong(conn, reqmsg.Header.Seq, 1, ident)
			if err != nil {
				t.Error(err)
			}
			err = replyStatus(conn, reqmsg.Header.Seq, 1, 0)
			if err != nil {
				t.Error(err)
			}
			ident++
		}
	}()
	conn := &MockConn{fd: fds[0], seq: 1}
	defer conn.Close()
	var wg sync.WaitGroup
	mux, err := NewMux()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		mux.Close()
		wg.Wait()
	}()
	wg.Add(1)
	go func() {
		mux.Serve()
		wg.Done()
	}()
	c := NewClient(conn, mux)

	N := 100
	for i := 0; i < N; i++ {
		req := NewRequest(20, 0)
		rsps, err := c.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if len(rsps) < 1 {
			t.Fatalf("too few response %v\n", len(rsps))
		}
		ident := *(*uint32)(unsafe.Pointer(&rsps[0].Body[0]))
		want := uint32(123 + i)
		if ident != want {
			t.Errorf("want %v; but got %v\n", want, ident)
		}
	}
}
