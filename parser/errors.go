package parser

import (
	"fmt"
	"strings"
)

type Severity int

const (
	SeverityWarning Severity = iota
	SeverityError
)

func (severity Severity) String() string {
	switch severity {
	case SeverityWarning:
		return "warning"
	case SeverityError:
		return "error"
	default:
		return "unknown"
	}
}

type ParseError struct {
	File     string
	Line     int
	Severity Severity
}

func (err ParseError) location() string {
	return fmt.Sprintf("%s:%d: %s:", err.File, err.Line, err.Severity)
}

type ErrMissingTerminalBox struct {
	ParseError
	DesktopImage string
}

func (err ErrMissingTerminalBox) Error() string {
	return fmt.Sprintf(
		"%s background image '%s': terminal-box is not set.\n"+
			"May report \"incorrect format\" for the background PNG.",
		err.location(), err.DesktopImage,
	)
}

type ErrFileNotFound struct {
	ParseError
	Property string
	Path     string
}

func (err ErrFileNotFound) Error() string {
	return fmt.Sprintf("%s %s: file not found: '%s'", err.location(), err.Property, err.Path)
}

type ErrInvalidImageFormat struct {
	ParseError
	Property string
	Path     string
	Got      string
}

func (err ErrInvalidImageFormat) Error() string {
	return fmt.Sprintf("%s %s: '%s' is not a valid PNG. Got %s instead.", err.location(), err.Property, err.Path, err.Got)
}

type ErrFontNotFound struct {
	ParseError
	Property string
	FontName string
	ThemeDir string
}

func (err ErrFontNotFound) Error() string {
	return fmt.Sprintf(
		"%s %s: font '%s' not found.\n"+
			"  Scanned '%s' for .pf2 files, but none had a PFF2NAME matching '%s'.\n"+
			"  Tip: run `strings yourfont.pf2 | head` to check its embedded name.",
		err.location(), err.Property, err.FontName, err.ThemeDir, err.FontName)
}

type ErrUnknownComponent struct {
	ParseError
	Name string
}

func (err ErrUnknownComponent) Error() string {
	return fmt.Sprintf(
		"%s unknown component type: '%s'.\n"+
			"  Known types: label, boot_menu, progress_bar, vbox, hbox, canvas, image.",
		err.location(), err.Name,
	)
}

type ErrUnknownProperty struct {
	ParseError
	Component string
	Key       string
}

func (err ErrUnknownProperty) Error() string {
	return fmt.Sprintf(
		"%s unknown property '%s' in %s (GRUB will ignore this).",
		err.location(), err.Key, err.Component,
	)
}

type ErrBadValue struct {
	ParseError
	Property string
	Value    string
	Reason   string
}

func (err ErrBadValue) Error() string {
	return fmt.Sprintf(
		"%s bad value for '%s': '%s' — %s",
		err.location(), err.Property, err.Value, err.Reason,
	)
}

type ErrMalformedLined struct {
	ParseError
	RawLine string
}

func (err ErrMalformedLined) Error() string {
	return fmt.Sprintf(
		"%s malformed line (expected 'key: value', 'key = value', or '+ type {'): %q",
		err.location(), err.RawLine,
	)
}

type ErrGlobMatch struct {
	ParseError
	Property string
	Pattern  string
	ThemeDir string
}

func (err ErrGlobMatch) Error() string {
	return fmt.Sprintf(
		"%s %s: glob pattern '%s' matched no files in '%s'.",
		err.location(), err.Property, err.Pattern, err.ThemeDir,
	)
}

type ErrorList struct {
	Errors []error
}

func (errorList *ErrorList) Add(err error) {
	errorList.Errors = append(errorList.Errors, err)
}

func (errorList *ErrorList) HasErrors() bool {
	for _, err := range errorList.Errors {
		switch e := err.(type) {
		case ErrMissingTerminalBox:
			if e.Severity == SeverityError {
				return true
			}
		case ErrFileNotFound:
			if e.Severity == SeverityError {
				return true
			}
		case ErrInvalidImageFormat:
			if e.Severity == SeverityError {
				return true
			}
		case ErrFontNotFound:
			if e.Severity == SeverityError {
				return true
			}
		case ErrUnknownComponent:
			if e.Severity == SeverityError {
				return true
			}
		case ErrBadValue:
			if e.Severity == SeverityError {
				return true
			}
		case ErrMalformedLined:
			if e.Severity == SeverityError {
				return true
			}
		case ErrGlobMatch:
			if e.Severity == SeverityError {
				return true
			}
		}
	}
	return false
}

func (errorList *ErrorList) Error() string {
	var msg strings.Builder

	for i, err := range errorList.Errors {
		if i > 0 {
			msg.WriteRune('\n')
		}
		msg.WriteString(err.Error())
	}

	return msg.String()
}
