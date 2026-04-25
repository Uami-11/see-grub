package renderer

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// ---------------------------------------------------------------------------
// PF2 font structures
// ---------------------------------------------------------------------------

// PF2Font holds everything we extracted from a .pf2 file that the
// renderer needs to draw text.
type PF2Font struct {
	Name      string          // the PFF2NAME value, e.g. "Isabella Regular 72"
	FilePath  string          // absolute path to the .pf2 file
	PointSize int             // PTSZ section value
	Glyphs    map[rune]*Glyph // codepoint → glyph data
	Ascent    int             // pixels above baseline
	Descent   int             // pixels below baseline
}

// Glyph holds the bitmap and metrics for a single character.
type Glyph struct {
	Width       int           // bitmap width in pixels
	Height      int           // bitmap height in pixels
	XOffset     int           // horizontal offset when drawing (can be negative)
	YOffset     int           // vertical offset from baseline (can be negative)
	DeviceWidth int           // how far to advance the cursor after this glyph
	Image       *ebiten.Image // the rendered glyph as an Ebitengine image
}

// ---------------------------------------------------------------------------
// Font registry
// ---------------------------------------------------------------------------

// FontRegistry maps PFF2NAME strings to loaded PF2Font instances.
// This is built once at startup by scanning the theme directory.
type FontRegistry struct {
	fonts map[string]*PF2Font // key is the PFF2NAME string
}

// NewFontRegistry scans themeDir for all .pf2 files, reads their embedded
// names, and returns a registry ready for lookup.
// Fonts are loaded lazily — we read names now, bitmaps on first use.
func NewFontRegistry(themeDir string) (*FontRegistry, []error) {
	reg := &FontRegistry{
		fonts: make(map[string]*PF2Font),
	}

	pattern := filepath.Join(themeDir, "*.pf2")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return reg, []error{fmt.Errorf("scanning for .pf2 files: %w", err)}
	}

	var errs []error
	for _, path := range matches {
		font, err := loadPF2(path)
		if err != nil {
			errs = append(errs, fmt.Errorf("loading %s: %w", path, err))
			continue
		}
		reg.fonts[font.Name] = font
	}

	return reg, errs
}

// Lookup finds a font by its PFF2NAME. Returns nil if not found.
func (r *FontRegistry) Lookup(name string) *PF2Font {
	return r.fonts[name]
}

// Names returns all registered font names, for diagnostics.
func (r *FontRegistry) Names() []string {
	names := make([]string, 0, len(r.fonts))
	for name := range r.fonts {
		names = append(names, name)
	}
	return names
}

// ---------------------------------------------------------------------------
// PF2 loader
// ---------------------------------------------------------------------------

