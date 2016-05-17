package main

import (
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"log"
	"math/cmplx"
	"fmt"
	"math"
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

	label struct {
		l, r, t, b, i *gtk.Label
	}
}

func newma() *ma {
	ma := &ma{}

	pb, err := gdk.PixbufNew(gdk.COLORSPACE_RGB, false, 8, 256, 256)
	if err != nil {
		log.Fatal("pixbuf: ", err)
	}

	ma.pb = pb
	ma.setCoords(-0.5, 0.0, 3.0)

	return ma
}

func (ma *ma)setCoords(cx float64, cy float64, zw float64) {
	ma.w = ma.pb.GetWidth()
	ma.h = ma.pb.GetHeight()

	ma.minx = cx - (zw / 2)
	ma.maxx = cx + (zw / 2)
	ma.miny = cy - ((ma.maxx - ma.minx) * float64(ma.h) / float64(ma.w) / 2)
	ma.maxy = cy + ((ma.maxx - ma.minx) * float64(ma.h) / float64(ma.w) / 2)

	ma.sx = (ma.maxx - ma.minx)/float64(ma.w - 1)
	ma.sy = (ma.maxy - ma.miny)/float64(ma.h - 1)

	// http://math.stackexchange.com/a/30560
	ma.iter = int(math.Sqrt(math.Abs(2.0 * math.Sqrt(math.Abs(1 - math.Sqrt(5.0 / zw))))) * 66.5)
}

func (ma *ma)screenCoords(x float64, y float64) (float64, float64) {
	return (ma.minx + x * ma.sx), (ma.maxy - y * ma.sy)
}

var palette = [...][3]float64{
	{ 1.0, 0.0, 0.0 },
	{ 1.0, 1.0, 0.0 },
	{ 0.0, 1.0, 0.0 },
	{ 0.0, 1.0, 1.0 },
	{ 0.0, 0.0, 1.0 },
	{ 1.0, 0.0, 1.0 },
}

var log_escape = math.Log(2)

func (ma *ma)getColor(z, c complex128, i int) []byte {
	if i == ma.iter {
		return []byte{0, 0, 0}
	}
	for extra := 0; extra < 3; extra++ {
		z = z * z + c
		i++
	}
	mu := float64(i + 1) - math.Log(math.Log(cmplx.Abs(z))) / log_escape
	clr1 := int(mu)

	t2 := mu - float64(clr1)
	t1 := 1.0 - t2

	c1 := palette[clr1 % len(palette)]
	c2 := palette[(clr1 + 1) % len(palette)]

	return []byte{ byte(255 * (c1[0] * t1 + c2[0] * t2)),
		byte(255 * (c1[1] * t1 + c2[1] * t2)),
		byte(255 * (c1[2] * t1 + c2[2] * t2)) }
}

func (ma *ma)redraw() {
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

			copy(px[o:], ma.getColor(z, c, i))
		}
	}
}

func (ma *ma)updateLabels() {
	ma.label.l.SetText(fmt.Sprintf("%6.4E", ma.minx))
	ma.label.r.SetText(fmt.Sprintf("%6.4E", ma.maxx))
	ma.label.t.SetText(fmt.Sprintf("%6.4E", ma.miny))
	ma.label.b.SetText(fmt.Sprintf("%6.4E", ma.maxy))
	ma.label.i.SetText(fmt.Sprintf("%d", ma.iter))
}

func (ma *ma)widget() gtk.IWidget {
	hb, err := gtk.BoxNew(gtk.ORIENTATION_HORIZONTAL, 5)
	if err != nil {
		log.Fatal(err)
	}
	
	gr, err := gtk.GridNew()
	if err != nil {
		log.Fatal(err)
	}

	ph := fmt.Sprintf("%6.4E", 1.0)

	ma.label.l = label(ph)
	ma.label.r = label(ph)
	ma.label.t = label(ph)
	ma.label.b = label(ph)
	ma.label.i = label("000000")

	ma.updateLabels()

	gr.Attach(label("minx:"), 0, 0, 1, 1)
	gr.Attach(label("maxx:"), 0, 1, 1, 1)
	gr.Attach(label("miny:"), 0, 2, 1, 1)
	gr.Attach(label("maxy:"), 0, 3, 1, 1)
	gr.Attach(label("iter:"), 0, 4, 1, 1)
	gr.Attach(ma.label.l, 1, 0, 1, 1)
	gr.Attach(ma.label.r, 1, 1, 1, 1)
	gr.Attach(ma.label.t, 1, 2, 1, 1)
	gr.Attach(ma.label.b, 1, 3, 1, 1)
	gr.Attach(ma.label.i, 1, 4, 1, 1)

	hb.PackStart(gr, false, false, 0)

	hb.PackEnd(ma.pictureWidget(), true, false, 0)

	return hb
}

func (ma *ma)pictureWidget() gtk.IWidget {
	eb, err := gtk.EventBoxNew()
	if err != nil {
		log.Fatal(err)
	}

	im, err := gtk.ImageNewFromPixbuf(ma.pb)
	if err != nil {
		log.Fatal(err)
	}

	eb.Add(im)

	eb.AddEvents(int(gdk.SCROLL_MASK))

	zw := 3.0
	cx := -0.5
	cy := 0.0

	redraw := func() {
		ma.setCoords(cx, cy, zw)
		ma.redraw()
		ma.updateLabels()
		im.SetFromPixbuf(ma.pb)
		eb.QueueDraw()
	}

	redraw()

	_, err = eb.Connect("button_press_event", func(win *gtk.Window, ev *gdk.Event) {
		e := &gdk.EventButton{ev}
		cx, cy = ma.screenCoords(e.X(), e.Y())
		redraw()
	})
	_, err = eb.Connect("scroll-event", func(win *gtk.Window, ev *gdk.Event) {
		es := &gdk.EventScroll{ev}
		delta := es.DeltaY()
		if delta > 0.5 {
			delta = 0.5
		}
		delta *= (zw / 5.0)
		switch es.Direction() {
		case gdk.SCROLL_UP:
			zw -= delta
		case gdk.SCROLL_DOWN:
			zw += delta
		}
		redraw()
	})
	if err != nil {
		log.Fatal("connect: ", err)
	}
	return eb
}

func label(s string) *gtk.Label {
	l, err := gtk.LabelNew(s)
	if err != nil {
		log.Fatal(err)
	}
	return l
}

func main() {
	gtk.Init(nil)
	win, err := gtk.WindowNew(gtk.WINDOW_TOPLEVEL)
	if err != nil {
		log.Fatal(err)
	}
	win.Connect("destroy", gtk.MainQuit)

	ma := newma()

	win.Add(ma.widget())
	win.ShowAll()

	gtk.Main()
}
