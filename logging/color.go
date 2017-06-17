// +build !windows

package logging

import (
	"fmt"
	"io"
)

// ColorType represents terminal color
type ColorType uint16

// Color Code https://en.wikipedia.org/wiki/ANSI_escape_code
// 30–37: set text color to one of the colors 0 to 7,
// 40–47: set background color to one of the colors 0 to 7,
// 39: reset text color to default,
// 49: reset background color to default,
// 1: make text bold / bright (this is the standard way to access the bright color variants),
// 22: turn off bold / bright effect, and
// 0: reset all text properties (color, background, brightness, etc.) to their default values.
// For example, one could select bright purple text on a green background (eww!) with the code `\x1B[35;1;42m`
const (
	ColorRed     ColorType = 31
	ColorGreen   ColorType = 32
	ColorYellow  ColorType = 33
	ColorBlue    ColorType = 34
	ColorMagenta ColorType = 35
	ColorCyan    ColorType = 36
	ColorWhite   ColorType = 37
	ColorGray    ColorType = 90
)

// colorString convert a string to a color string with color code.
func colorString(code int, str string) string {
	return fmt.Sprintf("\x1b[%d;1m%s\x1b[39;22m", code, str)
}

// FprintWithColor formats string with terminal colors and writes to w.
// It returns the number of bytes written and any write error encountered.
func FprintWithColor(w io.Writer, str string, code ColorType) (int, error) {
	return fmt.Fprint(w, colorString(int(code), str))
}
