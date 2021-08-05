package nl

import (
	"fmt"
	"syscall"
	"unsafe"
)

type Conn struct {
	fd  int
	seq int
}

func Open(proto int, groups ...int) (*Conn, error) {
	c := new(Conn)
	c.seq = 1
	typ := syscall.SOCK_RAW | syscall.SOCK_CLOEXEC
	fd, err := syscall.Socket(syscall.AF_NETLINK, typ, proto)
	if err != nil {
		return nil, err
	}
	var addr syscall.SockaddrNetlink
	addr.Family = syscall.AF_NETLINK
	for _, group := range groups {
		addr.Groups |= 1 << (group - 1)
	}
	err = syscall.Bind(fd, &addr)
	if err != nil {
		syscall.Close(fd)
		return nil, err
	}
	c.fd = fd
	return c, nil
}

func (c *Conn) Fd() int {
	return c.fd
}

func (c *Conn) Close() {
	syscall.Close(c.fd)
}

func (c *Conn) Read(b []byte) (int, error) {
	n, from, err := syscall.Recvfrom(c.fd, b, 0)
	if err != nil {
		return n, err
	}
	_, ok := from.(*syscall.SockaddrNetlink)
	if !ok {
		return n, fmt.Errorf("not netlink addr %v", from)
	}
	return n, err
}

func (c *Conn) Write(b []byte) (int, error) {
	var iovs [1]syscall.Iovec
	iovs[0].Base = &b[0]
	iovs[0].Len = uint64(len(b))
	return c.Writev(iovs[:])
}

func (c *Conn) Writev(iovs []syscall.Iovec) (int, error) {
	var addr syscall.SockaddrNetlink
	addr.Family = syscall.AF_NETLINK
	var msg syscall.Msghdr
	msg.Name = (*byte)(unsafe.Pointer(&addr))
	msg.Namelen = syscall.SizeofSockaddrNetlink
	msg.Iov = &iovs[0]
	msg.Iovlen = uint64(len(iovs))
	n, err := Sendmsg(c.fd, &msg, 0)
	return n, err
}

func (c *Conn) TakeSeq() int {
	seq := c.seq
	c.seq++
	return seq
}
