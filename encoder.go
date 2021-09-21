package nl

type Encoder interface {
	Len() int
	Encode([]byte) (int, error)
}

type Encoders []Encoder

func (es Encoders) Len() int {
	n := 0
	for _, e := range es {
		n += e.Len()
	}
	return n
}

func (es Encoders) Encode(b []byte) (int, error) {
	off := 0
	for _, e := range es {
		n, err := e.Encode(b[off:])
		if err != nil {
			return off, err
		}
		off += n
	}
	return off, nil
}
