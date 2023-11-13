package main

import (
	"io"
	"os"
	"os/exec"
	"syscall"
)

// Usage: your_docker.sh run <image> <command> <arg1> <arg2> ...
func main() {
	// You can use print statements as follows for debugging, they'll be visible when running tests.
	//fmt.Println("Logs from your program will appear here!")

	command := os.Args[3]
	args := os.Args[4:len(os.Args)]

	cmd := exec.Command(command, args...)

	// output, err := cmd.Output()
	// if err != nil {
	// 	fmt.Printf("Err: %v", err)
	// 	os.Exit(1)
	// }
	// fmt.Println(string(output))

	stderrPipe, _ := cmd.StderrPipe()
	stdoutPipe, _ := cmd.StdoutPipe()

	if err := cmd.Run(); err != nil {
		if exiterror, ok := err.(*exec.ExitError); ok {
			waitstatus := exiterror.Sys().(syscall.WaitStatus)
			os.Exit(waitstatus.ExitStatus())
		}
	}

	go io.Copy(os.Stdout, stdoutPipe)
	go io.Copy(os.Stderr, stderrPipe)

}
