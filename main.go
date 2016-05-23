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
     <object class="GtkGrid" id="labels">
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

	ma := gim.Newma()

	builder.ConnectSignals(map[string]interface{}{
		"resize": func(da *gtk.DrawingArea, p uintptr) {
			rect := gdk.WrapRectangle(p)
			allocpb(rect.GetWidth(), rect.GetHeight())
		},
		"drawArea": func(da *gtk.DrawingArea, cr *cairo.Context) {
			ma.Redraw(cx, cy, zw, pb)
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
	ma.PopulateLabels(&labelPopulator{gr: gri.(*gtk.Grid)})

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
