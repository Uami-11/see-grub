package renderer

import (
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

type Background struct {
	Image  *ebiten.Image
	Color  color.RGBA
	Width  int
	Height int
}

func LoadBackground(themeDir, imagePath, colorStr string) (*Background, error) {
	bg := &Background{
		Color: FallbackColor(colorStr, ColorBlack),
	}

	if imagePath == "" {
		return bg, nil
	}

	resolved := resolvePath(themeDir, imagePath)
	img, w, h, err := loadPNG(resolved)
	if err != nil {
		return bg, fmt.Errorf("desktop-image: %w", err)
	}

	bg.Image = img
	bg.Width = w
	bg.Height = h
	return bg, nil
}

func (bg *Background) Draw(dst *ebiten.Image) {
	if bg.Image == nil {
		dst.Fill(bg.Color)
		return
	}

	w, h := dst.Bounds().Dx(), dst.Bounds().Dy()
	srcW, srcH := bg.Image.Bounds().Dx(), bg.Image.Bounds().Dy()

	op := &ebiten.DrawImageOptions{}

	scaleX := float64(w) / float64(srcW)
	scaleY := float64(h) / float64(srcH)
	op.GeoM.Scale(scaleX, scaleY)

	dst.DrawImage(bg.Image, op)
}

type PixmapStyle struct {
	NW, N, NE *ebiten.Image
	W, C, E   *ebiten.Image
	SW, S, SE *ebiten.Image

	CornerW int
	CornerH int
}

func LoadPixmapStyle(themeDir, pattern string) (*PixmapStyle, error) {
	resolved := resolvePath(themeDir, pattern)

	matches, err := filepath.Glob(resolved)
	if err != nil || len(matches) == 0 {
		return nil, fmt.Errorf("pixmap style %q matched no files", pattern)
	}

	sort.Strings(matches)

	ps := &PixmapStyle{}

	for _, path := range matches {
		base := filepath.Base(path)
		name := strings.TrimSuffix(base, filepath.Ext(base))

		idx := strings.LastIndex(name, "_")
		if idx < 0 {
			continue
		}
		suffix := name[idx+1:]

		img, _, _, err := loadPNG(path)
		if err != nil {
			continue // skip missing/corrupt slices
		}

		switch suffix {
		case "nw":
			ps.NW = img
		case "n":
			ps.N = img
		case "ne":
			ps.NE = img
		case "w":
			ps.W = img
		case "c":
			ps.C = img
		case "e":
			ps.E = img
		case "sw":
			ps.SW = img
		case "s":
			ps.S = img
		case "se":
			ps.SE = img
		}
	}

	// Derive corner size from the NW image if available.
	if ps.NW != nil {
		ps.CornerW = ps.NW.Bounds().Dx()
		ps.CornerH = ps.NW.Bounds().Dy()
	}

	return ps, nil
}

func (ps *PixmapStyle) Draw(dst *ebiten.Image, r Rect) {
	if ps == nil {
		return
	}

	cw := ps.CornerW
	ch := ps.CornerH

	drawScaled := func(img *ebiten.Image, x, y, dstW, dstH int) {
		if img == nil || dstW <= 0 || dstH <= 0 {
			return
		}
		sx := float64(dstW) / float64(img.Bounds().Dx())
		sy := float64(dstH) / float64(img.Bounds().Dy())
		drawSlice(dst, img, x, y, sx, sy)
	}

	// Corners at native corner size
	drawScaled(ps.NW, r.X, r.Y, cw, ch)
	drawScaled(ps.NE, r.X+r.W-cw, r.Y, cw, ch)
	drawScaled(ps.SW, r.X, r.Y+r.H-ch, cw, ch)
	drawScaled(ps.SE, r.X+r.W-cw, r.Y+r.H-ch, cw, ch)

	// Edges stretch between corners
	drawScaled(ps.N, r.X+cw, r.Y, r.W-cw*2, ch)
	drawScaled(ps.S, r.X+cw, r.Y+r.H-ch, r.W-cw*2, ch)
	drawScaled(ps.W, r.X, r.Y+ch, cw, r.H-ch*2)
	drawScaled(ps.E, r.X+r.W-cw, r.Y+ch, cw, r.H-ch*2)

	// Center fills remaining area
	drawScaled(ps.C, r.X+cw, r.Y+ch, r.W-cw*2, r.H-ch*2)
}

func drawSlice(dst *ebiten.Image, img *ebiten.Image, x, y int, scaleX, scaleY float64) {
	if img == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scaleX, scaleY)
	op.GeoM.Translate(float64(x), float64(y))
	dst.DrawImage(img, op)
}

func loadPNG(path string) (*ebiten.Image, int, int, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	img, format, err := image.Decode(f)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("decode: %w", err)
	}
	if format != "png" {
		return nil, 0, 0, fmt.Errorf("expected PNG, got %q", format)
	}

	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	return ebiten.NewImageFromImage(img), w, h, nil
}

func resolvePath(themeDir, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(themeDir, path)
}

func imageWidth(img *ebiten.Image) int {
	if img == nil {
		return 1
	}
	w := img.Bounds().Dx()
	if w == 0 {
		return 1
	}
	return w
}

func imageHeight(img *ebiten.Image) int {
	if img == nil {
		return 1
	}
	h := img.Bounds().Dy()
	if h == 0 {
		return 1
	}
	return h
}
