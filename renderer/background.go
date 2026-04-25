package renderer

import (
	"fmt"
	"image"
	"image/color"
	_ "image/png" // register PNG decoder
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// Background
// ---------------------------------------------------------------------------

// Background holds the loaded desktop image and fallback color.
// Either Image or Color is used — Image takes priority if both are set.
type Background struct {
	Image  *ebiten.Image // nil if no desktop-image was set or load failed
	Color  color.RGBA    // used when Image is nil
	Width  int
	Height int
}

// LoadBackground loads the desktop-image PNG from the theme directory.
// If the path is empty or the file fails to load, it falls back to
// desktop-color. Either way it always returns a usable Background.
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

// Draw fills the destination image with the background.
// If a background image is set, it is scaled to fill dst exactly.
// Otherwise the fallback color is used.
func (bg *Background) Draw(dst *ebiten.Image) {
	if bg.Image == nil {
		dst.Fill(bg.Color)
		return
	}

	w, h := dst.Bounds().Dx(), dst.Bounds().Dy()
	srcW, srcH := bg.Image.Bounds().Dx(), bg.Image.Bounds().Dy()

	op := &ebiten.DrawImageOptions{}

	// Scale the image to exactly fill the window.
	// For GRUB themes the background is always designed to match the
	// target resolution, so this is usually a 1:1 draw — but we scale
	// correctly in case the preview window is ever resized.
	scaleX := float64(w) / float64(srcW)
	scaleY := float64(h) / float64(srcH)
	op.GeoM.Scale(scaleX, scaleY)

	dst.DrawImage(bg.Image, op)
}

// ---------------------------------------------------------------------------
// Pixmap style (9-slice scaling)
// ---------------------------------------------------------------------------

// PixmapStyle holds the 9 slices of a GRUB pixmap style pattern.
// GRUB uses these for terminal-box borders and boot_menu item borders.
//
// A pixmap style is a glob like "terminal_box_*.png" that matches
// exactly 8 files named with the following suffixes:
//
//	_c.png  — center          (scales in both X and Y)
//	_e.png  — east edge       (scales in Y only)
//	_n.png  — north edge      (scales in X only)
//	_ne.png — north-east corner (no scaling)
//	_nw.png — north-west corner (no scaling)
//	_s.png  — south edge      (scales in X only)
//	_se.png — south-east corner (no scaling)
//	_sw.png — south-west corner (no scaling)
//	_w.png  — west edge       (scales in Y only)
type PixmapStyle struct {
	// The 9 slice images. Any can be nil if the file was missing.
	NW, N, NE *ebiten.Image
	W, C, E   *ebiten.Image
	SW, S, SE *ebiten.Image

	// Corner size derived from the NW corner image dimensions.
	// All corners are assumed to be the same size.
	CornerW int
	CornerH int
}

// LoadPixmapStyle loads a 9-slice pixmap style from a glob pattern.
// The pattern should be something like "assets/terminal_box_*.png".
// Files are matched by their suffix before the extension.
func LoadPixmapStyle(themeDir, pattern string) (*PixmapStyle, error) {
	resolved := resolvePath(themeDir, pattern)

	matches, err := filepath.Glob(resolved)
	if err != nil || len(matches) == 0 {
		return nil, fmt.Errorf("pixmap style %q matched no files", pattern)
	}

	// Sort so we process files in a consistent order.
	sort.Strings(matches)

	ps := &PixmapStyle{}

	for _, path := range matches {
		// Determine which slice this file is by its suffix.
		// e.g. "terminal_box_nw.png" → suffix "nw"
		base := filepath.Base(path)
		name := strings.TrimSuffix(base, filepath.Ext(base))

		// Find the last underscore to get the directional suffix.
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

// Draw renders the 9-slice pixmap style to fill the given Rect on dst.
// This is how GRUB draws terminal-box borders and menu item borders —
// corners are drawn at their natural size, edges are stretched to fill.
func (ps *PixmapStyle) Draw(dst *ebiten.Image, r Rect) {
	if ps == nil {
		return
	}

	cw := ps.CornerW
	ch := ps.CornerH

	// If we have no corner size info, try to derive it from whatever
	// corner image we do have.
	if cw == 0 || ch == 0 {
		for _, img := range []*ebiten.Image{ps.NW, ps.NE, ps.SW, ps.SE} {
			if img != nil {
				cw = img.Bounds().Dx()
				ch = img.Bounds().Dy()
				break
			}
		}
	}

	// Inner area dimensions (the stretchable region).
	innerW := r.W - cw*2
	innerH := r.H - ch*2

	// --- Corners (no scaling) ---
	drawSlice(dst, ps.NW, r.X, r.Y, 1, 1)
	drawSlice(dst, ps.NE, r.X+r.W-cw, r.Y, 1, 1)
	drawSlice(dst, ps.SW, r.X, r.Y+r.H-ch, 1, 1)
	drawSlice(dst, ps.SE, r.X+r.W-cw, r.Y+r.H-ch, 1, 1)

	// --- Edges (scaled in one axis) ---
	// North and south edges scale horizontally.
	if innerW > 0 {
		scaleX := float64(innerW) / float64(imageWidth(ps.N))
		drawSlice(dst, ps.N, r.X+cw, r.Y, scaleX, 1)
		drawSlice(dst, ps.S, r.X+cw, r.Y+r.H-ch, scaleX, 1)
	}

	// West and east edges scale vertically.
	if innerH > 0 {
		scaleY := float64(innerH) / float64(imageHeight(ps.W))
		drawSlice(dst, ps.W, r.X, r.Y+ch, 1, scaleY)
		drawSlice(dst, ps.E, r.X+r.W-cw, r.Y+ch, 1, scaleY)
	}

	// --- Center (scales in both axes) ---
	if innerW > 0 && innerH > 0 {
		scaleX := float64(innerW) / float64(imageWidth(ps.C))
		scaleY := float64(innerH) / float64(imageHeight(ps.C))
		drawSlice(dst, ps.C, r.X+cw, r.Y+ch, scaleX, scaleY)
	}
}

// drawSlice draws a single slice image at (x, y) with the given scale.
// If img is nil, it does nothing — missing slices are silently skipped.
func drawSlice(dst *ebiten.Image, img *ebiten.Image, x, y int, scaleX, scaleY float64) {
	if img == nil {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(scaleX, scaleY)
	op.GeoM.Translate(float64(x), float64(y))
	dst.DrawImage(img, op)
}

// ---------------------------------------------------------------------------
// PNG loader
// ---------------------------------------------------------------------------

// loadPNG loads a PNG file from disk and returns an ebiten.Image.
// Also returns the natural width and height of the image.
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

// ---------------------------------------------------------------------------
// Path helpers (package-level, used by other renderer files too)
// ---------------------------------------------------------------------------

// resolvePath resolves a theme-relative path to an absolute path.
// If path is already absolute it is returned unchanged.
func resolvePath(themeDir, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(themeDir, path)
}

// imageWidth safely returns the width of an ebiten.Image, or 1 if nil.
// The "or 1" prevents divide-by-zero in scale calculations.
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

// imageHeight safely returns the height of an ebiten.Image, or 1 if nil.
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
