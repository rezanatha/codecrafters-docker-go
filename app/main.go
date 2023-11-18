package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// Usage: your_docker.sh run <image> <command> <arg1> <arg2> ...
func copyFile(destination string, source string) error {
	/*
		Use defer and Close() whenever we are working on files
		(i.e., using APIs such as os.Open or os.Create) as it kills the process associated with it.
		Otherwise we would encounter "text file busy" error.
	*/

	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	sourceStat, err := sourceFile.Stat()
	if err != nil {
		return err
	}
	sourcePermission := sourceStat.Mode()

	destinationFile, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return err
	}

	err = destinationFile.Chmod(sourcePermission)
	if err != nil {
		return err
	}
	return nil
}
func main() {
	command := os.Args[3]
	args := os.Args[4:len(os.Args)]

	//==== mkdir
	tempDir, err := os.MkdirTemp("", "chroot_temp")
	if err != nil {
		log.Fatal(err)
	}

	defer os.RemoveAll(tempDir)

	chrootCommand := filepath.Join(tempDir, filepath.Base(command))

	///==== copy binary (what to copy?)
	command, err = exec.LookPath(command)
	if err != nil {
		log.Fatal(err)
	}

	if err := copyFile(chrootCommand, command); err != nil {
		log.Fatal(err)
	}

	///==== chroot
	if err := syscall.Chroot(tempDir); err != nil {
		log.Fatal(err)
	}

	///==== create dev/null
	os.Mkdir("/dev", 0755)
	devNull, _ := os.Create("/dev/null")
	devNull.Close()

	///==== run command
	chrootCommand = filepath.Join("/", filepath.Base(command))

	cmd := exec.Command(chrootCommand, args...)
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	go io.Copy(os.Stdout, stdout)
	go io.Copy(os.Stderr, stderr)
	err = cmd.Run()
	if err != nil {
		fmt.Printf("Run() err: %v \n", err)
		exitError, _ := err.(*exec.ExitError)
		os.Exit(exitError.ExitCode())
	}

	os.Exit(0)
	/* STEPS
	We want to execute a binary after doing chroot, so that the binary will think the root is the directory we create, instead of the real root directory
	1. mkdir temporary folder as our root. call this "jail"
	2. copy binary from anywhere to jail
	3. chroot to jail
	4. execute binary

	*/
}
