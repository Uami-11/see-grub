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

	textW, _ := MeasureText(font, text)

	switch strings.ToLower(c.Align) {
	case "center":
		x = x - textW/2
	case "right":
		x = x - textW
	default:
		// "left" or empty — x is already the left edge, no adjustment needed.
	}

	DrawText(dst, font, text, x, y, clr)
}

func resolveText(text, id string) string {
	if id == "__timeout__" {
		return fmt.Sprintf(strings.ReplaceAll(text, "%d", "%d"), CountdownSeconds)
	}
	return text
}
