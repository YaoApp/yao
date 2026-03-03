package common

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Prompter abstracts user interaction for testability.
type Prompter interface {
	Confirm(message string) bool
	Choose(message string, options []string) int
}

// StdinPrompter reads user input from stdin.
type StdinPrompter struct{}

// Confirm asks a yes/no question. Returns true for "y" or "Y".
func (p *StdinPrompter) Confirm(message string) bool {
	fmt.Printf("%s [Y/n] ", message)
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "" || answer == "y" || answer == "yes"
}

// Choose presents options and returns the 0-based index of the selection.
func (p *StdinPrompter) Choose(message string, options []string) int {
	fmt.Println(message)
	for i, opt := range options {
		fmt.Printf("  [%d] %s\n", i+1, opt)
	}
	fmt.Print("Enter choice: ")
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(answer)

	var choice int
	if _, err := fmt.Sscanf(answer, "%d", &choice); err != nil || choice < 1 || choice > len(options) {
		return -1
	}
	return choice - 1
}

// AutoConfirmPrompter always confirms yes. Used for non-interactive mode and tests.
type AutoConfirmPrompter struct{}

func (p *AutoConfirmPrompter) Confirm(message string) bool           { return true }
func (p *AutoConfirmPrompter) Choose(message string, _ []string) int { return 0 }

// MockPrompter records calls and returns pre-configured responses.
type MockPrompter struct {
	ConfirmResponses []bool
	ChooseResponses  []int
	ConfirmCalls     []string
	ChooseCalls      []string
	confirmIdx       int
	chooseIdx        int
}

func (p *MockPrompter) Confirm(message string) bool {
	p.ConfirmCalls = append(p.ConfirmCalls, message)
	if p.confirmIdx < len(p.ConfirmResponses) {
		resp := p.ConfirmResponses[p.confirmIdx]
		p.confirmIdx++
		return resp
	}
	return true
}

func (p *MockPrompter) Choose(message string, options []string) int {
	p.ChooseCalls = append(p.ChooseCalls, message)
	if p.chooseIdx < len(p.ChooseResponses) {
		resp := p.ChooseResponses[p.chooseIdx]
		p.chooseIdx++
		return resp
	}
	return 0
}