// loadPF2 reads a .pf2 file from disk and returns a fully populated PF2Font
// with all glyphs loaded into Ebitengine images.
func loadPF2(path string) (*PF2Font, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	// Verify the PFF2 magic header.
	// All valid PF2 files start with the 4-byte magic "PFF2".
	if len(data) < 4 || string(data[:4]) != "PFF2" {
		return nil, fmt.Errorf("not a valid PF2 file (bad magic bytes)")
	}

	font := &PF2Font{
		FilePath: path,
		Glyphs:   make(map[rune]*Glyph),
	}

	// --- Section scan ---
	// Walk through sections until we find DATA (which must come last).
	// We collect the info we need from each section as we go.

	offset := 4 // skip past "PFF2" magic

	// We need to remember the CHIX section to decode it after we find DATA.
	var chixData []byte
	var dataSection []byte

	for offset < len(data) {
		// Need at least 8 bytes for a section header (4 name + 4 length).
		if offset+8 > len(data) {
			break
		}

		sectionName := string(data[offset : offset+4])
		sectionLen := int(binary.BigEndian.Uint32(data[offset+4 : offset+8]))
		offset += 8

		if offset+sectionLen > len(data) {
			return nil, fmt.Errorf("section %q length %d exceeds file size", sectionName, sectionLen)
		}

		sectionData := data[offset : offset+sectionLen]
		offset += sectionLen

		switch sectionName {
		case "NAME":
			// The embedded font name — this is what theme.txt references.
			// Strip any null terminator.
			font.Name = strings.TrimRight(string(sectionData), "\x00")

		case "PTSZ":
			// Point size — stored as big-endian uint16.
			if len(sectionData) >= 2 {
				font.PointSize = int(binary.BigEndian.Uint16(sectionData))
			}

		case "ASCE":
			// Ascent in pixels above baseline.
			if len(sectionData) >= 2 {
				font.Ascent = int(binary.BigEndian.Uint16(sectionData))
			}

		case "DESC":
			// Descent in pixels below baseline.
			if len(sectionData) >= 2 {
				font.Descent = int(binary.BigEndian.Uint16(sectionData))
			}

		case "CHIX":
			// Glyph index — we process this after finding DATA.
			chixData = sectionData

		case "DATA":
			// Raw glyph bitmaps — referenced by offsets in CHIX.
			dataSection = sectionData
		}
	}

	if font.Name == "" {
		// Fallback: use the filename without extension.
		base := filepath.Base(path)
		font.Name = strings.TrimSuffix(base, filepath.Ext(base))
	}

	// --- Decode glyphs from CHIX + DATA ---
	if chixData != nil && dataSection != nil {
		if err := decodeGlyphs(font, chixData, dataSection); err != nil {
			// Non-fatal: log but continue. A font with some missing glyphs
			// is better than no font at all.
			fmt.Fprintf(os.Stderr, "warning: %s: glyph decode error: %v\n", path, err)
		}
	}

	return font, nil
}

// ---------------------------------------------------------------------------
// Glyph decoding
// ---------------------------------------------------------------------------

// decodeGlyphs walks the CHIX index and reads each glyph's bitmap from DATA.
//
// CHIX entry format (9 bytes each):
//
//	4 bytes — unicode codepoint (big-endian uint32)
//	1 byte  — flags (we don't use these)
//	4 bytes — byte offset into DATA section
func decodeGlyphs(font *PF2Font, chix, data []byte) error {
	const chixEntrySize = 9

	if len(chix)%chixEntrySize != 0 {
		return fmt.Errorf("CHIX section length %d is not a multiple of %d", len(chix), chixEntrySize)
	}

	numGlyphs := len(chix) / chixEntrySize

	for i := 0; i < numGlyphs; i++ {
		entry := chix[i*chixEntrySize : (i+1)*chixEntrySize]

		codepoint := rune(binary.BigEndian.Uint32(entry[0:4]))
		// entry[4] is flags — skip
		dataOffset := int(binary.BigEndian.Uint32(entry[5:9]))

		glyph, err := decodeGlyph(data, dataOffset)
		if err != nil {
			// Skip this glyph but keep going.
			continue
		}

		font.Glyphs[codepoint] = glyph
	}

	return nil
}

