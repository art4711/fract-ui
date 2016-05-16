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

func (ma *ma)screenCoords(x float64, y float64) (float64, float64) {
	return (ma.minx + x * ma.sx), (ma.maxy - y * ma.sy)
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
	win.Connect("destroy", gtk.MainQuit)

	eb, err := gtk.EventBoxNew()
	if err != nil {
		log.Fatal(err)
	}
	win.Add(eb)

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

	eb.Add(im)

	eb.AddEvents(int(gdk.KEY_PRESS_MASK|gdk.SCROLL_MASK))

	zw := 3.0
	cx := -0.5
	cy := 0.0

	_, err = eb.Connect("button_press_event", func(win *gtk.Window, ev *gdk.Event) {
		e := &gdk.EventButton{ev}
		cx, cy = ma.screenCoords(e.X(), e.Y())
		log.Printf("button %v %v %v %v", e.X(), e.Y(), cx, cy)
		ma.setCoords(cx, cy, zw)
		ma.fill()
		im.SetFromPixbuf(ma.pb)
		win.QueueDraw()
	})
	_, err = eb.Connect("scroll-event", func(win *gtk.Window, ev *gdk.Event) {
		es := &gdk.EventScroll{ev}
		delta := es.DeltaY() * (zw / 5.0)
		switch es.Direction() {
		case gdk.SCROLL_UP:
			zw -= delta
		case gdk.SCROLL_DOWN:
			zw += delta
		}
		ma.setCoords(cx, cy, zw)
		ma.fill()
		im.SetFromPixbuf(ma.pb)
		win.QueueDraw()
	})
	if err != nil {
		log.Fatal("connect: ", err)
	}

	win.ShowAll()

	gtk.Main()
}
