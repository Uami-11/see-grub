package renderer

import (
	"fmt"
	"strconv"
	"strings"
)

type Dimensions struct {
	Width  int
	Height int
}

type Rect struct {
	X, Y, W, H int
}

func ResolveRect(left, top, width, height string, screen Dimensions) Rect {
	return Rect{
		X: ResolveX(left, screen),
		Y: ResolveY(top, screen),
		W: ResolveDim(width, screen.Width),
		H: ResolveDim(height, screen.Height),
	}
}

func ResolveX(s string, screen Dimensions) int {
	return ResolveDim(s, screen.Width)
}

func ResolveY(s string, screen Dimensions) int {
	return ResolveDim(s, screen.Height)
}

func ResolveDim(s string, reference int) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}

	if strings.HasSuffix(s, "%") {
		pctStr := strings.TrimSuffix(s, "%")
		pct, err := strconv.ParseFloat(pctStr, 64)
		if err != nil {
			return 0
		}
		return int(pct / 100.0 * float64(reference))
	}

	if strings.HasPrefix(s, "+") {
		numStr := strings.TrimPrefix(s, "+")
		n, err := strconv.Atoi(numStr)
		if err != nil {
			return 0
		}
		return reference + n
	}

	if strings.HasPrefix(s, "-") {
		numStr := strings.TrimPrefix(s, "-")
		n, err := strconv.Atoi(numStr)
		if err != nil {
			return 0
		}
		return reference - n
	}

	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}

func ResolveTerminalRect(
	left, top, width, height string,
	screen Dimensions,
) Rect {
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

func (r Rect) String() string {
	return fmt.Sprintf("Rect{x:%d y:%d w:%d h:%d}", r.X, r.Y, r.W, r.H)
}

func (r Rect) Contains(x, y int) bool {
	return x >= r.X && x < r.X+r.W &&
		y >= r.Y && y < r.Y+r.H
}

func (r Rect) Inset(n int) Rect {
	return Rect{
		X: r.X + n,
		Y: r.Y + n,
		W: r.W - n*2,
		H: r.H - n*2,
	}
}
