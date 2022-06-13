package nl

import (
	"syscall"
)

type Client struct {
	conn Conner
	mux  *Mux
	done bool
	ch   chan *Msg
	req  *Request
}

func NewClient(conn Conner, mux *Mux) *Client {
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

	c.ch = make(chan *Msg, 1)
	c.done = false

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
		}
	}

	return rsps, nil
}

func (c *Client) ServeMsg(msg *Msg) bool {
	if c.done {
		return false
	}
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
	c.ch <- msg
	switch t {
	case syscall.NLMSG_DONE:
		close(c.ch)
		c.done = true
	case syscall.NLMSG_ERROR:
		close(c.ch)
		c.done = true
	default:
		if c.req.Header.Flags&syscall.NLM_F_DUMP == 0 {
			close(c.ch)
			c.done = true
		}
	}
	return true
}
