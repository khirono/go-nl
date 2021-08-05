package nl

import (
	"sync"
	"syscall"
)

type Mux struct {
	epfd int
	es   map[int]*muxEntry
	mu   sync.Mutex
}

func NewMux() (*Mux, error) {
	m := new(Mux)
	fd, err := syscall.EpollCreate1(0)
	if err != nil {
		return nil, err
	}
	m.epfd = fd
	m.es = make(map[int]*muxEntry)
	return m, nil
}

func (m *Mux) Close() {
	syscall.Close(m.epfd)
}

func (m *Mux) PushHandler(conn *Conn, handler Handler) error {
	fd := conn.Fd()
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.es[fd]
	if ok {
		e.handlers = append(e.handlers, handler)
		return nil
	}
	e = newmuxEntry(conn)
	e.handlers = append(e.handlers, handler)
	m.es[fd] = e
	event := &syscall.EpollEvent{
		Events: syscall.EPOLLIN,
		Fd:     int32(fd),
	}
	return syscall.EpollCtl(m.epfd, syscall.EPOLL_CTL_ADD, fd, event)
}

func (m *Mux) PushHandlerFunc(conn *Conn, f func(msg *Msg) bool) error {
	return m.PushHandler(conn, HandlerFunc(f))
}

func (m *Mux) PopHandler(conn *Conn) {
	fd := conn.Fd()
	m.mu.Lock()
	defer m.mu.Unlock()
	e, ok := m.es[fd]
	if !ok {
		return
	}
	n := len(e.handlers)
	if n <= 1 {
		syscall.EpollCtl(m.epfd, syscall.EPOLL_CTL_DEL, fd, nil)
		delete(m.es, fd)
		return
	}
	e.handlers = e.handlers[:n]
}

func (m *Mux) Serve() error {
	events := make([]syscall.EpollEvent, 1)
	for {
		_, err := syscall.EpollWait(m.epfd, events, -1)
		if err != nil {
			return err
		}
		for _, event := range events {
			m.mu.Lock()
			e, ok := m.es[int(event.Fd)]
			if !ok {
				m.mu.Unlock()
				continue
			}
			e.ServeConn()
			m.mu.Unlock()
		}
	}
	return nil
}

type muxEntry struct {
	conn     *Conn
	buf      []byte
	handlers HandlerStack
}

func newmuxEntry(conn *Conn) *muxEntry {
	e := new(muxEntry)
	e.conn = conn
	e.buf = make([]byte, 64*1024)
	return e
}

func (e *muxEntry) ServeConn() error {
	n, err := e.conn.Read(e.buf)
	if err != nil {
		return err
	}
	b := make([]byte, n)
	copy(b, e.buf[:n])
	off := 0
	for off < len(b) {
		msg, n, err := DecodeMsg(b[off:])
		if err != nil {
			return err
		}
		e.handlers.ServeMsg(msg)
		off += n
	}
	return nil
}

type Handler interface {
	ServeMsg(*Msg) bool
}

type HandlerFunc func(*Msg) bool

func (f HandlerFunc) ServeMsg(msg *Msg) bool {
	return f(msg)
}

type HandlerStack []Handler

func (hl HandlerStack) ServeMsg(msg *Msg) bool {
	n := len(hl)
	for i := n - 1; i >= 0; i-- {
		if hl[i].ServeMsg(msg) {
			return true
		}
	}
	return false
}
