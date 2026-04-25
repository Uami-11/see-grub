package renderer

import (
	"fmt"
	"image/color"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/Uami-11/see-grub/parser"
)

type Game struct {
	theme    *parser.Theme
	themeDir string

	screenW int
	screenH int

	background *Background
	fonts      *FontRegistry

	labels    []parser.Component
	bootMenus []*BootMenu

	initErrors []string
}

func NewGame(theme *parser.Theme, themeDir string) (*Game, error) {
	g := &Game{
		theme:    theme,
		themeDir: themeDir,
	}

	bg, err := LoadBackground(themeDir, theme.DesktopImage, theme.DesktopColor)
	if err != nil {
		g.initErrors = append(g.initErrors, fmt.Sprintf("background: %v", err))
	}
	g.background = bg

	if bg != nil && bg.Width > 0 && bg.Height > 0 {
		g.screenW = bg.Width
		g.screenH = bg.Height
	} else {
		g.screenW = 1920
		g.screenH = 1080
	}

	fonts, fontErrs := NewFontRegistry(themeDir)
	for _, err := range fontErrs {
		g.initErrors = append(g.initErrors, fmt.Sprintf("font: %v", err))
	}
	g.fonts = fonts

	g.buildComponents(theme.Components)

	return g, nil
}

func (g *Game) buildComponents(components []parser.Component) {
	for _, c := range components {
		switch c.Type {
		case parser.ComponentLabel:
			g.labels = append(g.labels, c)

		case parser.ComponentBootMenu:
			bm := NewBootMenu(c, g.fonts, g.themeDir)
			g.bootMenus = append(g.bootMenus, bm)

		case parser.ComponentProgressBar:
			g.labels = append(g.labels, c)

		case parser.ComponentImage:

		case parser.ComponentVBox, parser.ComponentHBox:
			g.buildComponents(c.Children)
		}
	}
}

func (g *Game) Update() error {
	if ebiten.IsKeyPressed(ebiten.KeyEscape) || ebiten.IsKeyPressed(ebiten.KeyQ) {
		os.Exit(0)
	}

	for _, bm := range g.bootMenus {
		bm.HandleInput()
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	screenDims := Dimensions{Width: g.screenW, Height: g.screenH}

	g.background.Draw(screen)

	for _, c := range g.theme.Components {
		switch c.Type {
		case parser.ComponentLabel:
			DrawLabel(screen, c, g.fonts, screenDims)

		case parser.ComponentBootMenu:
			for _, bm := range g.bootMenus {
				if bm.Component.Line == c.Line {
					bm.Draw(screen, screenDims)
					break
				}
			}
		}
	}

	if len(g.initErrors) > 0 {
		g.drawErrorOverlay(screen)
	}
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.screenW, g.screenH
}

func (g *Game) drawErrorOverlay(screen *ebiten.Image) {
	lineHeight := 18
	padding := 10
	panelH := padding*2 + len(g.initErrors)*lineHeight
	panelW := g.screenW / 2

	panelX := padding
	panelY := g.screenH - panelH - padding

	overlay := ebiten.NewImage(panelW, panelH)
	overlay.Fill(color.RGBA{R: 0x10, G: 0x00, B: 0x00, A: 0xcc})

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(panelX), float64(panelY))
	screen.DrawImage(overlay, op)

	termFont := g.fonts.Lookup(g.theme.TerminalFont)

	for i, msg := range g.initErrors {
		x := panelX + padding
		y := panelY + padding + i*lineHeight + lineHeight

		if termFont != nil {
			DrawText(screen, termFont, "✗ "+msg, x, y, ColorWhite)
		}
	}
}

func Run(theme *parser.Theme, themeDir string) error {
	game, err := NewGame(theme, themeDir)
	if err != nil {
		return fmt.Errorf("initialising renderer: %w", err)
	}

	ebiten.SetWindowSize(game.screenW, game.screenH)
	ebiten.SetWindowTitle("see-grub — " + themeDir)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	ebiten.SetWindowSizeLimits(320, 240, -1, -1)

	return ebiten.RunGame(game)
}
