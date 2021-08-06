package nl

import (
	"sync"
	"syscall"
)

type Mux struct {
	epfd int
	es   map[int]muxEntry
	mu   sync.Mutex
}

func NewMux() (*Mux, error) {
	m := new(Mux)
	fd, err := syscall.EpollCreate1(0)
	if err != nil {
		return nil, err
	}
	m.epfd = fd
	m.es = make(map[int]muxEntry)
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
		e.handlers = append([]Handler{handler}, e.handlers...)
		m.es[fd] = e
		return nil
	}
	buf := make([]byte, 64*1024)
	m.es[fd] = muxEntry{conn, buf, HandlerStack{handler}}
	return m.subscribe(fd)
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
		m.unsubscribe(fd)
		delete(m.es, fd)
		return
	}
	e.handlers = append([]Handler{}, e.handlers[1:]...)
	m.es[fd] = e
}

func (m *Mux) Serve() error {
	events := make([]syscall.EpollEvent, 1)
	for {
		_, err := syscall.EpollWait(m.epfd, events, -1)
		if err != nil {
			return err
		}
		event := events[0]

		fd := int(event.Fd)
		m.unsubscribe(fd)
		m.mu.Lock()
		e, ok := m.es[fd]
		if !ok {
			m.mu.Unlock()
			continue
		}
		m.mu.Unlock()
		b, err := e.ReadBytes()
		if err != nil {
			continue
		}
		go func() {
			e.Serve(b)
			m.mu.Lock()
			_, ok := m.es[fd]
			if ok {
				m.subscribe(fd)
			}
			m.mu.Unlock()
		}()
	}
	return nil
}

func (m *Mux) subscribe(fd int) error {
	event := &syscall.EpollEvent{
		Events: syscall.EPOLLIN,
		Fd:     int32(fd),
	}
	return syscall.EpollCtl(m.epfd, syscall.EPOLL_CTL_ADD, fd, event)
}

func (m *Mux) unsubscribe(fd int) error {
	return syscall.EpollCtl(m.epfd, syscall.EPOLL_CTL_DEL, fd, nil)
}

type muxEntry struct {
	conn     *Conn
	buf      []byte
	handlers HandlerStack
}

func (e muxEntry) ReadBytes() ([]byte, error) {
	n, err := e.conn.Read(e.buf)
	if err != nil {
		return nil, err
	}
	b := make([]byte, n)
	copy(b, e.buf[:n])
	return b, nil
}

func (e muxEntry) Serve(b []byte) error {
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

func (hs HandlerStack) ServeMsg(msg *Msg) bool {
	for _, h := range hs {
		if h.ServeMsg(msg) {
			return true
		}
	}
	return false
}
