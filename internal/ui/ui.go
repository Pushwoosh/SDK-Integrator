// Package ui talks to the person at the terminal; install.sh feeds the binary through a pipe, so it reads /dev/tty.
package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func Say(format string, args ...any) { fmt.Printf(format+"\n", args...) }

func Fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "pushwoosh-integrator: "+format+"\n", args...)
	os.Exit(1)
}

// Confirm asks a yes/no question with yes as the default; without a terminal it answers no.
func Confirm(question string) bool {
	tty, err := os.Open("/dev/tty")
	if err != nil {
		return false
	}
	defer tty.Close()

	fmt.Printf("%s [Y/n] ", question)
	answer, _ := bufio.NewReader(tty).ReadString('\n')
	answer = strings.ToLower(strings.TrimSpace(answer))

	return answer == "" || answer == "y" || answer == "yes"
}
