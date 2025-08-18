package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

type SimpleTerminal struct {
	prompt string
}

func NewSimpleTerminal() *SimpleTerminal {
	return &SimpleTerminal{prompt: "simple> "}
}

func (st *SimpleTerminal) RunCommand(command string) {
	command = strings.TrimSpace(command)
	if command == "" {
		return
	}

	// Handle built-in commands
	switch strings.ToLower(command) {
	case "exit", "quit":
		fmt.Println("Goodbye!")
		os.Exit(0)
	case "clear", "cls":
		st.ClearScreen()
		return
	case "help":
		st.ShowHelp()
		return
	}

	// Execute system command
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/c", command)
	} else {
		cmd = exec.Command("sh", "-c", command)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

func (st *SimpleTerminal) ClearScreen() {
	if runtime.GOOS == "windows" {
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	} else {
		fmt.Print("\033[H\033[2J")
	}
}

func (st *SimpleTerminal) ShowHelp() {
	fmt.Println("Simple Terminal Commands:")
	fmt.Println("  help  - Show this help")
	fmt.Println("  clear - Clear screen")
	fmt.Println("  exit  - Exit terminal")
	fmt.Println()
	fmt.Println("Any other command will be executed as system command.")
}

func (st *SimpleTerminal) Start() {
	fmt.Println("Simple Interactive Terminal")
	fmt.Println("Type 'help' for commands, 'exit' to quit")
	fmt.Println()

	scanner := bufio.NewScanner(os.Stdin)
	
	for {
		fmt.Print(st.prompt)
		
		if !scanner.Scan() {
			break
		}
		
		command := scanner.Text()
		st.RunCommand(command)
	}
}

func main() {
	terminal := NewSimpleTerminal()
	terminal.Start()
}
