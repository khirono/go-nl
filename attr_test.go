package nl

import (
	"bytes"
	"testing"
)

func TestAttr_Encode(t *testing.T) {
	data := []byte{
		0x20, 0x00,
		0x01, 0x80,
		0x08, 0x00,
		0x02, 0x00,
		0x9a, 0x78, 0x00, 0x00,
		0x06, 0x00,
		0x03, 0x00,
		0x32, 0x54, 0x00, 0x00,
		0x0b, 0x00,
		0x04, 0x00,
		0x67, 0x74, 0x70, 0x35, 0x67, 0x30, 0x00, 0x00,
	}
	a := &Attr{
		Type: 1,
		Value: AttrList{
			{
				Type:  2,
				Value: AttrU32(0x789a),
			},
			{
				Type:  3,
				Value: AttrU16(0x5432),
			},
			{
				Type:  4,
				Value: AttrString("gtp5g0"),
			},
		},
	}
	n := a.Len()
	b := make([]byte, n)
	r, err := a.Encode(b)
	if err != nil {
		t.Fatal(err)
	}
	if r != len(data) {
		t.Errorf("want: %v; but got %v\n", len(data), r)
	}
	if !bytes.Equal(b, data) {
		t.Errorf("want: %x; but got %x\n", data, b)
	}
}

func TestDecodeAttr(t *testing.T) {
	data := []byte{
		0x20, 0x00,
		0x01, 0x00,
		0x08, 0x00,
		0x02, 0x00,
		0x9a, 0x78, 0x00, 0x00,
		0x06, 0x00,
		0x03, 0x00,
		0x32, 0x54, 0x00, 0x00,
		0x0b, 0x00,
		0x04, 0x00,
		0x67, 0x74, 0x70, 0x35, 0x67, 0x30, 0x00, 0x00,
	}
	hdr, n, err := DecodeAttrHdr(data)
	if err != nil {
		t.Fatal(err)
	}
	if hdr.Type != 1 {
		t.Errorf("want: %v; but got %v\n", 1, hdr.Type)
	}
	if hdr.Len != 32 {
		t.Errorf("want: %v; but got %v\n", 32, hdr.Len)
	}
	d := data[n:]
	for len(d) > 0 {
		hdr, n, err := DecodeAttrHdr(d)
		if err != nil {
			t.Fatal(err)
		}
		switch hdr.Type {
		case 2:
			v, _, err := DecodeAttrU32(d[n:])
			if err != nil {
				t.Fatal(err)
			}
			if v != 0x789a {
				t.Errorf("want: %v; but got %v\n", 0x789a, v)
			}
		case 3:
			v, _, err := DecodeAttrU16(d[n:])
			if err != nil {
				t.Fatal(err)
			}
			if v != 0x5432 {
				t.Errorf("want: %v; but got %v\n", 0x5432, v)
			}
		case 4:
			v, _, err := DecodeAttrString(d[n:])
			if err != nil {
				t.Fatal(err)
			}
			if v != "gtp5g0" {
				t.Errorf("want: %v; but got %v\n", "gtp5g0", v)
			}
		default:
			t.Errorf("unknown header type %v\n", hdr.Type)
		}
		d = d[hdr.Len.Align():]
	}
}
