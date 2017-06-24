package logging

import (
	"fmt"
	"io"
	"syscall"
)

// ColorType represents terminal color
type ColorType uint16

/*
foregroundBlue      = uint16(0x0001)
foregroundGreen     = uint16(0x0002)
foregroundRed       = uint16(0x0004)
foregroundIntensity = uint16(0x0008)
*/
const (
	ColorRed     ColorType = 0x0004 | 0x0008
	ColorGreen   ColorType = 0x0002 | 0x0008
	ColorYellow  ColorType = 0x0004 | 0x0002 | 0x0008
	ColorBlue    ColorType = 0x0001 | 0x0008
	ColorMagenta ColorType = 0x0001 | 0x0004 | 0x0008
	ColorCyan    ColorType = 0x0002 | 0x0001 | 0x0008
	ColorWhite   ColorType = 0x0004 | 0x0001 | 0x0002 | 0x0008
	ColorGray    ColorType = 0x0004 | 0x0002 | 0x0001
)

var (
	kernel32                    = syscall.NewLazyDLL("kernel32.dll")
	procSetConsoleTextAttribute = kernel32.NewProc("SetConsoleTextAttribute")
)

func setConsoleTextAttribute(wAttributes uint16) bool {
	ret, _, _ := procSetConsoleTextAttribute.Call(
		uintptr(syscall.Stdout),
		uintptr(wAttributes))
	return ret != 0
}

// FprintWithColor formats string with terminal colors and writes to w.
// It returns the number of bytes written and any write error encountered.
func FprintWithColor(w io.Writer, str string, code ColorType) (int, error) {
	if setConsoleTextAttribute(uint16(code)) {
		defer setConsoleTextAttribute(uint16(ColorGray))
	}
	return fmt.Fprint(w, str)
}
