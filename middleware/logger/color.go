// +build !windows

package logger

import (
	"fmt"
	"io"
)

//ColorType ...
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
	ColorCodeRed     ColorType = 31
	ColorCodeGreen   ColorType = 32
	ColorCodeYellow  ColorType = 33
	ColorCodeBlue    ColorType = 34
	ColorCodeMagenta ColorType = 35
	ColorCodeCyan    ColorType = 36
	ColorCodeWhite   ColorType = 37
	ColorCodeGray    ColorType = 90
)

// colorString convert a string to a color string with color code.
func colorString(code int, str string) string {
	return fmt.Sprintf("\x1b[%d;1m%s\x1b[39;22m ", code, str)
}

//PrintStrWithColor ...
func PrintStrWithColor(w io.Writer, str string, code ColorType) {
	fmt.Fprint(w, colorString(int(code), str))
}
