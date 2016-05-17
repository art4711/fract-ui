package main

import (
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"log"
	"math/cmplx"
	"fmt"
	"math"
	"time"
	"sync"
	"runtime"
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

	lastDuration time.Duration

	label struct {
		l, r, t, b, i, d *gtk.Label
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
	return (ma.minx + x * ma.sx), (ma.miny + y * ma.sy)
}

var palette = [...][3]float64{
	{ 1.0, 0.0, 0.0 },
	{ 1.0, 0.5, 0.0 },
	{ 1.0, 1.0, 0.0 },
	{ 0.5, 1.0, 0.5 },
	{ 0.0, 1.0, 1.0 },
	{ 0.5, 0.5, 0.5 },
}

var log_escape = math.Log(2)

func (ma *ma)getColor(z, c complex128, i int) (byte, byte, byte) {
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

	return byte(255 * (c1[0] * t1 + c2[0] * t2)),
		byte(255 * (c1[1] * t1 + c2[1] * t2)),
		byte(255 * (c1[2] * t1 + c2[2] * t2))
}

func (ma *ma)redrawRange(starty int, endy int, nc int, rs int, px []byte, wg *sync.WaitGroup) {
	for y := starty; y < endy; y++ {
		cy := ma.miny + float64(y) * ma.sy
		for x := 0; x < ma.w; x++ {
			cx := ma.minx + float64(x) * ma.sx
			o := y * rs + x * nc

			c := complex(cx, cy)
			z := c
			px[o], px[o + 1], px[o +2] = 0, 0, 0
			for i := 0; i < ma.iter; i++ {
				if cmplx.Abs(z) > 2.0 {
					px[o], px[o + 1], px[o + 2] = ma.getColor(z, c, i)
					break
				}
				z = z * z + c
			}
		}
	}
	wg.Done()
}

func (ma *ma)redraw() {
	nc := ma.pb.GetNChannels()
	rs := ma.pb.GetRowstride()
	px := ma.pb.GetPixels()

	startt := time.Now()

	var wg sync.WaitGroup

	steps := runtime.NumCPU()
	for i := 0; i < steps; i++ {
		wg.Add(1)
		go ma.redrawRange(i * ma.h / steps, (i + 1) * ma.h / steps, nc, rs, px, &wg)
	}

	wg.Wait()
	ma.lastDuration = time.Since(startt)
}

func (ma *ma)updateLabels(gr *gtk.Grid) {
	if gr != nil {
		ma.label.l = label("")
		ma.label.r = label("")
		ma.label.t = label("")
		ma.label.b = label("")
		ma.label.i = label("")
		ma.label.d = label("")
		gr.Attach(label("minx:"), 0, 0, 1, 1)
		gr.Attach(label("maxx:"), 0, 1, 1, 1)
		gr.Attach(label("miny:"), 0, 2, 1, 1)
		gr.Attach(label("maxy:"), 0, 3, 1, 1)
		gr.Attach(label("iter:"), 0, 4, 1, 1)
		gr.Attach(label("time:"), 0, 5, 1, 1)
		gr.Attach(ma.label.l, 1, 0, 1, 1)
		gr.Attach(ma.label.r, 1, 1, 1, 1)
		gr.Attach(ma.label.t, 1, 2, 1, 1)
		gr.Attach(ma.label.b, 1, 3, 1, 1)
		gr.Attach(ma.label.i, 1, 4, 1, 1)
		gr.Attach(ma.label.d, 1, 5, 1, 1)
	}
	ma.label.l.SetText(fmt.Sprintf("%6.4E", ma.minx))
	ma.label.r.SetText(fmt.Sprintf("%6.4E", ma.maxx))
	ma.label.t.SetText(fmt.Sprintf("%6.4E", ma.miny))
	ma.label.b.SetText(fmt.Sprintf("%6.4E", ma.maxy))
	ma.label.i.SetText(fmt.Sprintf("%d", ma.iter))
	ma.label.d.SetText(fmt.Sprintf("%v", ma.lastDuration))
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
	ma.updateLabels(gr)

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
		ma.updateLabels(nil)
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
		e := &gdk.EventScroll{ev}
		delta := e.DeltaY()
		if delta > 0.5 {
			delta = 0.5
		}
		delta *= (zw / 5.0)
		nzw := zw
		switch e.Direction() {
		case gdk.SCROLL_UP:
			nzw -= delta
		case gdk.SCROLL_DOWN:
			nzw += delta
		}

		// We want the screen to canvas translated coordinate be the same before and after the zoom.
		// This means: ominx + EX * osx = nminx + EX * nsx  (o-prefix is old, n is new) after some
		// algebra we get this:
		cx = cx - zw / 2.0 + nzw / 2.0 + e.X() * (zw - nzw) / float64(ma.w - 1)
		cy = cy - zw / 2.0 + nzw / 2.0 + e.Y() * (zw - nzw) / float64(ma.h - 1)
		zw = nzw
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
	runtime.LockOSThread()
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
