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

	switch p.currentBlock.Type {
	case ComponentLabel:
		parsr.applyLabelProperty(key, value)
	case ComponentBootMenu:
		parsr.applyBootMenuProperty(key, value)
	case ComponentProgressBar:
		parsr.applyProgressBarProperty(key, value)
	case ComponentImage:
		parsr.applyImageProperty(key, value)
	case ComponentVBox, ComponentHBox:
		parsr.warnUnknown(key, string(p.currentBlock.Type))
	default:
	}
}
