package gim

import (
	"unsafe"
)

type Pb struct {
	h, w int
	px   []uint32
}

type Pixbuf interface {
	GetRowstride() int
	GetPixels() []uint32
	GetWidth() int
	GetHeight() int
}

func NewPixbuf(h, w int) *Pb {
	return &Pb{h: h, w: w, px: make([]uint32, h*w)}
}

func (pb *Pb) GetRowstride() int {
	return pb.w
}

func (pb *Pb) GetRowstrideBytes() int {
	return pb.w * 4
}

func (pb *Pb) GetPixels() []uint32 {
	return pb.px
}

func (pb *Pb) GetWidth() int {
	return pb.w
}

func (pb *Pb) GetHeight() int {
	return pb.h
}

func (pb *Pb) GetPixelData() unsafe.Pointer {
	return unsafe.Pointer(&pb.px[0])
}
