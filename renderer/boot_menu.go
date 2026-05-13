package renderer

import (
	"encoding/json"
	"image/color"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/Uami-11/see-grub/parser"
)

var defaultMenuEntries = []string{
	"Arch Linux",
	"Ubuntu",
	"Windows 11",
}

var MenuEntries []string

func init() {
	MenuEntries = append([]string{}, defaultMenuEntries...)
	LoadEntries()
}

func entriesFilePath() string {
	cfgDir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	dir := filepath.Join(cfgDir, "see-grub")
	return filepath.Join(dir, "entries.json")
}

func LoadEntries() {
	path := entriesFilePath()
	if path == "" {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var loaded []string
	if err := json.Unmarshal(data, &loaded); err != nil {
		return
	}
	if len(loaded) > 0 {
		MenuEntries = loaded
	}
}

func SaveEntries() error {
	path := entriesFilePath()
	if path == "" {
		return nil
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.Marshal(MenuEntries)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func ResetMenuEntries() {
	MenuEntries = append([]string{}, defaultMenuEntries...)
	SaveEntries()
}

type BootMenu struct {
	Component parser.Component

	ItemFont      *PF2Font
	ItemStyle     *PixmapStyle
	SelectedStyle *PixmapStyle
	MenuStyle     *PixmapStyle

	// Resolved colors
	ItemColor         color.RGBA
	SelectedItemColor color.RGBA

	// Resolved dimensions
	ItemHeight  int
	ItemPadding int
	ItemSpacing int

	// Navigation state
	Selected     int // index into MenuEntries, 0-based
	ScrollOffset int // index of first visible entry

	// Icons
	Icons     map[string]*ebiten.Image
	IconW     int
	IconH     int
	IconSpace int
}

func NewBootMenu(
	c parser.Component,
	fonts *FontRegistry,
	themeDir string,
) *BootMenu {
	bm := &BootMenu{
		Component:    c,
		Selected:     0,
		ScrollOffset: 0,
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
	if c.MenuPixmapStyle != "" {
		style, err := LoadPixmapStyle(themeDir, c.MenuPixmapStyle)
		if err == nil {
			bm.MenuStyle = style
		}
	}

	// --- Icons ---
	bm.Icons = loadIcons(themeDir)
	bm.IconW = ResolveDim(c.IconWidth, bm.ItemHeight)
	if bm.IconW <= 0 {
		bm.IconW = parseIntOrDefault(c.IconWidth, 32)
	}
	bm.IconH = ResolveDim(c.IconHeight, bm.ItemHeight)
	if bm.IconH <= 0 {
		bm.IconH = parseIntOrDefault(c.IconHeight, 32)
	}
	bm.IconSpace = parseIntOrDefault(c.ItemIconSpace, 8)

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

	if bm.MenuStyle != nil {
		bm.MenuStyle.Draw(dst, menuRect)
	}

	hasBorder := bm.ItemStyle != nil || bm.SelectedStyle != nil

	cornerH := 0
	if hasBorder {
		if bm.ItemStyle != nil && bm.ItemStyle.CornerH > 0 {
			cornerH = bm.ItemStyle.CornerH
		} else if bm.SelectedStyle != nil && bm.SelectedStyle.CornerH > 0 {
			cornerH = bm.SelectedStyle.CornerH
		}
	}

	itemStep := bm.ItemHeight + bm.ItemSpacing
	availableH := menuRect.H - cornerH
	visibleCount := availableH/itemStep + 1
	if visibleCount < 1 {
		visibleCount = 1
	}

	maxOffset := len(MenuEntries) - visibleCount
	if maxOffset < 0 {
		maxOffset = 0
	}
	if bm.ScrollOffset > bm.Selected {
		bm.ScrollOffset = bm.Selected
	}
	if bm.ScrollOffset < bm.Selected-visibleCount+1 {
		bm.ScrollOffset = bm.Selected - visibleCount + 1
	}
	if bm.ScrollOffset > maxOffset {
		bm.ScrollOffset = maxOffset
	}

	for i := bm.ScrollOffset; i < len(MenuEntries); i++ {
		relIndex := i - bm.ScrollOffset
		itemY := menuRect.Y + cornerH + relIndex*(bm.ItemHeight+bm.ItemSpacing)
		itemRect := Rect{X: menuRect.X, Y: itemY, W: menuRect.W, H: bm.ItemHeight}

		var borderRect Rect
		if hasBorder {
			borderRect = Rect{
				X: menuRect.X,
				Y: itemY - cornerH,
				W: menuRect.W,
				H: bm.ItemHeight + cornerH*2,
			}
		} else {
			borderRect = itemRect
		}

		if itemRect.Y > menuRect.Y+menuRect.H {
			break
		}

		bm.drawItem(dst, itemRect, borderRect, MenuEntries[i], i == bm.Selected)
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

	iconX := itemRect.X + bm.ItemPadding
	iconY := itemRect.Y + (itemRect.H-bm.IconH)/2

	textOffset := 0
	iconKey := firstWordLowercased(text)
	if icon, ok := bm.Icons[iconKey]; ok && bm.IconW > 0 && bm.IconH > 0 {
		op := &ebiten.DrawImageOptions{}
		sx := float64(bm.IconW) / float64(icon.Bounds().Dx())
		sy := float64(bm.IconH) / float64(icon.Bounds().Dy())
		op.GeoM.Scale(sx, sy)
		op.GeoM.Translate(float64(iconX), float64(iconY))
		dst.DrawImage(icon, op)
		textOffset = bm.IconW + bm.IconSpace
	}

	textW, textH := MeasureText(bm.ItemFont, text)
	centerRowTop := borderRect.Y + borderRect.H/3
	centerRowH := borderRect.H / 3

	var textX int
	if textOffset > 0 {
		textX = borderRect.X + textOffset + bm.ItemPadding
	} else {
		textX = borderRect.X + (borderRect.W-textW)/2
	}
	textY := centerRowTop + (centerRowH-textH)/2 + bm.ItemFont.Ascent

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

func loadIcons(themeDir string) map[string]*ebiten.Image {
	iconsDir := filepath.Join(themeDir, "icons")
	entries, err := os.ReadDir(iconsDir)
	if err != nil {
		return nil
	}
	icons := make(map[string]*ebiten.Image)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".png") {
			continue
		}
		stem := strings.TrimSuffix(name, ".png")
		img, _, _, err := loadPNG(filepath.Join(iconsDir, name))
		if err != nil {
			continue
		}
		icons[stem] = img
	}
	return icons
}

func firstWordLowercased(s string) string {
	parts := strings.SplitN(s, " ", 2)
	return strings.ToLower(parts[0])
}
