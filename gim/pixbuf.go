package gim

type pb struct {
	h, w int
	px []byte
}

type Pixbuf interface {
	GetNChannels() int
	GetRowstride() int
	GetPixels() []byte
	GetWidth() int
	GetHeight() int
}

func NewPixbuf(h, w int) Pixbuf {
	return &pb{ h: h, w: w, px: make([]byte, 3*h*w) }
}

func (pb *pb)GetNChannels() int {
	return 3
}

func (pb *pb)GetRowstride() int {
	return pb.w*3
}

func (pb *pb)GetPixels() []byte {
	return pb.px
}

func (pb *pb)GetWidth() int {
	return pb.w
}

func (pb *pb)GetHeight() int {
	return pb.h
}
