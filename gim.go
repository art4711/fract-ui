package main

import (
	"github.com/gotk3/gotk3/cairo"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"log"
	"math/cmplx"
	"fmt"
	"math"
	"time"
	"sync"
	"runtime"
	"reflect"
)

type ma struct {
	pb *gdk.Pixbuf

	Maxx float64
	Maxy float64

	Minx float64
	Miny float64

	sx float64
	sy float64

	Iter int

	h int
	w int

	LastDuration time.Duration

	dl dataLabels
}

func newma() *ma {
	ma := &ma{}
	return ma
}

func (ma *ma)setCoords(cx float64, cy float64, zw float64) {
	ma.w = ma.pb.GetWidth()
	ma.h = ma.pb.GetHeight()

	ma.Minx = cx - (zw / 2)
	ma.Maxx = cx + (zw / 2)
	ma.Miny = cy - ((ma.Maxx - ma.Minx) * float64(ma.h) / float64(ma.w) / 2)
	ma.Maxy = cy + ((ma.Maxx - ma.Minx) * float64(ma.h) / float64(ma.w) / 2)

	ma.sx = (ma.Maxx - ma.Minx)/float64(ma.w - 1)
	ma.sy = (ma.Maxy - ma.Miny)/float64(ma.h - 1)

	// http://math.stackexchange.com/a/30560
	ma.Iter = int(math.Sqrt(math.Abs(2.0 * math.Sqrt(math.Abs(1 - math.Sqrt(5.0 / zw))))) * 66.5)
}

func (ma *ma)screenCoords(x float64, y float64) (float64, float64) {
	return (ma.Minx + x * ma.sx), (ma.Miny + y * ma.sy)
}

var palette = [...][3]float64{
	{ 1.00, 0.00, 0.00 },
	{ 1.00, 1.00, 0.00 },
	{ 0.00, 1.00, 1.00 },
}

var log_escape = math.Log(2)

