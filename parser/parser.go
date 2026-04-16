package parser

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

type parserState int

const (
	stateTopLevel parserState = iota
	stateInBlock
)

type Parser struct {
	filePath       string
	themeDir       string
	errors         ErrorList
	state          parserState
	currentLine    int
	currentBlock   Component
	inNestedBlock  bool
	nestedChildren []Component
}

func Parse(themePath string) (*Theme, *ErrorList) {
	absPath, err := filepath.Abs(themePath)
	if err != nil {
		errList := &ErrorList{}
		errList.Add(ErrMalformedLined{
			ParseError: ParseError{
				File:     themePath,
				Line:     0,
				Severity: SeverityError,
			},
			RawLine: "Could not resolve the path: " + err.Error(),
		})

		return &Theme{}, errList
	}

	fileTheme, err := os.Open(absPath)
	if err != nil {
		errList := &ErrorList{}
		errList.Add(ErrFileNotFound{
			ParseError: ParseError{
				File:     absPath,
				Line:     0,
				Severity: SeverityError,
			},
			Property: "theme file",
			Path:     absPath,
		})
		return &Theme{}, errList
	}

	defer fileTheme.Close()

	theParser := &Parser{
		filePath: absPath,
		themeDir: filepath.Dir(absPath),
		state:    stateTopLevel,
	}

	theme := theParser.parse(fileTheme)
	theParser.validate(theme)

	return theme, &theParser.errors
}

func (parsr *Parser) parse(fileTheme *os.File) *Theme {
	theme := &Theme{}
	scanner := bufio.NewScanner(fileTheme)

	for scanner.Scan() {
		parsr.currentLine++
		raw := scanner.Text()

		line := strings.TrimSpace(raw)

		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		switch parsr.state {
		case stateTopLevel:
			parsr.parseTopLevel(line, theme)
		case stateInBlock:
			parsr.parseInBlock(line, theme)
		}
	}

	if parsr.state == stateInBlock {
		parsr.errors.Add(ErrMalformedLined{
			ParseError: ParseError{
				File:     parsr.filePath,
				Line:     parsr.currentLine,
				Severity: SeverityError,
			},
			RawLine: "unexpected end of file: missing closing }",
		})
	}

	return theme
}

func (parsr *Parser) parseTopLevel(line string, theme *Theme) {
	if strings.HasPrefix(line, "+") {
		parsr.openBlock(line)
		return
	}

	key, value, ok := parseKeyValue(line, ":")
	if !ok {
		parsr.errors.Add(ErrMalformedLined{
			ParseError: ParseError{
				File:     parsr.filePath,
				Line:     parsr.currentLine,
				Severity: SeverityError,
			},
			RawLine: line,
		})
		return
	}

	parsr.applyGlobal(key, value, theme)
}

func (parsr *Parser) openBlock(line string) {
	inner := strings.TrimPrefix(line, "+")
	inner = strings.TrimSpace(inner)

	if !strings.HasSuffix(inner, "{") {
		parsr.errors.Add(ErrMalformedLined{
			ParseError: ParseError{
				File:     parsr.filePath,
				Line:     parsr.currentLine,
				Severity: SeverityError,
			},
			RawLine: line,
		})
		return
	}

	typeName := strings.TrimSuffix(inner, "{")
	typeName = strings.TrimSpace(typeName)

	componentType := ComponentType(typeName)

	switch componentType {
	case ComponentLabel, ComponentBootMenu, ComponentProgressBar,
		ComponentVBox, ComponentHBox, ComponentImage:
		parsr.currentBlock = Component{
			Type: componentType,
			Line: parsr.currentLine,
		}
		parsr.state = stateInBlock
	default:
		parsr.errors.Add(ErrUnknownComponent{
			ParseError: ParseError{
				File:     parsr.filePath,
				Line:     parsr.currentLine,
				Severity: SeverityError,
			},
			Name: typeName,
		})
		parsr.currentBlock = Component{
			Type: "unknown",
			Line: parsr.currentLine,
		}
		parsr.state = stateInBlock
	}
}

