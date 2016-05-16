package main

import (
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"log"
	"math/cmplx"
)

type ma struct {
	pb *gdk.Pixbuf

	maxx float64
	maxy float64

	minx float64
	miny float64

	sx float64
	sy float64

	iter int

	h int
	w int
}

func newma(pb *gdk.Pixbuf) *ma {
	ma := &ma{ pb: pb }

	w := pb.GetWidth()
	h := pb.GetHeight()

	ma.setCoords(-0.5, 0.0, 3.0)

	ma.h = h
	ma.w = w

	ma.iter = 100

	return ma
}

func (ma *ma)setCoords(cx float64, cy float64, zw float64) {
	w := ma.pb.GetWidth()
	h := ma.pb.GetHeight()

	ma.minx = cx - (zw / 2)
	ma.maxx = cx + (zw / 2)
	ma.miny = cy - ((ma.maxx - ma.minx) * float64(h) / float64(w) / 2)
	ma.maxy = cy + ((ma.maxx - ma.minx) * float64(h) / float64(w) / 2)

	ma.sx = (ma.maxx - ma.minx)/float64(w - 1)
	ma.sy = (ma.maxy - ma.miny)/float64(h - 1)
}

func (ma *ma)fill() {
	nc := ma.pb.GetNChannels()
	rs := ma.pb.GetRowstride()
	px := ma.pb.GetPixels()

	for y := 0; y < ma.h; y++ {
		cy := ma.maxy - float64(y) * ma.sy
		for x := 0; x < ma.w; x++ {
			cx := ma.minx + float64(x) * ma.sx
			o := y * rs + x * nc

			c := complex(cx, cy)
			z := c
			i := 0
			for ; i < ma.iter; i++ {
				if cmplx.Abs(z) > 2.0 {
					break
				}
				z = z * z + c
			}

			rc := byte(0)
			if i < ma.iter {
				c := float64(i) / float64(ma.iter)
				rc = byte(c * 255)
			}

			px[o + 0] = rc
			px[o + 1] = rc
			px[o + 2] = rc
		}
	}
}

func main() {
	gtk.Init(nil)
	win, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		log.Fatal(err)
	}

	pb, err := gdk.PixbufNew(gdk.COLORSPACE_RGB, false, 8, 256, 256)
	if err != nil {
		log.Fatal("pixbuf: ", err)
	}

	ma := newma(pb)

	ma.fill()

	im, err := gtk.ImageNewFromPixbuf(pb)
	if err != nil {
		log.Fatal(err)
	}


	zw := 3.0
	win.Connect("key-press-event", func(win *gtk.Window, ev *gdk.Event) {
		zw -= 0.05
		ma.setCoords(-0.5, 0, zw)
		ma.fill()
		im.SetFromPixbuf(ma.pb)
		win.QueueDraw()
	})

	win.Add(im)
	win.Connect("destroy", gtk.MainQuit)
	win.ShowAll()

	gtk.Main()
}
