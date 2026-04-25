package renderer

import (
	"image/color"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/Uami-11/see-grub/parser"
)

// MenuEntries are the dummy boot options shown in the preview.
// These match common real-world setups so the theme looks authentic.
var MenuEntries = []string{
	"Arch Linux",
	"CachyOS",
	"Windows 11",
}

// BootMenu holds the runtime state for a boot_menu component.
// Unlike labels, the boot menu is stateful — it tracks which item
// is currently selected so it can be updated on keypress.
type BootMenu struct {
	// Parsed component data from theme.txt
	Component parser.Component

	// Resolved assets
	ItemFont      *PF2Font
	ItemStyle     *PixmapStyle // normal item border
	SelectedStyle *PixmapStyle // selected item border

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

// NewBootMenu constructs a BootMenu from a parsed component,
// loading all assets and resolving all values upfront.
// This is called once at startup, not on every frame.
func NewBootMenu(
	c parser.Component,
	fonts *FontRegistry,
	themeDir string,
) *BootMenu {
	bm := &BootMenu{
		Component: c,
		Selected:  0,
	}

	// --- Font ---
	bm.ItemFont = fonts.Lookup(c.ItemFont)

	// --- Colors ---
	bm.ItemColor = FallbackColor(c.ItemColor, ColorWhite)
	bm.SelectedItemColor = FallbackColor(c.SelectedItemColor, ColorWhite)

	// --- Pixmap styles ---
	// These are optional — many themes only define selected_item_pixmap_style
	// and leave item_pixmap_style empty (unselected items have no border).
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
	// These are stored as strings in the component. We parse them to ints
	// here so every Draw call doesn't re-parse.
	bm.ItemHeight = parseIntOrDefault(c.ItemHeight, 40)
	bm.ItemPadding = parseIntOrDefault(c.ItemPadding, 10)
	bm.ItemSpacing = parseIntOrDefault(c.ItemSpacing, 0)

	return bm
}

// HandleInput checks for up/down arrow keypresses and updates Selected.
// Returns true if the selection changed (so the renderer knows to redraw).
// Called once per game Update() tick.
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

// Draw renders all menu items onto dst.
// Called every frame from the main renderer's Draw().
func (bm *BootMenu) Draw(dst *ebiten.Image, screen Dimensions) {
	c := bm.Component

	// Resolve the menu box position and size.
	menuRect := ResolveRect(c.Left, c.Top, c.Width, c.Height, screen)

	// --- Draw each entry ---
	// Items are stacked vertically. Each item occupies ItemHeight pixels,
	// with ItemSpacing additional pixels of gap between items.
	//
	// Note: ItemSpacing in GRUB is the distance from the TOP of one item
	// to the TOP of the next — i.e. it includes the item height itself.
	// If ItemSpacing < ItemHeight the items would overlap, but themes
	// are expected to set it sensibly.
	//
	// Your theme sets item_spacing = 180 and item_height = 100,
	// so there's 80px of gap between the bottom of one item and
	// the top of the next. We replicate that here.

	for i, entry := range MenuEntries {
		// Top Y of this item relative to the menu box top.
		var itemY int
		if bm.ItemSpacing > 0 {
			// GRUB: spacing is top-of-item to top-of-next-item.
			itemY = menuRect.Y + i*bm.ItemSpacing
		} else {
			// No spacing set: stack items with just item_height.
			itemY = menuRect.Y + i*(bm.ItemHeight+bm.ItemPadding)
		}

		itemRect := Rect{
			X: menuRect.X,
			Y: itemY,
			W: menuRect.W,
			H: bm.ItemHeight,
		}

		// Stop drawing if we've exceeded the menu box's height.
		if itemRect.Y+itemRect.H > menuRect.Y+menuRect.H {
			break
		}

		bm.drawItem(dst, itemRect, entry, i == bm.Selected)
	}
}

// drawItem draws a single menu item — its border pixmap and label text.
func (bm *BootMenu) drawItem(
	dst *ebiten.Image,
	itemRect Rect,
	text string,
	selected bool,
) {
	// --- Draw border pixmap ---
	if selected && bm.SelectedStyle != nil {
		bm.SelectedStyle.Draw(dst, itemRect)
	} else if !selected && bm.ItemStyle != nil {
		bm.ItemStyle.Draw(dst, itemRect)
	}

	// --- Draw text ---
	if bm.ItemFont == nil {
		return
	}

	// Choose the right color.
	clr := bm.ItemColor
	if selected {
		clr = bm.SelectedItemColor
	}

	// Measure text so we can center it vertically within the item height.
	textW, textH := MeasureText(bm.ItemFont, text)

	// Horizontal position: item_padding from the left edge.
	textX := itemRect.X + bm.ItemPadding

	// Vertical centering: place baseline so text is centered in item height.
	// Center Y of item → subtract half the text height to get the top of
	// the text → add ascent to get the baseline.
	textY := itemRect.Y + (itemRect.H-textH)/2 + bm.ItemFont.Ascent

	// Ignore textW if not doing center/right alignment —
	// boot menu items are always left-aligned in GRUB.
	_ = textW

	// Use the color.RGBA directly — FallbackColor returns color.RGBA.
	DrawText(dst, bm.ItemFont, text, textX, textY, clr)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// parseIntOrDefault parses a string as an integer.
// Returns defaultVal if the string is empty or not a valid integer.
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
