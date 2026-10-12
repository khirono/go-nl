package nl

import (
	"sync"
	"syscall"
	"testing"
)

func TestMux(t *testing.T) {
	N := 10000
	for i := 0; i < N; i++ {
		var wg sync.WaitGroup
		mux, err := NewMux()
		if err != nil {
			t.Fatal(err)
		}
		wg.Add(1)
		go func() {
			mux.Serve()
			wg.Done()
		}()
		conn, err := Open(syscall.NETLINK_ROUTE)
		if err != nil {
			mux.Close()
			wg.Wait()
			t.Fatal(err)
		}
		mux.PushHandlerFunc(conn, func(msg *Msg) bool {
			return false
		})
		conn.Close()
		mux.Close()
		wg.Wait()
	}
}

// fdOpen reports whether fd is an open file descriptor.
func fdOpen(fd int) bool {
	_, _, errno := syscall.Syscall(syscall.SYS_FCNTL, uintptr(fd), syscall.F_GETFD, 0)
	return errno == 0
}

// reuseWakeFd opens /dev/null until it is handed the wake fd number Close
// released (fds take the lowest free number), and returns that fd. Every fd it
// opens stays open until the test ends.
func reuseWakeFd(t *testing.T, mux *Mux) int {
	t.Helper()
	for i := 0; i < 64; i++ {
		fd, err := syscall.Open("/dev/null", syscall.O_RDONLY|syscall.O_CLOEXEC, 0)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { syscall.Close(fd) })
		if fd == mux.cfds[1] {
			return fd
		}
	}
	t.Fatalf("never got the released wake fd %d back", mux.cfds[1])
	return -1
}

// TestMuxServeAfterCloseKeepsReusedFd covers Serve's exit closing the wake fd
// again after Close already did: an fd opened in between gets that number and
// must survive Serve returning.
func TestMuxServeAfterCloseKeepsReusedFd(t *testing.T) {
	mux, err := NewMux()
	if err != nil {
		t.Fatal(err)
	}
	mux.Close()
	fd := reuseWakeFd(t, mux)

	if err := mux.Serve(); err != nil {
		t.Fatal(err)
	}

	if !fdOpen(fd) {
		t.Fatalf("fd %d, opened after Close, was closed when Serve returned", fd)
	}
}

// TestMuxCloseTwiceKeepsReusedFd covers a second Close.
func TestMuxCloseTwiceKeepsReusedFd(t *testing.T) {
	mux, err := NewMux()
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		mux.Serve()
		wg.Done()
	}()
	mux.Close()
	wg.Wait()
	fd := reuseWakeFd(t, mux)

	mux.Close()

	if !fdOpen(fd) {
		t.Fatalf("fd %d, opened after the first Close, was closed by the second", fd)
	}
}
