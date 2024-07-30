package fmt

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/yaoapp/gou/process"
)

// ProcessPrintf utils.fmt.Printf
func ProcessPrintf(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	format := process.ArgsString(0)
	if process.NumOfArgs() == 1 {
		fmt.Print(format)
		return nil
	}
	args := process.Args[1:]
	fmt.Printf(format, args...)
	return nil
}

// ProcessColorPrintf utils.fmt.GreenPrintf
func ProcessColorPrintf(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	colorName := strings.ToLower(process.ArgsString(0))
	format := process.ArgsString(1)
	if process.NumOfArgs() == 1 {
		fmt.Print(format)
		return nil
	}

	args := []interface{}{}
	if process.NumOfArgs() > 2 {
		args = process.Args[2:]
	}

	switch colorName {
	case "red":
		fmt.Print(color.RedString(fmt.Sprintf(format, args...)))
	case "green":
		fmt.Print(color.GreenString(fmt.Sprintf(format, args...)))
	case "yellow":
		fmt.Print(color.YellowString(fmt.Sprintf(format, args...)))
	case "blue":
		fmt.Print(color.BlueString(fmt.Sprintf(format, args...)))
	case "magenta":
		fmt.Print(color.MagentaString(fmt.Sprintf(format, args...)))
	case "cyan":
		fmt.Print(color.CyanString(fmt.Sprintf(format, args...)))
	case "white":
		fmt.Print(color.WhiteString(fmt.Sprintf(format, args...)))
	case "black":
		fmt.Print(color.BlackString(fmt.Sprintf(format, args...)))
	case "hired":
		fmt.Print(color.HiRedString(fmt.Sprintf(format, args...)))
	case "higreen":
		fmt.Print(color.HiGreenString(fmt.Sprintf(format, args...)))
	case "hiyellow":
		fmt.Print(color.HiYellowString(fmt.Sprintf(format, args...)))
	case "hiblue":
		fmt.Print(color.HiBlueString(fmt.Sprintf(format, args...)))
	case "himagenta":
		fmt.Print(color.HiMagentaString(fmt.Sprintf(format, args...)))
	case "hicyan":
		fmt.Print(color.HiCyanString(fmt.Sprintf(format, args...)))
	case "hiwhite":
		fmt.Print(color.HiWhiteString(fmt.Sprintf(format, args...)))
	case "hiblack":
		fmt.Print(color.HiBlackString(fmt.Sprintf(format, args...)))
	default:
		fmt.Print(fmt.Sprintf(format, args...))
	}

	return nil
}
