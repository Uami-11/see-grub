package renderer

import (
	"fmt"
	"image/color"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/Uami-11/see-grub/parser"
)

// ---------------------------------------------------------------------------
// Game — the Ebitengine entry point
// ---------------------------------------------------------------------------

// Game implements ebiten.Game and owns all renderer state.
// It is constructed once from a parsed Theme, then handed to ebiten.RunGame.
type Game struct {
	// Source data
	theme    *parser.Theme
	themeDir string

	// Screen size — derived from the background image dimensions.
	screenW int
	screenH int

	// Loaded assets
	background *Background
	fonts      *FontRegistry

	// Built components — processed from theme.Components at startup.
	// We keep labels as plain parser.Component values since DrawLabel
	// is stateless. BootMenus are stateful so they get their own type.
	labels    []parser.Component
	bootMenus []*BootMenu

	// Error overlay — any non-fatal errors accumulated during init
	// are drawn as a semi-transparent panel so you can see them while
	// still seeing the theme render underneath.
	initErrors []string
}

// ---------------------------------------------------------------------------
// Constructor
// ---------------------------------------------------------------------------

// NewGame builds a Game from a parsed theme.
// All asset loading happens here — by the time RunGame starts, everything
// is in memory and Draw() never touches the filesystem.
func NewGame(theme *parser.Theme, themeDir string) (*Game, error) {
	g := &Game{
		theme:    theme,
		themeDir: themeDir,
	}

	// --- Background ---
	// We load this first because its dimensions become the window size.
	bg, err := LoadBackground(themeDir, theme.DesktopImage, theme.DesktopColor)
	if err != nil {
		// Non-fatal: report it but continue with a black background.
		g.initErrors = append(g.initErrors, fmt.Sprintf("background: %v", err))
	}
	g.background = bg

	// --- Screen size ---
	// Use the background image's natural size as the window dimensions.
	// This is how real GRUB works — the theme is designed for a specific
	// resolution and the terminal fills that resolution.
	//
	// Fallback: if no background image was loaded, use 1920×1080 as a
	// sensible default for modern displays.
	if bg != nil && bg.Width > 0 && bg.Height > 0 {
		g.screenW = bg.Width
		g.screenH = bg.Height
	} else {
		g.screenW = 1920
		g.screenH = 1080
	}

	// --- Fonts ---
	fonts, fontErrs := NewFontRegistry(themeDir)
	for _, err := range fontErrs {
		g.initErrors = append(g.initErrors, fmt.Sprintf("font: %v", err))
	}
	g.fonts = fonts

	// --- Components ---
	g.buildComponents(theme.Components)

	return g, nil
}

// buildComponents walks the parsed component list and sets up each one.
// Labels are stored as-is. BootMenus are initialized with NewBootMenu.
func (g *Game) buildComponents(components []parser.Component) {
	for _, c := range components {
		switch c.Type {
		case parser.ComponentLabel:
			g.labels = append(g.labels, c)

		case parser.ComponentBootMenu:
			bm := NewBootMenu(c, g.fonts, g.themeDir)
			g.bootMenus = append(g.bootMenus, bm)

		case parser.ComponentProgressBar:
			// Progress bar renders as a simple colored rectangle for now.
			// Full implementation (fg/bg/border) comes in a later step.
			g.labels = append(g.labels, c)

		case parser.ComponentImage:
			// Static image components — treated like labels for ordering.
			// Full image rendering comes in a later step.

		case parser.ComponentVBox, parser.ComponentHBox:
			// Container components — recurse into children.
			g.buildComponents(c.Children)
		}
	}
}

// ---------------------------------------------------------------------------
// Ebitengine interface — Update
// ---------------------------------------------------------------------------

// Update runs at 60hz and handles input.
// Ebitengine separates logic (Update) from rendering (Draw) so that
// game state is always updated at a consistent rate regardless of
// frame rate.
func (g *Game) Update() error {
	// ESC or Q quits the preview — matches GRUB's own quit behavior
	// (GRUB doesn't have quit, but for a preview tool it's essential).
	if ebiten.IsKeyPressed(ebiten.KeyEscape) || ebiten.IsKeyPressed(ebiten.KeyQ) {
		os.Exit(0)
	}

	// Delegate input to each boot menu.
	for _, bm := range g.bootMenus {
		bm.HandleInput()
	}

	return nil
}

