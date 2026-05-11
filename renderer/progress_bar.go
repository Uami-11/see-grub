package renderer

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/Uami-11/see-grub/parser"
)

var fillPixel *ebiten.Image

func init() {
	fillPixel = ebiten.NewImage(1, 1)
	fillPixel.Fill(color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
}

const progressPct = 0.65

func DrawProgressBar(dst *ebiten.Image, c parser.Component, screen Dimensions) {
	rect := ResolveRect(c.Left, c.Top, c.Width, c.Height, screen)
	if rect.W <= 0 || rect.H <= 0 {
		return
	}

	fg := FallbackColor(c.FgColor, ColorWhite)
	bg := FallbackColor(c.BgColor, color.RGBA{0x33, 0x33, 0x33, 0xff})
	borderClr := FallbackColor(c.BorderColor, color.RGBA{})

	if c.BorderColor != "" {
		fillRect(dst, rect, borderClr)
		rect = rect.Inset(1)
		if rect.W <= 0 || rect.H <= 0 {
			return
		}
	}

	fillRect(dst, rect, bg)

	fillW := int(float64(rect.W) * progressPct)
	if fillW > 0 {
		fillRect(dst, Rect{X: rect.X, Y: rect.Y, W: fillW, H: rect.H}, fg)
	}
}

func fillRect(dst *ebiten.Image, r Rect, clr color.RGBA) {
	if r.W <= 0 || r.H <= 0 {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(r.W), float64(r.H))
	op.GeoM.Translate(float64(r.X), float64(r.Y))
	op.ColorScale.Scale(
		float32(clr.R)/255,
		float32(clr.G)/255,
		float32(clr.B)/255,
		float32(clr.A)/255,
	)
	dst.DrawImage(fillPixel, op)
}
