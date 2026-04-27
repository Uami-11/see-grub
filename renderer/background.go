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

	// This border was created by splitting a 565×233 image into a 3×3 grid.
	// Each piece is ~188×77 or ~188×78. The correct rendering scales all
	// pieces uniformly to fit the item rect, preserving the 3×3 layout.
	//
	// Natural assembled size derived from the split tool:
	const naturalW = 565
	const naturalH = 233

	// Each cell in the 3×3 grid
	// col widths: 188, 189, 188 (left, center, right)
	// row heights: 78, 77, 78   (top, middle, bottom)
	cellW := [3]int{188, 189, 188}
	cellH := [3]int{78, 77, 78}

	// Scale factors to map natural size → actual item rect
	scaleX := float64(r.W) / float64(naturalW)
	scaleY := float64(r.H) / float64(naturalH)

	// Scaled cell sizes
	w0 := int(float64(cellW[0]) * scaleX)
	w1 := int(float64(cellW[1]) * scaleX)
	w2 := r.W - w0 - w1

	h0 := int(float64(cellH[0]) * scaleY)
	h1 := int(float64(cellH[1]) * scaleY)
	h2 := r.H - h0 - h1

	// X positions of the 3 columns
	x0 := r.X
	x1 := r.X + w0
	x2 := r.X + w0 + w1

	// Y positions of the 3 rows
	y0 := r.Y
	y1 := r.Y + h0
	y2 := r.Y + h0 + h1

	// Scale each piece from its natural cell size to the scaled cell size
	drawScaled := func(img *ebiten.Image, x, y, dstW, dstH int) {
		if img == nil || dstW <= 0 || dstH <= 0 {
			return
		}
		sx := float64(dstW) / float64(img.Bounds().Dx())
		sy := float64(dstH) / float64(img.Bounds().Dy())
		drawSlice(dst, img, x, y, sx, sy)
	}

	// Row 0 (top): NW, N, NE
	drawScaled(ps.NW, x0, y0, w0, h0)
	drawScaled(ps.N, x1, y0, w1, h0)
	drawScaled(ps.NE, x2, y0, w2, h0)

	// Row 1 (middle): W, C, E
	drawScaled(ps.W, x0, y1, w0, h1)
	drawScaled(ps.C, x1, y1, w1, h1)
	drawScaled(ps.E, x2, y1, w2, h1)

	// Row 2 (bottom): SW, S, SE
	drawScaled(ps.SW, x0, y2, w0, h2)
	drawScaled(ps.S, x1, y2, w1, h2)
	drawScaled(ps.SE, x2, y2, w2, h2)
}

// func max(vals ...int) int {
// 	m := 0
// 	for _, v := range vals {
// 		if v > m {
// 			m = v
// 		}
// 	}
// 	return m
// }

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
