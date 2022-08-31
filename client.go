package nl

import (
	"syscall"
)

type Client struct {
	conn Conner
	mux  *Mux
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

	ch := make(chan *Msg, 32)

	c.mux.PushHandlerFunc(c.conn, c.Handler(req, ch))
	defer c.mux.PopHandler(c.conn)

	var rsps []Msg
	for msg := range ch {
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

func (c *Client) Handler(req *Request, ch chan *Msg) HandlerFunc {
	var done bool
	return func(msg *Msg) bool {
		if done {
			return false
		}
		t := msg.Header.Type
		switch {
		case t == syscall.NLMSG_DONE:
		case t == syscall.NLMSG_ERROR:
		case req.ContainsReplyType(int(t)):
		default:
			return false
		}
		if msg.Header.Seq != req.Header.Seq {
			return false
		}
		if msg.Header.Pid == 0 {
			return false
		}
		ch <- msg
		switch t {
		case syscall.NLMSG_DONE:
			done = true
			close(ch)
		case syscall.NLMSG_ERROR:
			done = true
			close(ch)
		default:
			if !req.NeedAck() {
				done = true
				close(ch)
			}
		}
		return true
	}
}
