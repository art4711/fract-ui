# Simple Mandelbrot #

This started as an experiment to build a ui with gotk3 and ended as a
testbed for implementing some functionality in github.com/andlabs/ui.

There are two programs. `main-gtk.go` renders the ui with gtk.
`main-ui.go` renders the ui with ui. They attempt to have the same
functionality, but currently the colors are flipped for the gtk
version because of endianness. The most correct way to fix that should
be to implement the missing cairo functions in gotk3 and use cairo to
manage the pixmap instead of using gdk.

Also, the gtk version supports scroll wheel on top of the fractal,
while the ui version doesn't handle scroll wheel yet.

To zoom in the ui version, left-click to zoom in, right-click to zoom
out. The zoom SHOULD keep the point on the fractal under the mouse in
the same place before and after the zoom (it's actually a good way to
test if the libraries correctly translate corrdinates for the mouse).

## dependency ##

The two dependencies to make it work (non-gtk version) are those two
branches:

https://github.com/art4711/ui/tree/draw-pixmap
https://github.com/art4711/libui/tree/draw-pixmap