// Package renderer has all the functions to recreate the grub theme from theme.txt
package renderer

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"
)

func ParseHexColor(hex string) (color.RGBA, error) {
	hex = strings.TrimPrefix(hex, "#")

	switch len(hex) {
	case 8:
		// rrggbbaa
		r, _ := hexByte(hex[0:2])
		g, _ := hexByte(hex[2:4])
		b, _ := hexByte(hex[4:6])
		a, _ := hexByte(hex[6:8])
		return color.RGBA{R: r, G: g, B: b, A: a}, nil
	case 6:
		// Full format: rrggbb
		r, err := hexByte(hex[0:2])
		if err != nil {
			return color.RGBA{}, fmt.Errorf("invalid red component in %q: %w", hex, err)
		}
		g, err := hexByte(hex[2:4])
		if err != nil {
			return color.RGBA{}, fmt.Errorf("invalid green component in %q: %w", hex, err)
		}
		b, err := hexByte(hex[4:6])
		if err != nil {
			return color.RGBA{}, fmt.Errorf("invalid blue component in %q: %w", hex, err)
		}
		return color.RGBA{R: r, G: g, B: b, A: 0xff}, nil

	case 3:
		// Shorthand format: rgb
		r, err := hexByte(string(hex[0]) + string(hex[0]))
		if err != nil {
			return color.RGBA{}, fmt.Errorf("invalid red component in %q: %w", hex, err)
		}
		g, err := hexByte(string(hex[1]) + string(hex[1]))
		if err != nil {
			return color.RGBA{}, fmt.Errorf("invalid green component in %q: %w", hex, err)
		}
		b, err := hexByte(string(hex[2]) + string(hex[2]))
		if err != nil {
			return color.RGBA{}, fmt.Errorf("invalid blue component in %q: %w", hex, err)
		}
		return color.RGBA{R: r, G: g, B: b, A: 0xff}, nil

	default:
		return color.RGBA{}, fmt.Errorf("hex color %q must be 3 or 6 hex digits after '#'", hex)
	}
}

// hexByte parses a two-character hex string (e.g. "b1") into a byte.
func hexByte(hex string) (byte, error) {
	val, err := strconv.ParseUint(hex, 16, 8)
	if err != nil {
		return 0, err
	}
	return byte(val), nil
}

// MustParseHexColor is like ParseHexColor but panics on invalid input.
func MustParseHexColor(hex string) color.RGBA {
	colour, err := ParseHexColor(hex)
	if err != nil {
		panic(fmt.Sprintf("MustParseHexColor: %v", err))
	}
	return colour
}

func FallbackColor(hex string, fallback color.RGBA) color.RGBA {
	colour, err := ParseHexColor(hex)
	if err != nil {
		return fallback
	}
	return colour
}

var (
	ColorWhite       = color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}
	ColorBlack       = color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0xff}
	ColorTransparent = color.RGBA{R: 0x00, G: 0x00, B: 0x00, A: 0x00}
)
