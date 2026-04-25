package renderer

import (
	"fmt"
	"strconv"
	"strings"
)

// Dimensions holds the resolved screen (or container) size that percentage
// and relative values are calculated against.
// For top-level components this is the background image size.
// For children inside a vbox/hbox it would be the container's size.
type Dimensions struct {
	Width  int
	Height int
}

// Rect is a fully resolved rectangle in pixels.
// This is what every component gets handed at draw time.
type Rect struct {
	X, Y, W, H int
}

// ResolveRect resolves all four raw GRUB dimension strings for a component
// into a concrete pixel Rect, given the screen dimensions.
//
// Each value is resolved independently:
//   - left/top are resolved against screen width/height
//   - width/height are resolved against screen width/height
func ResolveRect(left, top, width, height string, screen Dimensions) Rect {
	return Rect{
		X: ResolveX(left, screen),
		Y: ResolveY(top, screen),
		W: ResolveDim(width, screen.Width),
		H: ResolveDim(height, screen.Height),
	}
}

// ResolveX resolves a horizontal position string against the screen width.
func ResolveX(s string, screen Dimensions) int {
	return ResolveDim(s, screen.Width)
}

// ResolveY resolves a vertical position string against the screen height.
func ResolveY(s string, screen Dimensions) int {
	return ResolveDim(s, screen.Height)
}

// ResolveDim resolves a single GRUB dimension string into pixels.
//
// GRUB supports four formats:
//
//	"600"    — absolute pixel value
//	"100%"   — percentage of the reference dimension (screen width or height)
//	"+10"    — 10px from the right/bottom edge (rare, used in some themes)
//	"-10"    — 10px from the right/bottom edge inward
//
// If the string is empty or unparseable, 0 is returned — matching GRUB's
// own silent fallback behavior.
func ResolveDim(s string, reference int) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	// Percentage: "100%", "50%", etc.
	if strings.HasSuffix(s, "%") {
		pctStr := strings.TrimSuffix(s, "%")
		pct, err := strconv.ParseFloat(pctStr, 64)
		if err != nil {
			return 0
		}
		return int(pct / 100.0 * float64(reference))
	}

	// Relative from edge: "+N" means N pixels from the far edge.
	// GRUB uses this for right/bottom anchoring.
	// e.g. width = "+0" means "fill to the right edge"
	if strings.HasPrefix(s, "+") {
		numStr := strings.TrimPrefix(s, "+")
		n, err := strconv.Atoi(numStr)
		if err != nil {
			return 0
		}
		return reference + n
	}

	// Negative offset from edge: "-N"
	if strings.HasPrefix(s, "-") {
		numStr := strings.TrimPrefix(s, "-")
		n, err := strconv.Atoi(numStr)
		if err != nil {
			return 0
		}
		return reference - n
	}

	// Plain integer: "308", "0", "1920"
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

// ResolveTerminalRect resolves the terminal-* global properties into a Rect.
// The terminal box defines the region GRUB composites the background image
// into — it's the outermost bounding box of the theme.
//
// Defaults: if any value is missing, it fills the full screen.
// This matches GRUB's own default behavior.
func ResolveTerminalRect(
	left, top, width, height string,
	screen Dimensions,
) Rect {
	// Default to full screen if not specified.
	if left == "" {
		left = "0"
	}
	if top == "" {
		top = "0"
	}
	if width == "" {
		width = "100%"
	}
	if height == "" {
		height = "100%"
	}

	return ResolveRect(left, top, width, height, screen)
}

// String returns a human-readable representation of a Rect for debugging.
func (r Rect) String() string {
	return fmt.Sprintf("Rect{x:%d y:%d w:%d h:%d}", r.X, r.Y, r.W, r.H)
}

// Contains reports whether the point (x, y) is inside the rect.
// Useful for mouse hit testing later if we add hover states.
func (r Rect) Contains(x, y int) bool {
	return x >= r.X && x < r.X+r.W &&
		y >= r.Y && y < r.Y+r.H
}

// Inset returns a new Rect shrunk by n pixels on all sides.
// Used for applying item_padding inside the boot menu.
func (r Rect) Inset(n int) Rect {
	return Rect{
		X: r.X + n,
		Y: r.Y + n,
		W: r.W - n*2,
		H: r.H - n*2,
	}
}