// ---------------------------------------------------------------------------
// Ebitengine interface — Draw
// ---------------------------------------------------------------------------

// Draw renders one frame. Called by Ebitengine after every Update.
// The draw order mirrors GRUB's own compositing order:
//  1. Background image (bottom layer)
//  2. Components in theme.txt order (back to front)
//  3. Error overlay (top layer, only if there were init errors)
func (g *Game) Draw(screen *ebiten.Image) {
	screenDims := Dimensions{Width: g.screenW, Height: g.screenH}

	// --- 1. Background ---
	g.background.Draw(screen)

	// --- 2. Components ---
	// We need to draw in theme.txt order (which preserves painter's algorithm
	// layering), but we've split components into typed slices for convenience.
	// So we walk the original component list and dispatch by type.
	for _, c := range g.theme.Components {
		switch c.Type {
		case parser.ComponentLabel:
			DrawLabel(screen, c, g.fonts, screenDims)

		case parser.ComponentBootMenu:
			// Find the matching BootMenu by component line number.
			// (Line number is unique per component in a valid theme.txt.)
			for _, bm := range g.bootMenus {
				if bm.Component.Line == c.Line {
					bm.Draw(screen, screenDims)
					break
				}
			}
		}
	}

	// --- 3. Error overlay ---
	if len(g.initErrors) > 0 {
		g.drawErrorOverlay(screen)
	}
}

// ---------------------------------------------------------------------------
// Ebitengine interface — Layout
// ---------------------------------------------------------------------------

// Layout tells Ebitengine the logical screen size.
// We return the background image's natural dimensions so that every
// pixel in our render corresponds to one pixel on the logical canvas.
//
// Ebitengine handles scaling to the actual OS window size automatically —
// so if the user's monitor is smaller than the theme resolution,
// Ebitengine scales it down while preserving the aspect ratio.
func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.screenW, g.screenH
}

// ---------------------------------------------------------------------------
// Error overlay
// ---------------------------------------------------------------------------

// drawErrorOverlay paints a semi-transparent dark panel in the bottom-left
// corner listing any errors that occurred during initialization.
// This lets you see the theme AND the error simultaneously.
func (g *Game) drawErrorOverlay(screen *ebiten.Image) {
	// Panel dimensions — grow with the number of errors.
	lineHeight := 18
	padding := 10
	panelH := padding*2 + len(g.initErrors)*lineHeight
	panelW := g.screenW / 2

	panelX := padding
	panelY := g.screenH - panelH - padding

	// Draw a semi-transparent dark background.
	// We do this by drawing a filled rectangle using a 1×1 image scaled up.
	overlay := ebiten.NewImage(panelW, panelH)
	overlay.Fill(color.RGBA{R: 0x10, G: 0x00, B: 0x00, A: 0xcc})

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(panelX), float64(panelY))
	screen.DrawImage(overlay, op)

	// Draw each error line as white text using the terminal font if available,
	// otherwise we use a built-in fallback.
	//
	// For now we print using the terminal font from the theme (if loaded).
	// In a future step we'll bundle a fallback monospace font for cases
	// where the theme's terminal font isn't available.
	termFont := g.fonts.Lookup(g.theme.TerminalFont)

	for i, msg := range g.initErrors {
		x := panelX + padding
		y := panelY + padding + i*lineHeight + lineHeight

		if termFont != nil {
			DrawText(screen, termFont, "✗ "+msg, x, y, ColorWhite)
		}
		// If no font is available we skip text for now —
		// font rendering without a loaded font is handled in a future step.
	}
}

// ---------------------------------------------------------------------------
// Run — public entry point
// ---------------------------------------------------------------------------

// Run initialises the Ebitengine window and starts the game loop.
// This is called from main.go once the theme is parsed and the Game is built.
func Run(theme *parser.Theme, themeDir string) error {
	game, err := NewGame(theme, themeDir)
	if err != nil {
		return fmt.Errorf("initialising renderer: %w", err)
	}

	ebiten.SetWindowSize(game.screenW, game.screenH)
	ebiten.SetWindowTitle("see-grub — " + themeDir)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	// Fullscreen hint: start the window at a sensible size but allow
	// the user to resize. The logical canvas stays at theme resolution
	// (Layout() handles that) so the rendering is always pixel-accurate.
	ebiten.SetWindowSizeLimits(320, 240, -1, -1)

	return ebiten.RunGame(game)
}