func (parsr *Parser) applyGlobal(key, value string, theme *Theme) {
	switch key {
	case "title-text":
		theme.TitleText = value
	case "desktop-image":
		theme.DesktopImage = value
	case "desktop-color":
		theme.DesktopColor = value
	case "terminal-font":
		theme.TerminalFont = value
	case "terminal-box":
		theme.TerminalBox = value
	case "terminal-left":
		theme.TerminalLeft = value
	case "terminal-top":
		theme.TerminalTop = value
	case "terminal-width":
		theme.TerminalWidth = value
	case "terminal-height":
		theme.TerminalHeight = value
	case "terminal-border":
		theme.TerminalBorder = value
	default:
		parsr.errors.Add(ErrUnknownProperty{
			ParseError: ParseError{
				File:     parsr.filePath,
				Line:     parsr.currentLine,
				Severity: SeverityWarning,
			},
			Component: "global",
			Key:       key,
		})
	}
}

func (parsr *Parser) parseInBlock(line string, theme *Theme) {
	if line == "}" {
		parsr.closeBlock(theme)
		return
	}

	key, value, ok := parseKeyValue(line, "=")
	if !ok {
		parsr.errors.Add(ErrMalformedLined{
			ParseError: ParseError{
				File:     parsr.filePath,
				Line:     parsr.currentLine,
				Severity: SeverityError,
			},
			RawLine: line,
		})
		return
	}

	parsr.applyProperty(key, value)
}

func (parsr *Parser) applyProperty(key, value string) {
	switch key {
	case "left":
		parsr.currentBlock.Left = value
		return
	case "top":
		parsr.currentBlock.Top = value
		return
	case "width":
		parsr.currentBlock.Width = value
		return
	case "height":
		parsr.currentBlock.Height = value
		return
	}

	switch parsr.currentBlock.Type {
	case ComponentLabel:
		parsr.applyLabelProperty(key, value)
	case ComponentBootMenu:
		parsr.applyBootMenuProperty(key, value)
	case ComponentProgressBar:
		parsr.applyProgressBarProperty(key, value)
	case ComponentImage:
		parsr.applyImageProperty(key, value)
	case ComponentVBox, ComponentHBox:
		parsr.warnUnknown(key, string(parsr.currentBlock.Type))
	default:
	}
}

func (parsr *Parser) applyLabelProperty(key, value string) {
	switch key {
	case "text":
		parsr.currentBlock.Text = value
	case "font":
		parsr.currentBlock.Font = value
	case "color":
		parsr.currentBlock.Color = value
	case "align":
		parsr.currentBlock.Align = value
	case "id":
		parsr.currentBlock.ID = value
	default:
		parsr.warnUnknown(key, "label")
	}
}

func (parsr *Parser) applyBootMenuProperty(key, value string) {
	switch key {
	case "item_font":
		parsr.currentBlock.ItemFont = value
	case "item_color":
		parsr.currentBlock.ItemColor = value
	case "selected_item_color":
		parsr.currentBlock.SelectedItemColor = value
	case "item_height":
		parsr.currentBlock.ItemHeight = value
	case "item_padding":
		parsr.currentBlock.ItemPadding = value
	case "item_spacing":
		parsr.currentBlock.ItemSpacing = value
	case "item_pixmap_style":
		parsr.currentBlock.ItemPixmapStyle = value
	case "selected_item_pixmap_style":
		parsr.currentBlock.SelectedItemPixmapStyle = value
	case "icon_width":
		parsr.currentBlock.IconWidth = value
	case "icon_height":
		parsr.currentBlock.IconHeight = value
	case "item_icon_space":
		parsr.currentBlock.ItemIconSpace = value
	default:
		parsr.warnUnknown(key, "boot_menu")
	}
}

func (parsr *Parser) applyProgressBarProperty(key, value string) {
	switch key {
	case "fg_color":
		parsr.currentBlock.FgColor = value
	case "bg_color":
		parsr.currentBlock.BgColor = value
	case "border_color":
		parsr.currentBlock.BorderColor = value
	default:
		parsr.warnUnknown(key, "progress_bar")
	}
}

