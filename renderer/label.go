package renderer

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/Uami-11/see-grub/parser"
)

const CountdownSeconds = 5

func DrawLabel(
	dst *ebiten.Image,
	c parser.Component,
	fonts *FontRegistry,
	screen Dimensions,
) {
	text := resolveText(c.Text, c.ID)
	if text == "" {
		return
	}

	font := fonts.Lookup(c.Font)
	if font == nil {
		return
	}

	clr := FallbackColor(c.Color, ColorWhite)

	x := ResolveX(c.Left, screen)
	y := ResolveY(c.Top, screen)
	w := ResolveDim(c.Width, screen.Width)

	textW, _ := MeasureText(font, text)

	switch strings.ToLower(c.Align) {
	case "center":
		if w > 0 {
			x = x + (w-textW)/2
		} else {
			x = x - textW/2
		}
	case "right":
		if w > 0 {
			x = x + w - textW
		} else {
			x = x - textW
		}
	default:
		// "left" or empty — x is already the left edge, no adjustment needed.
	}

	DrawText(dst, font, text, x, y+font.Ascent, clr)
}

func resolveText(text, id string) string {
	if id == "__timeout__" {
		return fmt.Sprintf(text, CountdownSeconds)
	}
	return text
}
