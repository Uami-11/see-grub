package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Uami-11/see-grub/parser"
	"github.com/Uami-11/see-grub/renderer"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorGreen  = "\033[32m"
	colorCyan   = "\033[36m"
	colorBold   = "\033[1m"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: see-grub <theme-directory or theme.txt path>\n")
		os.Exit(1)
	}

	themePath := resolveThemePath(os.Args[1])
	if themePath == "" {
		fmt.Fprintf(os.Stderr, "%serror:%s could not find theme.txt in '%s'\n",
			colorRed, colorReset, os.Args[1])
		os.Exit(1)
	}

	fmt.Printf("%s%ssee-grub — theme diagnostics%s\n", colorBold, colorCyan, colorReset)
	fmt.Printf("Parsing: %s\n\n", themePath)

	// In main(), after resolving themePath, check for --gfxmode flag
	var gfxW, gfxH int
	for _, arg := range os.Args[2:] {
		if strings.HasPrefix(arg, "--gfxmode=") {
			val := strings.TrimPrefix(arg, "--gfxmode=")
			// Strip ,auto or ,keep suffixes
			val = strings.Split(val, ",")[0]
			parts := strings.Split(val, "x")
			if len(parts) == 2 {
				gfxW, _ = strconv.Atoi(parts[0])
				gfxH, _ = strconv.Atoi(parts[1])
			}
		}
	}

	theme, errs := parser.Parse(themePath)

	printErrorList(errs)
	printTheme(theme)

	if errs.HasErrors() {
		fmt.Fprintf(os.Stderr, "\n%sCannot open preview: theme has errors (see above).%s\n",
			colorRed, colorReset)
		os.Exit(1)
	}

	fmt.Printf("%sLaunching preview...%s\n", colorGreen, colorReset)
	fmt.Printf("  Press ESC or Q to quit.\n\n")

	themeDir := resolveThemeDir(themePath)
	if err := renderer.Run(theme, themeDir, gfxW, gfxH); err != nil {
		fmt.Fprintf(os.Stderr, "%srenderer error: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}
}

func resolveThemePath(arg string) string {
	info, err := os.Stat(arg)
	if err != nil {
		return ""
	}
	if info.IsDir() {
		candidate := filepath.Join(arg, "theme.txt")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		return ""
	}
	return arg
}

func resolveThemeDir(themePath string) string {
	info, err := os.Stat(themePath)
	if err != nil {
		return filepath.Dir(themePath)
	}
	if info.IsDir() {
		return themePath
	}
	return filepath.Dir(themePath)
}

func printErrorList(errs *parser.ErrorList) {
	if len(errs.Errors) == 0 {
		fmt.Printf("%s✓ No errors or warnings%s\n\n", colorGreen, colorReset)
		return
	}

	warnings := 0
	errors := 0

	for _, err := range errs.Errors {
		msg := err.Error()
		if isWarning(err) {
			warnings++
			fmt.Printf("%s⚠ %s%s\n", colorYellow, msg, colorReset)
		} else {
			errors++
			fmt.Printf("%s✗ %s%s\n", colorRed, msg, colorReset)
		}
	}

	fmt.Printf("\n")

	if errors > 0 {
		fmt.Printf("%s%d error(s)%s", colorRed, errors, colorReset)
	}
	if warnings > 0 {
		if errors > 0 {
			fmt.Printf(", ")
		}
		fmt.Printf("%s%d warning(s)%s", colorYellow, warnings, colorReset)
	}
	fmt.Printf("\n\n")
}

func isWarning(err error) bool {
	switch e := err.(type) {
	case parser.ErrUnknownProperty:
		return e.Severity == parser.SeverityWarning
	case parser.ErrGlobMatch:
		return e.Severity == parser.SeverityWarning
	case parser.ErrMissingTerminalBox:
		return e.Severity == parser.SeverityWarning
	case parser.ErrFileNotFound:
		return e.Severity == parser.SeverityWarning
	case parser.ErrInvalidImageFormat:
		return e.Severity == parser.SeverityWarning
	case parser.ErrFontNotFound:
		return e.Severity == parser.SeverityWarning
	case parser.ErrBadValue:
		return e.Severity == parser.SeverityWarning
	case parser.ErrMalformedLined:
		return e.Severity == parser.SeverityWarning
	case parser.ErrUnknownComponent:
		return e.Severity == parser.SeverityWarning
	}
	return false
}

func printTheme(theme *parser.Theme) {
	header("Global Options")

	field("title-text", theme.TitleText)
	field("title-font", theme.TitleFont)
	field("title-color", theme.TitleColor)
	field("desktop-image", theme.DesktopImage)
	field("desktop-color", theme.DesktopColor)
	field("terminal-font", theme.TerminalFont)
	field("terminal-box", theme.TerminalBox)
	field("terminal-left", theme.TerminalLeft)
	field("terminal-top", theme.TerminalTop)
	field("terminal-width", theme.TerminalWidth)
	field("terminal-height", theme.TerminalHeight)
	field("terminal-border", theme.TerminalBorder)
	field("terminal-background", theme.TerminalBackground)
	field("terminal-foreground", theme.TerminalForeground)

	fmt.Println()
	header(fmt.Sprintf("Components (%d)", len(theme.Components)))

	for i, c := range theme.Components {
		printComponent(i, c)
	}

	fmt.Println()
}

func printComponent(index int, c parser.Component) {
	fmt.Printf("\n  %s%s[%d] + %s%s (line %d)\n",
		colorBold, colorCyan, index, string(c.Type), colorReset, c.Line)

	if c.Left != "" || c.Top != "" {
		fmt.Printf("    position : left=%s top=%s\n", orEmpty(c.Left), orEmpty(c.Top))
	}
	if c.Width != "" || c.Height != "" {
		fmt.Printf("    size     : width=%s height=%s\n", orEmpty(c.Width), orEmpty(c.Height))
	}

	switch c.Type {
	case parser.ComponentLabel:
		field2("text", c.Text)
		field2("font", c.Font)
		field2("color", c.Color)
		field2("align", c.Align)
		if c.ID != "" {
			field2("id", c.ID)
		}
	case parser.ComponentBootMenu:
		field2("item_font", c.ItemFont)
		field2("item_color", c.ItemColor)
		field2("selected_item_color", c.SelectedItemColor)
		field2("item_height", c.ItemHeight)
		field2("item_padding", c.ItemPadding)
		field2("item_spacing", c.ItemSpacing)
		field2("item_pixmap_style", c.ItemPixmapStyle)
		field2("selected_item_pixmap_style", c.SelectedItemPixmapStyle)
		field2("icon_width", c.IconWidth)
		field2("icon_height", c.IconHeight)
		field2("item_icon_space", c.ItemIconSpace)
		field2("menu_pixmap_style", c.MenuPixmapStyle)
		field2("scrollbar", c.Scrollbar)
		field2("scrollbar_frame", c.ScrollbarFrame)
		field2("scrollbar_thumb", c.ScrollbarThumb)
		field2("menu_box_sw", c.MenuBoxSW)
	case parser.ComponentProgressBar:
		field2("fg_color", c.FgColor)
		field2("bg_color", c.BgColor)
		field2("border_color", c.BorderColor)
	case parser.ComponentImage:
		field2("file", c.File)
	}

	if len(c.Children) > 0 {
		fmt.Printf("    children : %d\n", len(c.Children))
		for j, child := range c.Children {
			printComponent(j, child)
		}
	}
}

func header(title string) {
	fmt.Printf("%s%s=== %s ===%s\n", colorBold, colorCyan, title, colorReset)
}

func field(name, value string) {
	if value == "" {
		fmt.Printf("  %-20s %s(not set)%s\n", name, colorYellow, colorReset)
	} else {
		fmt.Printf("  %-20s %s\n", name, value)
	}
}

func field2(name, value string) {
	if value == "" {
		return
	}
	fmt.Printf("    %-28s %s\n", name, value)
}

func orEmpty(s string) string {
	if s == "" {
		return "(not set)"
	}
	return s
}

func divider() {
	fmt.Println(strings.Repeat("─", 60))
}

var _ = divider
