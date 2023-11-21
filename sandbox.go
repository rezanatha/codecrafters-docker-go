//go:build linux
// +build linux

package main

import (
	"fmt"
	"os/exec"
	"syscall"
)

func main() {
	fmt.Println("hello world")
	cmd := exec.Command("/bin/sh")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWUTS | syscall.CLONE_NEWPID,
	}
	fmt.Println(syscall.CLONE_NEWPID)
	if err := cmd.Run(); err != nil {
		panic(err)
	}
}
