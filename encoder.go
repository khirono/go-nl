package nl

type Encoder interface {
	Len() int
	Encode([]byte) (int, error)
}
