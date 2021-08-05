package nl

import (
	"syscall"
	"unsafe"
)

func Sendmsg(s int, msg *syscall.Msghdr, flags int) (n int, err error) {
	r0, _, e1 := syscall.Syscall(syscall.SYS_SENDMSG, uintptr(s), uintptr(unsafe.Pointer(msg)), uintptr(flags))
	n = int(r0)
	if e1 != 0 {
		err = syscall.Errno(e1)
	}
	return
}

func Recvmsg(s int, msg *syscall.Msghdr, flags int) (n int, err error) {
	r0, _, e1 := syscall.Syscall(syscall.SYS_RECVMSG, uintptr(s), uintptr(unsafe.Pointer(msg)), uintptr(flags))
	n = int(r0)
	if e1 != 0 {
		err = syscall.Errno(e1)
	}
	return
}

type Ifreq struct {
	Name  [syscall.IFNAMSIZ]byte
	Index uint32
}

func IfnameToIndex(name string) (i int, err error) {
	fd, err := syscall.Socket(syscall.AF_INET, syscall.SOCK_DGRAM, 0)
	if err != nil {
		return 0, err
	}
	defer syscall.Close(fd)

	var ifreq Ifreq
	copy(ifreq.Name[:syscall.IFNAMSIZ-1], name)
	_, _, e1 := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.SIOCGIFINDEX), uintptr(unsafe.Pointer(&ifreq)))
	if e1 != 0 {
		err = syscall.Errno(e1)
	}
	i = int(ifreq.Index)
	return
}
