package main

import (
	"github.com/gotk3/gotk3/cairo"
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/gtk"
	"log"
	"runtime"
	"gim/gim"
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
     <object class="GtkGrid" id="controlLabels">
     </object>
   </child>

   <child>
     <object class="GtkGrid" id="imageLabels">
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

type drawControl struct {
	cx, cy, zw float64
	pb *gdk.Pixbuf
	dr gim.Drawer
}

func (dc *drawControl)allocpb(nw, nh int) {
	// we enforce squareness for now
	s := nw
	if s > nh {
		s = nh
	}
	pb, err := gdk.PixbufNew(gdk.COLORSPACE_RGB, false, 8, s, s)
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
	dc.dr.Redraw(dc.cx, dc.cy, dc.zw, dc.pb)
	gtk.GdkCairoSetSourcePixBuf(cr, dc.pb, 0, 0)
	cr.Paint()
}

func (dc *drawControl)moveTo(win *gtk.Window, ev *gdk.Event) {
	e := &gdk.EventButton{ev}
	dc.cx = dc.cx - (dc.zw / 2) + e.X() * dc.zw / float64(dc.pb.GetWidth() - 1)
	dc.cy = dc.cy - (dc.zw / 2) + e.Y() * dc.zw / float64(dc.pb.GetHeight() - 1)		// assumes square pb
	win.QueueDraw()
}

func (dc *drawControl)zoomTo(win *gtk.Window, ev *gdk.Event) {
	e := &gdk.EventScroll{ev}
	delta := e.DeltaY()
	if delta > 0.5 {
		delta = 0.5
	}
	delta *= (dc.zw / 5.0)

	switch e.Direction() {
	case gdk.SCROLL_UP:
		delta = -delta
	case gdk.SCROLL_DOWN:
		// nothing
	default:
		delta = 0
	}

	// We want the screen to canvas translated coordinate be the same before and after the zoom.
	dc.cx += delta * (0.5 - e.X() / float64(dc.pb.GetWidth() - 1))
	dc.cy += delta * (0.5 - e.Y() / float64(dc.pb.GetHeight() - 1)) // assumes square pb
	dc.zw += delta
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

	dc := &drawControl{ cx : -0.5, cy: 0.0, zw: 3.0, dr: gim.Newma() }
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