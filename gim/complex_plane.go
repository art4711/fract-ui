package gim

import (
	"math"
	"time"
	"sync"
	"runtime"
	"log"
)

type Drawer interface {
	PopulateLabels(lp LabelPopulator)
	Redraw(cx, cy, zw float64, pb Pixbuf)
}

type ma struct {
	Iter int `dl:"%d"`
	LastDuration time.Duration `dl:"%v,time"`

	dl DataLabels
}

func Newma() *ma {
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

func colorAt(c complex128, iter int) (byte, byte, byte) {
	z := c
	for i := 0; i < iter; i++ {
		re, im := real(z), imag(z)
		l := re * re + im * im
		if l > 4.0 {
			return getColor(l, i)
		}
		z = z * z + c
	}
	return 0, 0, 0
}

func (ma *ma)Redraw(cx, cy, zw float64, pb Pixbuf) {
	w := pb.GetWidth()
	h := pb.GetHeight()
	nc := pb.GetNChannels()
	rs := pb.GetRowstride()
	px := pb.GetPixels()

	aspect := float64(h) / float64(w)

	sx := zw / float64(w - 1)
	sy := zw * aspect / float64(h - 1)

	// http://math.stackexchange.com/a/30560
	ma.Iter = int(math.Sqrt(math.Abs(2.0 * math.Sqrt(math.Abs(1 - math.Sqrt(5.0 / zw))))) * 66.5)

	startt := time.Now()

	var wg sync.WaitGroup

	steps := runtime.NumCPU()
	for i := 0; i < steps; i++ {
		wg.Add(1)
		go func(starty, endy int) {
			for y := starty; y < endy; y++ {
				ci := cy - (zw * aspect / 2) + float64(y) * sy
				for x := 0; x < w; x++ {
					cr := cx - (zw / 2) + float64(x) * sx
					o := y * rs + x * nc
					px[o], px[o + 1], px[o + 2] = colorAt(complex(cr, ci), ma.Iter)
					px[o + 3] = 255
				}
			}
			wg.Done()
		}(i * h / steps, (i + 1) * h / steps)
	}

	wg.Wait()
	ma.LastDuration = time.Since(startt)
	log.Print(ma.LastDuration)
	ma.dl.Update(*ma)
}

func (ma *ma)PopulateLabels(lp LabelPopulator) {
	ma.dl.Populate(*ma, lp)
}