func (ma *ma)getColor(z, c complex128, i int) (byte, byte, byte) {
	for extra := 0; extra < 3; extra++ {
		z = z * z + c
		i++
	}
	mu := float64(i + 1) - math.Log(math.Log(cmplx.Abs(z))) / log_escape
	mu /= 16
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
		cy := ma.Miny + float64(y) * ma.sy
		for x := 0; x < ma.w; x++ {
			cx := ma.Minx + float64(x) * ma.sx
			o := y * rs + x * nc

			c := complex(cx, cy)
			z := c
			px[o], px[o + 1], px[o +2] = 0, 0, 0
			for i := 0; i < ma.Iter; i++ {
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
	ma.LastDuration = time.Since(startt)
}

type dataLabels []struct {
	name string
	fmt string
	labelName string
	keyLabel *gtk.Label
	valLabel *gtk.Label
}

func (dl *dataLabels)populate(gr *gtk.Grid) {
	l := func(s string) *gtk.Label {
		label, err := gtk.LabelNew(s)
		if err != nil {
			log.Fatal(err)
		}
		label.SetWidthChars(10)
		return label
	}
	for i := range *dl {
		ln := (*dl)[i].labelName
		if ln == "" {
			ln = (*dl)[i].name
		}
		(*dl)[i].keyLabel = l(ln)
		gr.Attach((*dl)[i].keyLabel, 0, i, 1, 1)
		(*dl)[i].valLabel = l("")
		gr.Attach((*dl)[i].valLabel, 1, i, 1, 1)
	}
}

func (dl dataLabels)update(obj interface{}) {
	v := reflect.ValueOf(obj)
	
	for _, d := range dl {
		d.valLabel.SetText(fmt.Sprintf(d.fmt, v.FieldByName(d.name).Interface()))
	}
}

const build = `
<interface>
  <object class="GtkBox" id="everything">
    <property name="orientation">horizontal</property>
    <child>
      <object class="GtkEventBox" id="eb">
        <child>
          <object class="GtkDrawingArea">
            <signal name="draw" handler="drawArea" />
            <signal name="size-allocate" handler="resize" />
            <property name="width-request">256</property>
            <property name="height-request">256</property>
          </object>
        </child>
        <property name="events">GDK_SCROLL_MASK</property>
        <signal name="button_press_event" handler="moveTo" />
        <signal name="scroll-event" handler="zoomTo" />
      </object>
      <packing>
        <property name="expand">true</property>
      </packing>
    </child>

   <child>
     <object class="GtkGrid" id="labels">
     </object>
   </child>

  </object>
</interface>
`

func (ma *ma)buildWidgets() gtk.IWidget {
	builder, err := gtk.BuilderNew()
	if err != nil {
		log.Fatal(err)
	}
	err = builder.AddFromString(build)
	if err != nil {
		log.Fatal(err)
	}

	ebi, err := builder.GetObject("eb")
	if err != nil {
		log.Fatal(err)
	}
	eb := ebi.(*gtk.EventBox)

	// XXX - how do we set this property in the xml?
	eb.AddEvents(int(gdk.SCROLL_MASK))

	zw := 3.0
	cx := -0.5
	cy := 0.0

	redraw := func() {
		eb.QueueDraw()
	}

	allocpb := func(nw, nh int) {
		pb, err := gdk.PixbufNew(gdk.COLORSPACE_RGB, false, 8, nw, nh)
		if err != nil {
			log.Fatal("pixbuf: ", err)
		}
		ma.pb = pb
	}
	allocpb(256, 256)

	builder.ConnectSignals(map[string]interface{}{
		"resize": func(da *gtk.DrawingArea, p uintptr) {
			rect := gdk.WrapRectangle(p)
			allocpb(rect.GetWidth(), rect.GetHeight())
		},
		"drawArea": func(da *gtk.DrawingArea, cr *cairo.Context) {
			ma.setCoords(cx, cy, zw)
			ma.redraw()
			ma.dl.update(*ma)
			gtk.GdkCairoSetSourcePixBuf(cr, ma.pb, 0, 0)
			cr.Paint()
		},
		"moveTo": func(win *gtk.Window, ev *gdk.Event) {
			e := &gdk.EventButton{ev}
			cx, cy = ma.screenCoords(e.X(), e.Y())
			redraw()
		},
		"zoomTo": func(win *gtk.Window, ev *gdk.Event) {
			e := &gdk.EventScroll{ev}
			delta := e.DeltaY()
			if delta > 0.5 {
				delta = 0.5
			}
			delta *= (zw / 5.0)

			switch e.Direction() {
			case gdk.SCROLL_UP:
				delta = -delta
			case gdk.SCROLL_DOWN:
				// nothing
			default:
				delta = 0
			}

			// We want the screen to canvas translated coordinate be the same before and after the zoom.
			// This means: ominx + EX * osx = nminx + EX * nsx  (o-prefix is old, n is new) after some
			// algebra we get this:
			cx += delta * (0.5 - e.X() / float64(ma.w - 1))
			cy += delta * (0.5 - e.Y() / float64(ma.h - 1))
			zw += delta
			redraw()
		},
	})

	ma.dl = dataLabels{
		{ name: "Minx", fmt: "%8.4E" },
		{ name: "Maxx", fmt: "%8.4E" },
		{ name: "Miny", fmt: "%8.4E" },
		{ name: "Maxy", fmt: "%8.4E" },
		{ name: "Iter", fmt: "%d" },
		{ name: "LastDuration", fmt: "%v", labelName: "time" },
	}

	gri, err := builder.GetObject("labels")
	if err != nil {
		log.Fatal(err)
	}
	gr := gri.(*gtk.Grid)

	ma.dl.populate(gr)

	redraw()

	obj, err := builder.GetObject("everything")
	if err != nil {
		log.Fatal(err)
	}

	return obj.(gtk.IWidget)
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

	win.Add(ma.buildWidgets())
	win.ShowAll()

	gtk.Main()
}
