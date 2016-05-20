package main

import (
	"github.com/gotk3/gotk3/cairo"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"log"
//	"math/cmplx"
	"fmt"
	"math"
	"time"
	"sync"
	"runtime"
	"reflect"
	"strings"
)

type ma struct {
	Maxx float64 `dl:"%8.4E"`
	Maxy float64 `dl:"%8.4E"`

	Minx float64 `dl:"%8.4E"`
	Miny float64 `dl:"%8.4E"`

	sx float64
	sy float64

	Iter int `dl:"%d"`

	LastDuration time.Duration `dl:"%v,time"`

	dl dataLabels
}

func newma() *ma {
	ma := &ma{}
	return ma
}

var palette = [...][3]float64{
	{ 1.00, 0.00, 0.00 },
	{ 1.00, 1.00, 0.00 },
	{ 0.00, 1.00, 1.00 },
}

var log_escape = math.Log(2)

func getColor(abs float64, i int) (byte, byte, byte) {
	mu := float64(i + 1) - math.Log(math.Log(abs)) / log_escape
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

func (ma *ma)redrawRange(starty, endy, nc, rs, w int, px []byte, wg *sync.WaitGroup) {
	for y := starty; y < endy; y++ {
		cy := ma.Miny + float64(y) * ma.sy
		for x := 0; x < w; x++ {
			cx := ma.Minx + float64(x) * ma.sx
			o := y * rs + x * nc

			c := complex(cx, cy)
			z := c
			px[o], px[o + 1], px[o +2] = 0, 0, 0
			for i := 0; i < ma.Iter; i++ {
				re, im := real(z), imag(z)
				l := re * re + im * im
				if l > 4.0 {
					px[o], px[o + 1], px[o + 2] = getColor(l, i)
					break
				}
				z = z * z + c
			}
		}
	}
	wg.Done()
}

func (ma *ma)redraw(cx, cy, zw float64, pb *gdk.Pixbuf) {
	w := pb.GetWidth()
	h := pb.GetHeight()

	ma.Minx = cx - (zw / 2)
	ma.Maxx = cx + (zw / 2)
	ma.Miny = cy - ((ma.Maxx - ma.Minx) * float64(h) / float64(w) / 2)
	ma.Maxy = cy + ((ma.Maxx - ma.Minx) * float64(h) / float64(w) / 2)

	ma.sx = (ma.Maxx - ma.Minx)/float64(w - 1)
	ma.sy = (ma.Maxy - ma.Miny)/float64(h - 1)

	// http://math.stackexchange.com/a/30560
	ma.Iter = int(math.Sqrt(math.Abs(2.0 * math.Sqrt(math.Abs(1 - math.Sqrt(5.0 / zw))))) * 66.5)

	nc := pb.GetNChannels()
	rs := pb.GetRowstride()
	px := pb.GetPixels()

	startt := time.Now()

	var wg sync.WaitGroup

	steps := runtime.NumCPU()
	for i := 0; i < steps; i++ {
		wg.Add(1)
		go ma.redrawRange(i * h / steps, (i + 1) * h / steps, nc, rs, w, px, &wg)
	}

	wg.Wait()
	ma.LastDuration = time.Since(startt)
}

type datalabel struct {
	name string
	fmt string
	keyLabel *gtk.Label
	valLabel *gtk.Label
}

type dataLabels struct {
	labels []datalabel
}

func (dl *dataLabels)populate(src interface{}, gr *gtk.Grid) {
	l := func(s string) *gtk.Label {
		label, err := gtk.LabelNew(s)
		if err != nil {
			log.Fatal(err)
		}
		label.SetWidthChars(10)
		return label
	}

	srcv := reflect.ValueOf(src)
	srct := srcv.Type()

	for i := 0; i < srct.NumField(); i++ {
		ft := srct.Field(i)
		tags := strings.SplitN(ft.Tag.Get("dl"), ",", 2)
		if tags[0] == "" {
			continue
		}

		ln := ft.Name
		if len(tags) == 2 {
			ln = tags[1]
		}
		dl.labels = append(dl.labels, datalabel{ fmt: tags[0], name: ft.Name, keyLabel: l(ln), valLabel: l("") })
	}

	for i := range dl.labels {
		gr.Attach(dl.labels[i].keyLabel, 0, i, 1, 1)
		gr.Attach(dl.labels[i].valLabel, 1, i, 1, 1)		
	}
}

func (dl dataLabels)update(obj interface{}) {
	v := reflect.ValueOf(obj)
	
	for _, l := range dl.labels {
		l.valLabel.SetText(fmt.Sprintf(l.fmt, v.FieldByName(l.name).Interface()))
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

	var pb *gdk.Pixbuf

	allocpb := func(nw, nh int) {
		s := nw
		if s > nh {
			s = nh
		}
		pb, err = gdk.PixbufNew(gdk.COLORSPACE_RGB, false, 8, s, s)
		if err != nil {
			log.Fatal("pixbuf: ", err)
		}
	}
	allocpb(256, 256)

	builder.ConnectSignals(map[string]interface{}{
		"resize": func(da *gtk.DrawingArea, p uintptr) {
			rect := gdk.WrapRectangle(p)
			allocpb(rect.GetWidth(), rect.GetHeight())
		},
		"drawArea": func(da *gtk.DrawingArea, cr *cairo.Context) {
			ma.redraw(cx, cy, zw, pb)
			ma.dl.update(*ma)
			gtk.GdkCairoSetSourcePixBuf(cr, pb, 0, 0)
			cr.Paint()
		},
		"moveTo": func(win *gtk.Window, ev *gdk.Event) {
			e := &gdk.EventButton{ev}
			cx = cx - (zw / 2) + e.X() * zw / float64(pb.GetWidth() - 1)
			cy = cy - (zw / 2) + e.Y() * zw / float64(pb.GetHeight() - 1)		// assumes square pb (should be w?)
			eb.QueueDraw()
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
			cx += delta * (0.5 - e.X() / float64(pb.GetWidth() - 1))
			cy += delta * (0.5 - e.Y() / float64(pb.GetHeight() - 1)) // wrong - works only on square pb.
			zw += delta
			eb.QueueDraw()
		},
	})

	gri, err := builder.GetObject("labels")
	if err != nil {
		log.Fatal(err)
	}
	gr := gri.(*gtk.Grid)

	ma.dl.populate(*ma, gr)

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
