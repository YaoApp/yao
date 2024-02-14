package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"github.com/fatih/color"
)

// Cli the CLI
type Cli struct {
	option *Option
}

// In the input stream
var reader io.Reader = os.Stdin

// Option the CLI option
type Option struct {
	Label  string
	Reader io.Reader
}

// SetReader set the reader
func SetReader(r io.Reader) {
	reader = r
}

// New create a new CLI
func New(option *Option) *Cli {
	if option.Reader == nil {
		option.Reader = reader
	}
	return &Cli{
		option: option,
	}
}

// Render the CLI UI
func (cli *Cli) Render(args []any) ([]string, error) {

	scanner := bufio.NewScanner(cli.option.Reader)
	var lines []string
	color.Blue("%s", cli.option.Label)
	fmt.Printf("%s", color.WhiteString("> "))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "exit()" {
			break
		}
		lines = append(lines, line)
		fmt.Printf("%s", color.WhiteString("> "))
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}
