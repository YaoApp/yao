package task

import (
	"fmt"
	"strings"
	"time"

	"github.com/yaoapp/yao/config"
)

const (
	logColorReset     = "\033[0m"
	logColorGreen     = "\033[32m"
	logColorRed       = "\033[31m"
	logColorCyan      = "\033[36m"
	logColorWhite     = "\033[37m"
	logColorGray      = "\033[90m"
	logColorBoldGreen = "\033[1;32m"
	logColorBoldRed   = "\033[1;31m"
	logColorBoldCyan  = "\033[1;36m"
)

func logTaskCreated(chatID, columnID, assistantID string) {
	if !config.IsDevelopment() {
		return
	}
	fmt.Println()
	fmt.Printf("%s%s%s\n", logColorBoldGreen, strings.Repeat("═", 60), logColorReset)
	fmt.Printf("%s  TASK CREATED%s\n", logColorBoldGreen, logColorReset)
	fmt.Printf("%s%s%s\n", logColorGreen, strings.Repeat("─", 60), logColorReset)
	fmt.Printf("%s  Chat ID:   %s%s%s\n", logColorGray, logColorWhite, chatID, logColorReset)
	fmt.Printf("%s  Column:    %s%s%s\n", logColorGray, logColorWhite, columnID, logColorReset)
	fmt.Printf("%s  Assistant: %s%s%s\n", logColorGray, logColorWhite, assistantID, logColorReset)
	fmt.Printf("%s  Time:      %s%s%s\n", logColorGray, logColorWhite, time.Now().Format("15:04:05.000"), logColorReset)
	fmt.Printf("%s%s%s\n", logColorGreen, strings.Repeat("─", 60), logColorReset)
}

func logTaskCompleted(chatID, columnID, assistantID, status string, duration time.Duration, err error) {
	if !config.IsDevelopment() {
		return
	}
	fmt.Printf("%s%s%s\n", logColorCyan, strings.Repeat("─", 60), logColorReset)
	if err != nil || status == "failed" {
		fmt.Printf("%s  TASK FAILED%s\n", logColorBoldRed, logColorReset)
	} else {
		fmt.Printf("%s  TASK COMPLETED%s\n", logColorBoldGreen, logColorReset)
	}
	fmt.Printf("%s  Chat ID:   %s%s%s\n", logColorGray, logColorWhite, chatID, logColorReset)
	fmt.Printf("%s  Column:    %s%s%s\n", logColorGray, logColorWhite, columnID, logColorReset)
	fmt.Printf("%s  Assistant: %s%s%s\n", logColorGray, logColorWhite, assistantID, logColorReset)
	fmt.Printf("%s  Status:    %s%s%s\n", logColorGray, logColorWhite, status, logColorReset)
	if err != nil {
		fmt.Printf("%s  Error:     %s%s%s\n", logColorGray, logColorRed, err.Error(), logColorReset)
	}
	fmt.Printf("%s  Duration:  %s%s%s\n", logColorGray, logColorWhite, duration.Round(time.Millisecond), logColorReset)
	fmt.Printf("%s%s%s\n", logColorCyan, strings.Repeat("─", 60), logColorReset)
}
