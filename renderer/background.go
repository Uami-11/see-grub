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

	if cw == 0 || ch == 0 {
		for _, img := range []*ebiten.Image{ps.NW, ps.NE, ps.SW, ps.SE} {
			if img != nil {
				cw = img.Bounds().Dx()
				ch = img.Bounds().Dy()
				break
			}
		}
	}

	innerW := r.W - cw*2
	innerH := r.H - ch*2

	drawSlice(dst, ps.NW, r.X, r.Y, 1, 1)
	drawSlice(dst, ps.NE, r.X+r.W-cw, r.Y, 1, 1)
	drawSlice(dst, ps.SW, r.X, r.Y+r.H-ch, 1, 1)
	drawSlice(dst, ps.SE, r.X+r.W-cw, r.Y+r.H-ch, 1, 1)

	if innerW > 0 {
		scaleX := float64(innerW) / float64(imageWidth(ps.N))
		drawSlice(dst, ps.N, r.X+cw, r.Y, scaleX, 1)
		drawSlice(dst, ps.S, r.X+cw, r.Y+r.H-ch, scaleX, 1)
	}

	if innerH > 0 {
		scaleY := float64(innerH) / float64(imageHeight(ps.W))
		drawSlice(dst, ps.W, r.X, r.Y+ch, 1, scaleY)
		drawSlice(dst, ps.E, r.X+r.W-cw, r.Y+ch, 1, scaleY)
	}

	if innerW > 0 && innerH > 0 {
		scaleX := float64(innerW) / float64(imageWidth(ps.C))
		scaleY := float64(innerH) / float64(imageHeight(ps.C))
		drawSlice(dst, ps.C, r.X+cw, r.Y+ch, scaleX, scaleY)
	}
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
