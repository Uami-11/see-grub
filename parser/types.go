// Package parser looks thorugh the theme.txt and parses it and looks for errors
package parser

type Theme struct {
	// ---- desktop values
	DesktopImage string
	DesktopColor string

	// --- terminal values
	TerminalBox          string
	TerminalTop          string
	TerminalLeft         string
	TerminalWidth        string
	TerminalHeight       string
	TerminalBorder       string
	TerminalFont         string
	TerminalBackground   string
	TerminalForeground   string

	TitleText  string
	TitleFont  string
	TitleColor string

	Components []Component
}

type ComponentType string

const (
	ComponentLabel       ComponentType = "label"
	ComponentBootMenu    ComponentType = "boot_menu"
	ComponentProgressBar ComponentType = "progress_bar"
	ComponentVBox        ComponentType = "vbox"
	ComponentHBox        ComponentType = "hbox"
	ComponentCanvas      ComponentType = "canvas"
	ComponentImage       ComponentType = "image"
)

type Component struct {
	Type ComponentType

	// line number in the theme.txt
	Line int

	// Position
	Left   string
	Top    string
	Width  string
	Height string

	// Label fields
	Text  string
	Font  string
	Color string
	Align string
	ID    string

	// Boot menu fields
	ItemFont                string
	ItemColor               string
	SelectedItemColor       string
	ItemHeight              string
	ItemPadding             string
	ItemSpacing             string
	ItemPixmapStyle         string
	SelectedItemPixmapStyle string
	IconWidth               string
	IconHeight              string
	ItemIconSpace           string
	MenuPixmapStyle         string
	Scrollbar               string
	ScrollbarFrame          string
	ScrollbarThumb          string
	MenuBoxSW               string

	// Progress bar fields
	FgColor     string
	BgColor     string
	BorderColor string

	File string

	Children []Component
}