func (parsr *Parser) applyImageProperty(key, value string) {
	switch key {
	case "file":
		parsr.currentBlock.File = value
	default:
		parsr.warnUnknown(key, "image")
	}
}

// warnUnknown adds an ErrUnknownProperty warning for an unrecognised key.
func (parsr *Parser) warnUnknown(key, component string) {
	parsr.errors.Add(ErrUnknownProperty{
		ParseError: ParseError{
			File:     parsr.filePath,
			Line:     parsr.currentLine,
			Severity: SeverityWarning,
		},
		Component: component,
		Key:       key,
	})
}

func (parsr *Parser) validate(theme *Theme) {
	if theme.DesktopImage != "" && theme.TerminalBox == "" {
		parsr.errors.Add(ErrMissingTerminalBox{
			ParseError: ParseError{
				File:     parsr.filePath,
				Line:     0,
				Severity: SeverityError,
			},
			DesktopImage: theme.DesktopImage,
		})
	}

	// Validate that asset files actually exist on disk.
	if theme.DesktopImage != "" {
		parsr.checkFileExists("desktop-image", theme.DesktopImage, 0)
	}

	// Validate component assets.
	for _, c := range theme.Components {
		switch c.Type {
		case ComponentBootMenu:
			if c.ItemPixmapStyle != "" {
				parsr.checkGlob("item_pixmap_style", c.ItemPixmapStyle, c.Line)
			}
			if c.SelectedItemPixmapStyle != "" {
				parsr.checkGlob("selected_item_pixmap_style", c.SelectedItemPixmapStyle, c.Line)
			}
		case ComponentImage:
			if c.File != "" {
				parsr.checkFileExists("file", c.File, c.Line)
			}
		}
	}
}

// checkFileExists verifies a file referenced in the theme exists on disk.
// Paths are resolved relative to the theme directory.
func (parsr *Parser) checkFileExists(property, path string, line int) {
	resolved := parsr.resolvePath(path)
	if _, err := os.Stat(resolved); os.IsNotExist(err) {
		parsr.errors.Add(ErrFileNotFound{
			ParseError: ParseError{
				File:     parsr.filePath,
				Line:     line,
				Severity: SeverityError,
			},
			Property: property,
			Path:     resolved,
		})
	}
}

// checkGlob verifies that a pixmap style glob matches at least one file.
func (parsr *Parser) checkGlob(property, pattern string, line int) {
	resolved := parsr.resolvePath(pattern)
	matches, err := filepath.Glob(resolved)
	if err != nil || len(matches) == 0 {
		parsr.errors.Add(ErrGlobMatch{
			ParseError: ParseError{
				File:     parsr.filePath,
				Line:     line,
				Severity: SeverityWarning,
			},
			Property: property,
			Pattern:  pattern,
			ThemeDir: parsr.themeDir,
		})
	}
}

// resolvePath resolves a theme-relative path to an absolute path.
// If the path is already absolute, it is returned unchanged.
func (parsr *Parser) resolvePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(parsr.themeDir, path)
}

// parseKeyValue splits a line on the first occurrence of sep (":" or "=")
// and returns the trimmed key and unquoted value.
// Returns ok=false if sep is not found.
func parseKeyValue(line, sep string) (key, value string, ok bool) {
	idx := strings.Index(line, sep)
	if idx < 0 {
		return "", "", false
	}

	key = strings.TrimSpace(line[:idx])
	value = strings.TrimSpace(line[idx+len(sep):])

	// Strip surrounding quotes from values.
	// Both `"value"` and `value` are valid in theme.txt.
	value = strings.Trim(value, `"`)

	return key, value, true
}

func (p *Parser) closeBlock(theme *Theme) {
	if p.currentBlock.Type != "unknown" {
		theme.Components = append(theme.Components, p.currentBlock)
	}
	// Reset state.
	p.currentBlock = Component{}
	p.state = stateTopLevel
}
