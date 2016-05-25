package main

import (
	"github.com/gotk3/gotk3/cairo"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"log"
	"runtime"
	"github.com/art4711/fract-ui/gim"
	"math"
	"time"
	"unsafe"
)

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
      <object class="GtkBox">
        <property name="orientation">vertical</property>
        <child>
          <object class="GtkGrid" id="controlLabels" />
        </child>

        <child>
          <object class="GtkGrid" id="imageLabels" />
        </child>
      </object>
    </child>
  </object>
</interface>
`

type labelPopulator struct {
	gr *gtk.Grid
	count int
}

func (lp *labelPopulator)AddKV(key string, kw, vw int) (gim.Label, gim.Label) {
	l := func(s string, w int) *gtk.Label {
		label, err := gtk.LabelNew(s)
		if err != nil {
			log.Fatal(err)
		}
		label.SetWidthChars(w)
		return label
	}

	kl := l(key, kw)
	vl := l("", vw)
	lp.gr.Attach(kl, 0, lp.count, 1, 1)
	lp.gr.Attach(vl, 1, lp.count, 1, 1)
	lp.count++
	return kl, vl
}

type pixbufWrap struct {
	pb *gdk.Pixbuf
}

func (pw pixbufWrap)GetWidth() int {
	return pw.pb.GetWidth()
}

func (pw pixbufWrap)GetHeight() int {
	return pw.pb.GetHeight()
}

func (pw pixbufWrap)GetRowstride() int {
	return pw.pb.GetRowstride() / 4
}

func (pw pixbufWrap)GetPixels() []uint32 {
	h := pw.GetHeight()
	rs := pw.GetRowstride()
	px := pw.pb.GetPixels()
	ptr := unsafe.Pointer(&px[0])
	arrsz := (h * rs)
	arrp := (*[1000000000]uint32)(ptr)
	return arrp[:arrsz]
}

type drawControl struct {
	Cx float64 `dl:"%8.4E"`
	Cy float64 `dl:"%8.4E"`
	Zw float64 `dl:"%8.4E"`

	Pxw float64 `dl:"%8.4E"`
	Mpxw float64 `dl:"%8.4E"`
	Mpxh float64 `dl:"%8.4E"`

	DrawTime time.Duration `dl:"%v"`

	pb *gdk.Pixbuf
	dr gim.Drawer

	dl gim.DataLabels
}

func (dc *drawControl)allocpb(nw, nh int) {
	// we enforce squareness for now
	s := nw
	if s > nh {
		s = nh
	}
	pb, err := gdk.PixbufNew(gdk.COLORSPACE_RGB, true, 8, s, s)
	if err != nil {
		log.Fatal("pixbuf: ", err)
	}
	dc.pb = pb
}

func (dc *drawControl)resize(da *gtk.DrawingArea, p uintptr) {
	rect := gdk.WrapRectangle(p)
	dc.allocpb(rect.GetWidth(), rect.GetHeight())
}

func (dc *drawControl)drawArea(da *gtk.DrawingArea, cr *cairo.Context) {
	st := time.Now()
	dc.dr.Redraw(dc.Cx, dc.Cy, dc.Zw, pixbufWrap{dc.pb})
	gtk.GdkCairoSetSourcePixBuf(cr, dc.pb, 0, 0)
	cr.Paint()
	dc.DrawTime = time.Since(st)
	dc.dl.Update(*dc)		// maybe not here?
}

func (dc *drawControl)moveTo(win *gtk.Window, ev *gdk.Event) {
	e := &gdk.EventButton{ev}
	dc.Cx = dc.Cx - (dc.Zw / 2) + e.X() * dc.Zw / float64(dc.pb.GetWidth() - 1)
	dc.Cy = dc.Cy - (dc.Zw / 2) + e.Y() * dc.Zw / float64(dc.pb.GetHeight() - 1)		// assumes square pb
	win.QueueDraw()
}

func (dc *drawControl)zoomTo(win *gtk.Window, ev *gdk.Event) {
	e := &gdk.EventScroll{ev}
	delta := e.DeltaY()
	if delta > 0.5 {
		delta = 0.5
	}
	delta *= (dc.Zw / 1.0)

	switch e.Direction() {
	case gdk.SCROLL_UP:
		delta = -delta
	case gdk.SCROLL_DOWN:
		// nothing
	default:
		delta = 0
	}

	// We want the screen to canvas translated coordinate be the same before and after the zoom.
	ncx := dc.Cx + delta * (0.5 - e.X() / float64(dc.pb.GetWidth() - 1))
	ncy := dc.Cy + delta * (0.5 - e.Y() / float64(dc.pb.GetHeight() - 1)) // assumes square pb
	nzw := dc.Zw + delta

	pxw := nzw / float64(dc.pb.GetWidth())		// pixel width
	mpxw := math.Abs(math.Nextafter(ncx, 0.0) - ncx)	// representable pixel width
	mpxh := math.Abs(math.Nextafter(ncy, 0.0) - ncy)	// representable pixel height

	if (delta < 0.0) && (pxw < (mpxw * 8.0) || pxw < (mpxh * 8.0)) {
		/*
		 * At high enough zoom levels we can no longer represent the numbers correctly enough.
		 * We calculate the width of one pixel (zw / width in pixels) and compare that to the
		 * precision we can iterate over floating point numbers at these coordinates. If we
		 * hit the limit, we no longer
		 * allow the zoom.
		 */
		return
	}

	dc.Cx = ncx
	dc.Cy = ncy
	dc.Zw = nzw
	dc.Pxw = pxw
	dc.Mpxw = mpxw
	dc.Mpxh = mpxh

	win.QueueDraw()
}

func buildWidgets() gtk.IWidget {
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

	dc := &drawControl{ Cx : -0.5, Cy: 0.0, Zw: 3.0, dr: gim.Newma() }
	dc.allocpb(256, 256)

	builder.ConnectSignals(map[string]interface{}{
		"resize": dc.resize,
		"drawArea": dc.drawArea,
		"moveTo": dc.moveTo,
		"zoomTo": dc.zoomTo,
	})

	gri, err := builder.GetObject("imageLabels")
	if err != nil {
		log.Fatal(err)
	}
	dc.dr.PopulateLabels(&labelPopulator{gr: gri.(*gtk.Grid)})

	gri, err = builder.GetObject("controlLabels")
	if err != nil {
		log.Fatal(err)
	}
	dc.dl.Populate(*dc, &labelPopulator{gr: gri.(*gtk.Grid)})

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

	win.Add(buildWidgets())
	win.ShowAll()

	gtk.Main()
}
