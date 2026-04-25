package renderer

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/Uami-11/see-grub/parser"
)

// CountdownSeconds is the dummy timeout value shown in the __timeout__ label.
// Real GRUB counts down from whatever `set timeout=N` is in grub.cfg.
// We show a fixed value since we're just previewing the theme.
const CountdownSeconds = 5

// DrawLabel draws a single label component onto dst.
//
// It needs:
//   - the parsed component (position, text, font name, color, align, id)
//   - the font registry to look up the font by PFF2NAME
//   - the screen dimensions to resolve percentage/relative positions
func DrawLabel(
	dst *ebiten.Image,
	c parser.Component,
	fonts *FontRegistry,
	screen Dimensions,
) {
	// --- Resolve text ---
	// The __timeout__ label uses a special format string with %d.
	// We substitute our dummy countdown value so the label renders
	// exactly as it would in real GRUB.
	text := resolveText(c.Text, c.ID)
	if text == "" {
		return
	}

	// --- Resolve font ---
	// Font name in theme.txt matches PFF2NAME inside the .pf2 binary,
	// not the filename. FontRegistry.Lookup handles that mapping.
	font := fonts.Lookup(c.Font)
	// If font not found we skip drawing — the parser already reported
	// ErrFontNotFound for this. Silently skipping here matches GRUB behavior.
	if font == nil {
		return
	}

	// --- Resolve color ---
	clr := FallbackColor(c.Color, ColorWhite)

	// --- Resolve position ---
	// We resolve X and Y independently because alignment shifts X but not Y.
	x := ResolveX(c.Left, screen)
	y := ResolveY(c.Top, screen)

	// --- Apply alignment ---
	// GRUB anchors the label differently depending on align:
	//   left   — x is the left edge of the text (default)
	//   center — x is the horizontal center of the text
	//   right  — x is the right edge of the text
	//
	// We measure the text width first so we can shift x accordingly.
	textW, _ := MeasureText(font, text)

	switch strings.ToLower(c.Align) {
	case "center":
		x = x - textW/2
	case "right":
		x = x - textW
	default:
		// "left" or empty — x is already the left edge, no adjustment needed.
	}

	// --- Draw ---
	// y is the baseline position. DrawText handles the ascent/descent offset
	// internally so callers just pass the top-left conceptual position.
	DrawText(dst, font, text, x, y+font.Ascent, clr)
}

// resolveText returns the display string for a label.
//
// Special case: labels with id = "__timeout__" contain a Go-style %d
// format verb that GRUB substitutes with the remaining seconds.
// We substitute our fixed CountdownSeconds value so the preview looks real.
//
// All other labels return their text unchanged.
func resolveText(text, id string) string {
	if id == "__timeout__" {
		// Replace GRUB's %d with our dummy countdown value.
		// fmt.Sprintf handles this naturally.
		return fmt.Sprintf(strings.ReplaceAll(text, "%d", "%d"), CountdownSeconds)
	}
	return text
}
