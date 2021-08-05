package nl

import (
	"syscall"
)

type Client struct {
	conn   *Conn
	mux    *Mux
	ch     chan *Msg
	reqhdr Header
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
	c.reqhdr = req.Header

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
			if msg.Header.Flags&syscall.NLM_F_MULTI == 0 {
				return rsps, nil
			}
		}
	}

	return rsps, nil
}

func (c *Client) ServeMsg(msg *Msg) bool {
	switch msg.Header.Type {
	case syscall.NLMSG_DONE:
	case syscall.NLMSG_ERROR:
	case c.reqhdr.Type:
	default:
		return false
	}
	if msg.Header.Seq != c.reqhdr.Seq {
		return false
	}
	if msg.Header.Pid == 0 {
		return false
	}
	c.ch <- msg
	return true
}
