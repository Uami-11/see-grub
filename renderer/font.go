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

type PF2Font struct {
	Name      string
	FilePath  string
	PointSize int
	Glyphs    map[rune]*Glyph
	Ascent    int
	Descent   int
}

type Glyph struct {
	Width       int
	Height      int
	XOffset     int
	YOffset     int
	DeviceWidth int
	Image       *ebiten.Image
}

type FontRegistry struct {
	fonts map[string]*PF2Font // key is the PFF2NAME string
}

func NewFontRegistry(themeDir string) (*FontRegistry, []error) {
	reg := &FontRegistry{
		fonts: make(map[string]*PF2Font),
	}

	var matches []string
	err := filepath.Walk(themeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(path, ".pf2") {
			matches = append(matches, path)
		}
		return nil
	})
	if err != nil {
		return reg, []error{fmt.Errorf("scanning for .pf2 files: %w", err)}
	}

	if len(matches) == 0 {
		return reg, []error{fmt.Errorf("no .pf2 files found in %q", themeDir)}
	}

	var errs []error
	for _, path := range matches {
		font, err := loadPF2(path)
		if err != nil {
			errs = append(errs, fmt.Errorf("loading %s: %w", path, err))
			continue
		}
		reg.fonts[font.Name] = font

		fmt.Printf("  font loaded: %q from %s\n", font.Name, filepath.Base(path))
	}

	return reg, errs
}

func (r *FontRegistry) Lookup(name string) *PF2Font {
	return r.fonts[name]
}

func (r *FontRegistry) Names() []string {
	names := make([]string, 0, len(r.fonts))
	for name := range r.fonts {
		names = append(names, name)
	}
	return names
}

func loadPF2(path string) (*PF2Font, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	if len(data) < 4 || string(data[:4]) != "PFF2" {
		return nil, fmt.Errorf("not a valid PF2 file (bad magic bytes)")
	}

	font := &PF2Font{
		FilePath: path,
		Glyphs:   make(map[rune]*Glyph),
	}

	offset := 4 // skip past "PFF2" magic

	var chixData []byte
	var dataSection []byte

	for offset < len(data) {
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
			font.Name = strings.TrimRight(string(sectionData), "\x00")

		case "PTSZ":
			if len(sectionData) >= 2 {
				font.PointSize = int(binary.BigEndian.Uint16(sectionData))
			}

		case "ASCE":
			if len(sectionData) >= 2 {
				font.Ascent = int(binary.BigEndian.Uint16(sectionData))
			}

		case "DESC":
			if len(sectionData) >= 2 {
				font.Descent = int(binary.BigEndian.Uint16(sectionData))
			}

		case "CHIX":
			chixData = sectionData

		case "DATA":
			dataSection = sectionData
		}
	}

	if font.Name == "" {
		base := filepath.Base(path)
		font.Name = strings.TrimSuffix(base, filepath.Ext(base))
	}

	if chixData != nil && dataSection != nil {
		if err := decodeGlyphs(font, chixData, dataSection); err != nil {
			fmt.Fprintf(os.Stderr, "warning: %s: glyph decode error: %v\n", path, err)
		}
	}

	return font, nil
}

func decodeGlyphs(font *PF2Font, chix, data []byte) error {
	const chixEntrySize = 9

	if len(chix)%chixEntrySize != 0 {
		return fmt.Errorf("CHIX section length %d is not a multiple of %d", len(chix), chixEntrySize)
	}

	numGlyphs := len(chix) / chixEntrySize

	for i := 0; i < numGlyphs; i++ {
		entry := chix[i*chixEntrySize : (i+1)*chixEntrySize]

		codepoint := rune(binary.BigEndian.Uint32(entry[0:4]))
		dataOffset := int(binary.BigEndian.Uint32(entry[5:9]))

		glyph, err := decodeGlyph(data, dataOffset)
		if err != nil {
			continue
		}

		font.Glyphs[codepoint] = glyph
	}

	return nil
}

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

	rowBytes := int(math.Ceil(float64(w) / 8.0))
	bitmapSize := rowBytes * h
	bitmapStart := offset + headerSize

	if bitmapStart+bitmapSize > len(data) {
		return nil, fmt.Errorf("glyph bitmap at offset %d exceeds data length", bitmapStart)
	}

	bitmap := data[bitmapStart : bitmapStart+bitmapSize]

	img := image.NewRGBA(image.Rect(0, 0, w, h))

	for row := 0; row < h; row++ {
		for col := 0; col < w; col++ {
			byteIdx := row*rowBytes + col/8
			bitIdx := 7 - (col % 8) // MSB first

			if byteIdx < len(bitmap) && (bitmap[byteIdx]>>uint(bitIdx))&1 == 1 {
				img.SetRGBA(col, row, color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
			} else {
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

func MeasureText(font *PF2Font, text string) (width, height int) {
	if font == nil {
		return 0, 0
	}

	for _, r := range text {
		g, ok := font.Glyphs[r]
		if !ok {
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

func DrawText(dst *ebiten.Image, font *PF2Font, text string, x, y int, clr color.RGBA) {
	if font == nil {
		return
	}

	cursor := x

	for _, r := range text {
		g, ok := font.Glyphs[r]
		if !ok {
			if space, ok := font.Glyphs[' ']; ok {
				cursor += space.DeviceWidth
			} else {
				cursor += font.PointSize / 2
			}
			continue
		}

		if g.Image != nil {
			op := &ebiten.DrawImageOptions{}

			drawX := float64(cursor + g.XOffset)
			drawY := float64(y - g.Height - g.YOffset + font.Descent)

			op.GeoM.Translate(drawX, drawY)

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

// DebugFonts scans themeDir for .pf2 files and returns diagnostic info
// about each one without opening a window. Called via --fonts flag.
func DebugFonts(themeDir string) ([]string, []error) {
	var lines []string
	var errs []error

	var pf2files []string
	filepath.Walk(themeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && strings.HasSuffix(path, ".pf2") {
			pf2files = append(pf2files, path)
		}
		return nil
	})

	lines = append(lines, fmt.Sprintf("Found %d .pf2 files:", len(pf2files)))

	for _, path := range pf2files {
		data, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, err)
			continue
		}

		base := filepath.Base(path)

		// Check magic
		if len(data) < 4 {
			lines = append(lines, fmt.Sprintf("  %-40s INVALID (too short)", base))
			continue
		}
		magic := string(data[:4])
		if magic != "PFF2" {
			lines = append(lines, fmt.Sprintf("  %-40s INVALID magic: %q", base, magic))
			continue
		}

		// Scan for NAME section
		name := ""
		offset := 4
		for offset+8 <= len(data) {
			sectionName := string(data[offset : offset+4])
			sectionLen := int(binary.BigEndian.Uint32(data[offset+4 : offset+8]))
			offset += 8
			if offset+sectionLen > len(data) {
				break
			}
			if sectionName == "NAME" {
				name = strings.TrimRight(string(data[offset:offset+sectionLen]), "\x00")
				break
			}
			offset += sectionLen
		}

		if name == "" {
			lines = append(lines, fmt.Sprintf("  %-40s NO NAME section found", base))
		} else {
			lines = append(lines, fmt.Sprintf("  %-40s -> %q", base, name))
		}
	}

	return lines, errs
}
