package nl

import (
	"syscall"
)

type Client struct {
	conn *Conn
	mux  *Mux
	done chan struct{}
	ch   chan *Msg
	req  *Request
}

func NewClient(conn *Conn, mux *Mux) *Client {
	c := new(Client)
	c.conn = conn
	c.mux = mux
	return c
}

func (c *Client) Do(req *Request) ([]Msg, error) {
	seq := c.conn.TakeSeq()
	req.Commit(seq)
	_, err := c.conn.Writev(req.Iovs)
	if err != nil {
		return nil, err
	}
	c.req = req

	c.done = make(chan struct{})
	defer close(c.done)

	c.ch = make(chan *Msg, 1)
	defer close(c.ch)

	c.mux.PushHandler(c.conn, c)
	defer c.mux.PopHandler(c.conn)

	var rsps []Msg
	for msg := range c.ch {
		switch msg.Header.Type {
		case syscall.NLMSG_DONE:
			err, _, _ := DecodeMsgError(msg.Body)
			return rsps, err
		case syscall.NLMSG_ERROR:
			err, _, _ := DecodeMsgError(msg.Body)
			return rsps, err
		default:
			rsps = append(rsps, *msg)
			if c.req.Header.Flags&syscall.NLM_F_DUMP == 0 {
				return rsps, nil
			}
		}
	}

	return rsps, nil
}

func (c *Client) ServeMsg(msg *Msg) bool {
	t := msg.Header.Type
	switch {
	case t == syscall.NLMSG_DONE:
	case t == syscall.NLMSG_ERROR:
	case c.req.ContainsReplyType(int(t)):
	default:
		return false
	}
	if msg.Header.Seq != c.req.Header.Seq {
		return false
	}
	if msg.Header.Pid == 0 {
		return false
	}
	select {
	case <-c.done:
		return false
	case c.ch <- msg:
	}
	return true
}
