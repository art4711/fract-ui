package main

import (
	"github.com/andlabs/ui"
	"github.com/art4711/fract-ui/gim"
	"math"
	"time"
)

type labelPopulator struct {
	// XXX - maybe two boxes for each column?
	lb *ui.Box
}

func (lp *labelPopulator) AddKV(key string, kw, vw int) (gim.Label, gim.Label) {
	box := ui.NewHorizontalBox()
	kl := ui.NewLabel(key + ": ")
	vl := ui.NewLabel("")
	box.Append(kl, false)
	box.Append(vl, false)

	lp.lb.Append(box, false)
	return kl, vl
}

type drawControl struct {
	Cx float64 `dl:"%8.4E"`
	Cy float64 `dl:"%8.4E"`
	Zw float64 `dl:"%8.4E"`

	Pxw  float64 `dl:"%8.4E"`
	Mpxw float64 `dl:"%8.4E"`
	Mpxh float64 `dl:"%8.4E"`

	DrawTime time.Duration `dl:"%v"`

	image *ui.Image
	image_size int

	dr gim.Drawer

	dl gim.DataLabels
}

func (dc *drawControl) allocImg(w, h int) {
	s := w
	if s > h {
		s = h
	}
	if dc.image != nil && s == dc.image_size {
		return
	}
	dc.image = ui.NewImage(s, s)
	dc.image_size = s
}

/*
func (dc *drawControl)moveTo(win *gtk.Window, ev *gdk.Event) {
	e := &gdk.EventButton{ev}
	dc.Cx = dc.Cx - (dc.Zw / 2) + e.X() * dc.Zw / float64(dc.pb.GetWidth() - 1)
	dc.Cy = dc.Cy - (dc.Zw / 2) + e.Y() * dc.Zw / float64(dc.pb.GetHeight() - 1)		// assumes square pb
	win.QueueDraw()
}
*/

func (dc *drawControl) zoomAt(mx, my, delta float64, out bool) {
	b := dc.image.Bounds()
	h := b.Max.Y
	w := b.Max.X

	delta *= -dc.Zw
	if out {
		delta = -delta
	}

	// We want the screen to canvas translated coordinate be the same before and after the zoom.
	ncx := dc.Cx + delta*(0.5-mx/float64(w-1))
	ncy := dc.Cy + delta*(0.5-my/float64(h-1)) // assumes square pb
	nzw := dc.Zw + delta

	pxw := nzw / float64(w)      // pixel width
	mpxw := math.Abs(math.Nextafter(ncx, 0.0) - ncx) // representable pixel width
	mpxh := math.Abs(math.Nextafter(ncy, 0.0) - ncy) // representable pixel height

	if (delta < 0.0) && (pxw < (mpxw*8.0) || pxw < (mpxh*8.0)) {
		// At high enough zoom levels we can no longer represent the numbers correctly enough.
		// We calculate the width of one pixel (zw / width in pixels) and compare that to the
		// precision we can iterate over floating point numbers at these coordinates. If we
		// hit the limit, we no longer
		// allow the zoom.
		return
	}

	dc.Cx = ncx
	dc.Cy = ncy
	dc.Zw = nzw
	dc.Pxw = pxw
	dc.Mpxw = mpxw
	dc.Mpxh = mpxh
}

func (dc *drawControl) Draw(a *ui.Area, dp *ui.AreaDrawParams) {
	st := time.Now()

	dc.allocImg(int(dp.AreaWidth), int(dp.AreaHeight))
	dc.dr.Redraw(dc.Cx, dc.Cy, dc.Zw, dc.image)
	dp.Context.Image(0, 0, dc.image)

	dc.DrawTime = time.Since(st)
	dc.dl.Update(*dc) // maybe not here?
}

func (dc *drawControl) MouseEvent(a *ui.Area, me *ui.AreaMouseEvent) {
	if me.Up == 1 {
		dc.zoomAt(me.X, me.Y, 0.2, false)
		a.QueueRedrawAll()
	}
	if me.Up == 3 {
		dc.zoomAt(me.X, me.Y, 0.2, true)
		a.QueueRedrawAll()
	}
}

func (dc *drawControl) MouseCrossed(a *ui.Area, left bool) {
}

func (dc *drawControl) DragBroken(a *ui.Area) {
}

func (dc *drawControl) KeyEvent(a *ui.Area, ke *ui.AreaKeyEvent) bool {
	return false
}

var selections = []struct {
	name string
	dr func() gim.Drawer
}{
	{ "mandelbrot", gim.Newma },
	{ "cubed", gim.Newcu },
}

func main() {
	err := ui.Main(func() {
		dc := &drawControl{Cx: -0.5, Cy: 0.0, Zw: 3.0, dr: gim.Newma()}
//		dc.image.alloc(256, 256)

		mainbox := ui.NewHorizontalBox()

		area := ui.NewArea(dc)

		cb := ui.NewCombobox()
		for i := range selections {
			cb.Append(selections[i].name)
		}
		cb.SetSelected(0)
		cb.OnSelected(func(cb *ui.Combobox) {
			sel := selections[cb.Selected()]
			dc.dr = sel.dr()
			area.QueueRedrawAll()
		})

		group := ui.NewGroup("area") // The group is necessary for gtk to be less confused.
		group.SetChild(area)
		mainbox.Append(group, true)

		labelbox := ui.NewVerticalBox()
		labelbox.Append(cb, false)


		lp := &labelPopulator{lb: labelbox}

		dc.dr.PopulateLabels(lp)
		dc.dl.Populate(*dc, lp)
		mainbox.Append(labelbox, true)
		window := ui.NewWindow("asdf", 400, 400, false)
		window.SetChild(mainbox)
		window.OnClosing(func(*ui.Window) bool {
			ui.Quit()
			return true
		})
		window.Show()
	})
	if err != nil {
		panic(err)
	}
}
