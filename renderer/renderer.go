package renderer

import (
	"fmt"
	"image/color"
	"os"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/Uami-11/see-grub/parser"
)

type loadedImage struct {
	Component parser.Component
	Image     *ebiten.Image
}

type Game struct {
	theme    *parser.Theme
	themeDir string

	screenW int
	screenH int

	designW int
	designH int

	background *Background
	fonts      *FontRegistry

	bootMenus []*BootMenu
	images    []loadedImage

	offscreen *ebiten.Image
	errPanel  *ebiten.Image

	initErrors []string
}

func NewGame(theme *parser.Theme, themeDir string, gfxW, gfxH int) (*Game, error) {
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
		g.designW = bg.Width
		g.designH = bg.Height
	} else {
		g.designW = 1920
		g.designH = 1080
	}

	if gfxW > 0 && gfxH > 0 {
		g.screenW = gfxW
		g.screenH = gfxH
	} else {
		g.screenW = g.designW
		g.screenH = g.designH
	}

	fonts, fontErrs := NewFontRegistry(themeDir)
	for _, err := range fontErrs {
		g.initErrors = append(g.initErrors, fmt.Sprintf("font: %v", err))
	}
	g.fonts = fonts

	g.offscreen = ebiten.NewImage(g.designW, g.designH)

	g.buildComponents(theme.Components)

	return g, nil
}

func (g *Game) buildComponents(components []parser.Component) {
	for _, c := range components {
		switch c.Type {
		case parser.ComponentBootMenu:
			bm := NewBootMenu(c, g.fonts, g.themeDir)
			g.bootMenus = append(g.bootMenus, bm)

		case parser.ComponentImage:
			path := resolvePath(g.themeDir, c.File)
			img, _, _, err := loadPNG(path)
			if err == nil {
				g.images = append(g.images, loadedImage{Component: c, Image: img})
			}

		case parser.ComponentVBox, parser.ComponentHBox, parser.ComponentCanvas:
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
	g.background.Draw(screen)

	g.offscreen.Clear()

	termRect := ResolveTerminalRect(
		g.theme.TerminalLeft, g.theme.TerminalTop,
		g.theme.TerminalWidth, g.theme.TerminalHeight,
		Dimensions{Width: g.designW, Height: g.designH})

	// Components are positioned relative to the terminal box.
	// screenDims controls percentage resolution in child components.
	screenDims := Dimensions{Width: termRect.W, Height: termRect.H}

	g.drawComponents(g.offscreen, g.theme.Components, termRect, screenDims)

	op := &ebiten.DrawImageOptions{}
	op.Filter = ebiten.FilterNearest
	scaleX := float64(g.screenW) / float64(g.designW)
	scaleY := float64(g.screenH) / float64(g.designH)
	op.GeoM.Scale(scaleX, scaleY)
	screen.DrawImage(g.offscreen, op)

	if len(g.initErrors) > 0 {
		g.drawErrorOverlay(screen)
	}
}

func (g *Game) drawComponents(dst *ebiten.Image, components []parser.Component, termRect Rect, screenDims Dimensions) {
	for _, c := range components {
		adjusted := c
		adjusted.Left = strconv.Itoa(ResolveDim(c.Left, screenDims.Width) + termRect.X)
		adjusted.Top = strconv.Itoa(ResolveDim(c.Top, screenDims.Height) + termRect.Y)

		switch c.Type {
		case parser.ComponentLabel:
			DrawLabel(dst, adjusted, g.fonts, screenDims)

		case parser.ComponentBootMenu:
			for _, bm := range g.bootMenus {
				if bm.Component.Line == c.Line {
					bmCopy := *bm
					bmCopy.Component = adjusted
					bmCopy.Draw(dst, screenDims)
					break
				}
			}

		case parser.ComponentProgressBar:
			DrawProgressBar(dst, adjusted, screenDims)

		case parser.ComponentImage:
			for _, entry := range g.images {
				if entry.Component.Line == c.Line {
					drawImage(dst, entry.Image, adjusted, screenDims)
					break
				}
			}

		case parser.ComponentVBox, parser.ComponentHBox, parser.ComponentCanvas:
			g.drawComponents(dst, c.Children, termRect, screenDims)
		}
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

	if g.errPanel == nil || g.errPanel.Bounds().Dx() != panelW || g.errPanel.Bounds().Dy() != panelH {
		g.errPanel = ebiten.NewImage(panelW, panelH)
		g.errPanel.Fill(color.RGBA{R: 0x10, G: 0x00, B: 0x00, A: 0xcc})
	}

	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(float64(panelX), float64(panelY))
	screen.DrawImage(g.errPanel, op)

	termFont := g.fonts.Lookup(g.theme.TerminalFont)

	for i, msg := range g.initErrors {
		x := panelX + padding
		y := panelY + padding + i*lineHeight + lineHeight

		if termFont != nil {
			DrawText(screen, termFont, "✗ "+msg, x, y, ColorWhite)
		}
	}
}

func drawImage(dst *ebiten.Image, img *ebiten.Image, c parser.Component, screen Dimensions) {
	rect := ResolveRect(c.Left, c.Top, c.Width, c.Height, screen)
	if rect.W <= 0 {
		rect.W = img.Bounds().Dx()
	}
	if rect.H <= 0 {
		rect.H = img.Bounds().Dy()
	}

	op := &ebiten.DrawImageOptions{}
	sx := float64(rect.W) / float64(img.Bounds().Dx())
	sy := float64(rect.H) / float64(img.Bounds().Dy())
	op.GeoM.Scale(sx, sy)
	op.GeoM.Translate(float64(rect.X), float64(rect.Y))
	dst.DrawImage(img, op)
}

func Run(theme *parser.Theme, themeDir string, gfxW, gfxH int) error {
	game, err := NewGame(theme, themeDir, gfxW, gfxH)
	if err != nil {
		return fmt.Errorf("initialising renderer: %w", err)
	}

	ebiten.SetWindowTitle("see-grub — " + themeDir)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowSizeLimits(320, 240, -1, -1)

	// Start at logical resolution — Ebitengine scales to window size
	// The user can resize/maximize to see the theme at full scale
	ebiten.SetWindowSize(game.screenW, game.screenH)
	fmt.Printf("Init errors (%d):\n", len(game.initErrors))
	for _, e := range game.initErrors {
		fmt.Println(" ", e)
	}

	return ebiten.RunGame(game)
}
