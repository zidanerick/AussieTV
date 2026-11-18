// Command iconbuilder generates PNG favicons and a portable Windows ICO without
// external dependencies. The primitive geometry is the SVG brandmark rendered
// as raster shapes, with 3x3 supersampling for small icon sizes.
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

type vec struct{ x, y float64 }

func withinRounded(x, y, l, t, w, h, r float64) bool {
	if x < l || x > l+w || y < t || y > t+h {
		return false
	}
	cx := math.Max(l+r, math.Min(l+w-r, x))
	cy := math.Max(t+r, math.Min(t+h-r, y))
	dx := x - cx
	dy := y - cy
	return dx*dx+dy*dy <= r*r
}
func segment(x, y float64, a, b vec, width float64) bool {
	dx := b.x - a.x
	dy := b.y - a.y
	t := ((x-a.x)*dx + (y-a.y)*dy) / (dx*dx + dy*dy)
	t = math.Max(0, math.Min(1, t))
	xx := x - (a.x + t*dx)
	yy := y - (a.y + t*dy)
	return xx*xx+yy*yy <= width*width/4
}
func insideTriangle(x, y float64, a, b, c vec) bool {
	cross := func(p, q vec) float64 { return p.x*q.y - p.y*q.x }
	p := vec{x, y}
	aa := cross(vec{b.x - a.x, b.y - a.y}, vec{p.x - a.x, p.y - a.y})
	bb := cross(vec{c.x - b.x, c.y - b.y}, vec{p.x - b.x, p.y - b.y})
	cc := cross(vec{a.x - c.x, a.y - c.y}, vec{p.x - c.x, p.y - c.y})
	return (aa >= 0 && bb >= 0 && cc >= 0) || (aa <= 0 && bb <= 0 && cc <= 0)
}
func sample(x, y float64) color.RGBA {
	navy := color.RGBA{16, 41, 101, 255}
	aqua := color.RGBA{20, 226, 199, 255}
	gold := color.RGBA{255, 203, 64, 255}
	transparent := color.RGBA{}
	if segment(x, y, vec{36, 109}, vec{30, 119}, 12) || segment(x, y, vec{92, 109}, vec{98, 119}, 12) {
		transparent = navy
	}
	if segment(x, y, vec{56, 40}, vec{35, 18}, 10) || segment(x, y, vec{72, 40}, vec{93, 18}, 10) {
		transparent = navy
	}
	if math.Hypot(x-35, y-18) <= 8 || math.Hypot(x-93, y-18) <= 8 {
		transparent = navy
	}
	if withinRounded(x, y, 9, 35, 110, 80, 19) {
		transparent = navy
		// abstract arcing yellow sweep at the top-right of the screen
		if x > 68 && x < 111 && y > 38 && y < 66 {
			high := 48.0 - 9.0*math.Sin((x-68)/43*math.Pi*.75)
			low := 50.0 + 16.0*math.Pow(math.Max(0, (x-68)/43), 2)
			if y >= high && y <= low {
				transparent = gold
			}
		}
		// Play triangle occupies the readable centre of a 16px favicon.
		if insideTriangle(x, y, vec{43, 50}, vec{94, 78}, vec{43, 106}) {
			transparent = aqua
		}
	}
	return transparent
}
func raster(size int) *image.RGBA {
	im := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			var rr, gg, bb, aa int
			for sy := 0; sy < 3; sy++ {
				for sx := 0; sx < 3; sx++ {
					p := sample((float64(x)+(float64(sx)+.5)/3)*128/float64(size), (float64(y)+(float64(sy)+.5)/3)*128/float64(size))
					rr += int(p.R) * int(p.A)
					gg += int(p.G) * int(p.A)
					bb += int(p.B) * int(p.A)
					aa += int(p.A)
				}
			}
			if aa > 0 {
				im.SetRGBA(x, y, color.RGBA{uint8(rr / aa), uint8(gg / aa), uint8(bb / aa), uint8(aa / 9)})
			}
		}
	}
	return im
}
func encodePNG(im image.Image) ([]byte, error) {
	var b bytes.Buffer
	err := png.Encode(&b, im)
	return b.Bytes(), err
}
func writeICO(path string, sizes []int) error {
	imgs := make([][]byte, len(sizes))
	for i, s := range sizes {
		b, err := encodePNG(raster(s))
		if err != nil {
			return err
		}
		imgs[i] = b
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if err = binary.Write(f, binary.LittleEndian, uint16(0)); err != nil {
		return err
	}
	_ = binary.Write(f, binary.LittleEndian, uint16(1))
	_ = binary.Write(f, binary.LittleEndian, uint16(len(sizes)))
	offset := 6 + 16*len(sizes)
	for i, s := range sizes {
		side := uint8(s)
		if s == 256 {
			side = 0
		}
		_, _ = f.Write([]byte{side, side, 0, 0})
		_ = binary.Write(f, binary.LittleEndian, uint16(1))
		_ = binary.Write(f, binary.LittleEndian, uint16(32))
		_ = binary.Write(f, binary.LittleEndian, uint32(len(imgs[i])))
		_ = binary.Write(f, binary.LittleEndian, uint32(offset))
		offset += len(imgs[i])
	}
	for _, b := range imgs {
		if _, err = f.Write(b); err != nil {
			return err
		}
	}
	return nil
}
func main() {
	if err := os.MkdirAll("assets", 0755); err != nil {
		panic(err)
	}
	if err := os.MkdirAll("static", 0755); err != nil {
		panic(err)
	}
	for _, s := range []int{32, 192} {
		b, err := encodePNG(raster(s))
		if err != nil {
			panic(err)
		}
		path := filepath.Join("static", fmt.Sprintf("favicon-%d.png", s))
		if err = os.WriteFile(path, b, 0644); err != nil {
			panic(err)
		}
	}
	if err := writeICO(filepath.Join("assets", "AussieTV.ico"), []int{16, 24, 32, 48, 64, 128, 256}); err != nil {
		panic(err)
	}
	ico, err := os.ReadFile(filepath.Join("assets", "AussieTV.ico"))
	if err != nil {
		panic(err)
	}
	if err = os.WriteFile(filepath.Join("static", "favicon.ico"), ico, 0644); err != nil {
		panic(err)
	}
	if err = writeWindowsResource(filepath.Join("assets", "AussieTV.ico"), "rsrc_windows_amd64.syso"); err != nil {
		panic(err)
	}
	fmt.Println("Generated AussieTV favicons, ICO and embedded Windows executable resources")
}
