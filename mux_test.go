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
