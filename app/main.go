package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

// Usage: your_docker.sh run <image> <command> <arg1> <arg2> ...
func main() {
	command := os.Args[3]
	args := os.Args[4:len(os.Args)]
	cmd := exec.Command(command, args...)
	// err := cmd.Run()

	//cmd := exec.Command("ldwds")
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	go io.Copy(os.Stdout, stdout)
	go io.Copy(os.Stderr, stderr)

	err := cmd.Run()
	if err != nil {
		fmt.Printf("err: %v", err)
		exitError, _ := err.(*exec.ExitError)
		os.Exit(exitError.ExitCode())
	}

	os.Exit(0)
}
