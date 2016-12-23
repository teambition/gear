package logger

import (
	"fmt"
	"io"
	"syscall"
)

//ColorType ...
type ColorType uint16

/*
foregroundBlue      = uint16(0x0001)
foregroundGreen     = uint16(0x0002)
foregroundRed       = uint16(0x0004)
foregroundIntensity = uint16(0x0008)
*/
const (
	ColorCodeRed     ColorType = 0x0004 | 0x0008
	ColorCodeGreen   ColorType = 0x0002 | 0x0008
	ColorCodeYellow  ColorType = 0x0004 | 0x0002 | 0x0008
	ColorCodeBlue    ColorType = 0x0001 | 0x0008
	ColorCodeMagenta ColorType = 0x0001 | 0x0004 | 0x0008
	ColorCodeCyan    ColorType = 0x0002 | 0x0001 | 0x0008
	ColorCodeWhite   ColorType = 0x0004 | 0x0001 | 0x0002 | 0x0008
	ColorCodeGray    ColorType = 0x0004 | 0x0002 | 0x0001
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

//PrintStrWithColor ...
func PrintStrWithColor(w io.Writer, str string, code ColorType) {
	setConsoleTextAttribute(uint16(code))
	fmt.Fprint(w, str)
	setConsoleTextAttribute(uint16(ColorCodeGray))
}
