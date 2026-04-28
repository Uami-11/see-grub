package renderer

import (
	"fmt"
	"image/color"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/Uami-11/see-grub/parser"
)

var MenuEntries = []string{
	"Arch Linux",
	"CachyOS",
	"Windows 11",
}

type BootMenu struct {
	Component parser.Component

	ItemFont      *PF2Font
	ItemStyle     *PixmapStyle
	SelectedStyle *PixmapStyle

	// Resolved colors
	ItemColor         color.RGBA
	SelectedItemColor color.RGBA

	// Resolved dimensions
	ItemHeight  int
	ItemPadding int
	ItemSpacing int

	// Navigation state
	Selected int // index into MenuEntries, 0-based
}

func NewBootMenu(
	c parser.Component,
	fonts *FontRegistry,
	themeDir string,
) *BootMenu {
	bm := &BootMenu{
		Component: c,
		Selected:  0,
	}

	bm.ItemFont = fonts.Lookup(c.ItemFont)

	bm.ItemColor = FallbackColor(c.ItemColor, ColorWhite)
	bm.SelectedItemColor = FallbackColor(c.SelectedItemColor, ColorWhite)

	// --- Pixmap styles ---
	if c.ItemPixmapStyle != "" {
		style, err := LoadPixmapStyle(themeDir, c.ItemPixmapStyle)
		if err == nil {
			bm.ItemStyle = style
		}
	}
	if c.SelectedItemPixmapStyle != "" {
		style, err := LoadPixmapStyle(themeDir, c.SelectedItemPixmapStyle)
		if err == nil {
			bm.SelectedStyle = style
		}
	}

	// --- Dimensions ---
	bm.ItemHeight = parseIntOrDefault(c.ItemHeight, 40)
	bm.ItemPadding = parseIntOrDefault(c.ItemPadding, 10)
	bm.ItemSpacing = parseIntOrDefault(c.ItemSpacing, 0)

	return bm
}

func (bm *BootMenu) HandleInput() bool {
	changed := false

	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) {
		if bm.Selected > 0 {
			bm.Selected--
			changed = true
		}
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) {
		if bm.Selected < len(MenuEntries)-1 {
			bm.Selected++
			changed = true
		}
	}

	return changed
}

func (bm *BootMenu) Draw(dst *ebiten.Image, screen Dimensions) {
	c := bm.Component
	menuRect := ResolveRect(c.Left, c.Top, c.Width, c.Height, screen)

	const naturalW, naturalH = 565, 233
	borderW := menuRect.W
	borderH := int(float64(borderW) / float64(naturalW) * float64(naturalH))

	step := bm.ItemSpacing
	if borderH > step {
		step = borderH + 20
	}

	for i, entry := range MenuEntries {
		// item_spacing is the distance between border tops
		borderY := menuRect.Y + i*step

		borderRect := Rect{
			X: menuRect.X,
			Y: borderY,
			W: borderW,
			H: borderH,
		}

		// itemRect is centered within the border for text positioning
		itemRect := Rect{
			X: menuRect.X,
			Y: borderY + (borderH-bm.ItemHeight)/2,
			W: menuRect.W,
			H: bm.ItemHeight,
		}

		if borderY > menuRect.Y+menuRect.H {
			break
		}

		bm.drawItem(dst, itemRect, borderRect, entry, i == bm.Selected)
	}
}

func (bm *BootMenu) drawItem(dst *ebiten.Image, itemRect Rect, borderRect Rect, text string, selected bool) {
	style := bm.ItemStyle
	if selected {
		style = bm.SelectedStyle
	}

	if style != nil {
		style.Draw(dst, borderRect)
	}

	if bm.ItemFont == nil {
		return
	}

	clr := bm.ItemColor
	if selected {
		clr = bm.SelectedItemColor
	}

	textW, textH := MeasureText(bm.ItemFont, text)
	// Text centered in the border's middle section (center row of 9-slice)
	// The center row is 1/3 of the border height
	centerRowTop := borderRect.Y + borderRect.H/3
	centerRowH := borderRect.H / 3
	textX := borderRect.X + (borderRect.W-textW)/2
	textY := centerRowTop + (centerRowH-textH)/2 + bm.ItemFont.Ascent

	_ = itemRect
	// Temporarily in drawItem, before DrawText:
	fmt.Printf("drawItem: borderRect=%v centerRowTop=%d centerRowH=%d textH=%d textY=%d\n",
		borderRect, centerRowTop, centerRowH, textH, textY)
	DrawText(dst, bm.ItemFont, text, textX, textY, clr)
}

func parseIntOrDefault(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return n
}
