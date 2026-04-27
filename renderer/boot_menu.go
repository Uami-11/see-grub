package renderer

import (
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

	for i, entry := range MenuEntries {
		var itemY int
		if bm.ItemSpacing > 0 {
			itemY = menuRect.Y + i*bm.ItemSpacing
		} else {
			itemY = menuRect.Y + i*(bm.ItemHeight+bm.ItemPadding)
		}

		itemRect := Rect{
			X: menuRect.X,
			Y: itemY,
			W: menuRect.W,
			H: bm.ItemHeight,
		}

		if itemRect.Y+itemRect.H > menuRect.Y+menuRect.H {
			break
		}

		bm.drawItem(dst, itemRect, entry, i == bm.Selected)
	}
}

func (bm *BootMenu) drawItem(dst *ebiten.Image, itemRect Rect, text string, selected bool) {
	// Draw border centered on item, at natural aspect ratio scaled to item width
	style := bm.ItemStyle
	if selected {
		style = bm.SelectedStyle
	}

	if style != nil {
		// Natural border dimensions from the 565×233 composite
		const naturalW, naturalH = 565, 233

		// Scale to item width, preserve aspect ratio
		borderW := itemRect.W
		borderH := int(float64(borderW) / float64(naturalW) * float64(naturalH))

		// Center vertically on the item
		borderY := itemRect.Y + (itemRect.H-borderH)/2

		borderRect := Rect{
			X: itemRect.X,
			Y: borderY,
			W: borderW,
			H: borderH,
		}
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
	textX := itemRect.X + (itemRect.W-textW)/2
	textY := itemRect.Y + (itemRect.H-textH)/2
	_ = textW

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