// decodeGlyph reads a single glyph bitmap from the DATA section at the
// given byte offset.
//
// Glyph header format (10 bytes):
//
//	2 bytes — width  (uint16, big-endian)
//	2 bytes — height (uint16, big-endian)
//	2 bytes — x offset (int16, big-endian) — horizontal draw offset
//	2 bytes — y offset (int16, big-endian) — vertical offset from baseline
//	2 bytes — device width (uint16) — cursor advance after this glyph
//
// Followed by the bitmap: height rows, each ceil(width/8) bytes wide,
// MSB first. A set bit means a lit (foreground) pixel.
func decodeGlyph(data []byte, offset int) (*Glyph, error) {
	const headerSize = 10

	if offset+headerSize > len(data) {
		return nil, fmt.Errorf("glyph header at offset %d exceeds data length", offset)
	}

	w := int(binary.BigEndian.Uint16(data[offset+0 : offset+2]))
	h := int(binary.BigEndian.Uint16(data[offset+2 : offset+4]))
	xOff := int(int16(binary.BigEndian.Uint16(data[offset+4 : offset+6])))
	yOff := int(int16(binary.BigEndian.Uint16(data[offset+6 : offset+8])))
	devW := int(binary.BigEndian.Uint16(data[offset+8 : offset+10]))

	// Bitmap: each row is ceil(width / 8) bytes.
	rowBytes := int(math.Ceil(float64(w) / 8.0))
	bitmapSize := rowBytes * h
	bitmapStart := offset + headerSize

	if bitmapStart+bitmapSize > len(data) {
		return nil, fmt.Errorf("glyph bitmap at offset %d exceeds data length", bitmapStart)
	}

	bitmap := data[bitmapStart : bitmapStart+bitmapSize]

	// Build an RGBA image from the bitmap.
	// Lit pixels → fully opaque white (we tint at draw time with ColorM).
	// Dark pixels → fully transparent (so the background shows through).
	img := image.NewRGBA(image.Rect(0, 0, w, h))

	for row := 0; row < h; row++ {
		for col := 0; col < w; col++ {
			byteIdx := row*rowBytes + col/8
			bitIdx := 7 - (col % 8) // MSB first

			if byteIdx < len(bitmap) && (bitmap[byteIdx]>>uint(bitIdx))&1 == 1 {
				// Lit pixel — white, fully opaque.
				img.SetRGBA(col, row, color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
			} else {
				// Dark pixel — transparent.
				img.SetRGBA(col, row, color.RGBA{})
			}
		}
	}

	ebitenImg := ebiten.NewImageFromImage(img)

	return &Glyph{
		Width:       w,
		Height:      h,
		XOffset:     xOff,
		YOffset:     yOff,
		DeviceWidth: devW,
		Image:       ebitenImg,
	}, nil
}

// ---------------------------------------------------------------------------
// Text measurement and drawing
// ---------------------------------------------------------------------------

// MeasureText returns the pixel width and height of a string rendered in
// the given font. Used for alignment (center, right) and layout.
func MeasureText(font *PF2Font, text string) (width, height int) {
	if font == nil {
		return 0, 0
	}

	for _, r := range text {
		g, ok := font.Glyphs[r]
		if !ok {
			// Use space width as fallback for missing glyphs.
			if space, ok := font.Glyphs[' ']; ok {
				width += space.DeviceWidth
			} else {
				width += font.PointSize / 2
			}
			continue
		}
		width += g.DeviceWidth
	}

	height = font.Ascent + font.Descent
	return width, height
}

// DrawText draws a string onto dst at pixel position (x, y), using the
// given font and color. y is the baseline position.
//
// Color is applied by multiplying the white glyph pixels with the target
// color using Ebitengine's ColorScale.
func DrawText(dst *ebiten.Image, font *PF2Font, text string, x, y int, clr color.RGBA) {
	if font == nil {
		return
	}

	cursor := x

	for _, r := range text {
		g, ok := font.Glyphs[r]
		if !ok {
			// Advance by a default amount for missing glyphs.
			if space, ok := font.Glyphs[' ']; ok {
				cursor += space.DeviceWidth
			} else {
				cursor += font.PointSize / 2
			}
			continue
		}

		if g.Image != nil {
			op := &ebiten.DrawImageOptions{}

			// Position the glyph: apply x/y offsets from the metrics,
			// then translate to the cursor + baseline position.
			// XOffset shifts horizontally (e.g. for italic slant).
			// YOffset shifts vertically relative to the baseline.
			drawX := float64(cursor + g.XOffset)
			drawY := float64(y - g.Height - g.YOffset + font.Descent)

			op.GeoM.Translate(drawX, drawY)

			// Tint the white glyph pixels to the desired color.
			// We scale each channel by the color's normalized value.
			op.ColorScale.Scale(
				float32(clr.R)/255.0,
				float32(clr.G)/255.0,
				float32(clr.B)/255.0,
				float32(clr.A)/255.0,
			)

			dst.DrawImage(g.Image, op)
		}

		cursor += g.DeviceWidth
	}
}
