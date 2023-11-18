package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

// Usage: your_docker.sh run <image> <command> <arg1> <arg2> ...

func copyFile(src, dst string) error {
	// Open the source file for reading
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()
	// Get the source file's permissions
	sourceInfo, err := sourceFile.Stat()
	if err != nil {
		return err
	}
	sourcePerm := sourceInfo.Mode()
	// Create the destination file for writing
	destinationFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	//defer destinationFile.Close()
	// Copy the contents of the source file to the destination file
	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return err
	}
	// Set the destination file's permissions to match the source file
	err = destinationFile.Chmod(sourcePerm)
	if err != nil {
		return err
	}
	return nil
}

// Usage: your_docker.sh run <image> <command> <arg1> <arg2> ...
func main() {
	command := os.Args[3]
	args := os.Args[4:len(os.Args)]
	cmd := exec.Command(command, args...)
	// Create root of executable command
	tempDir, err := os.MkdirTemp("", "mychroot")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tempDir)
	commandInChroot := filepath.Join(tempDir, filepath.Base(command))
	// Copy the binary command to the new root.
	command, err = exec.LookPath(command)
	if err != nil {
		panic(err)
	}
	if err := copyFile(command, commandInChroot); err != nil {
		panic(err)
	}
	// Enter the chroot.
	if err := syscall.Chroot(tempDir); err != nil {
		panic(err)
	}
	commandInChroot = filepath.Join("/", filepath.Base(command))

	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err := cmd.Run(); err != nil {
		fmt.Printf("Err: %v", err)
		os.Exit(cmd.ProcessState.ExitCode())
	}
}
