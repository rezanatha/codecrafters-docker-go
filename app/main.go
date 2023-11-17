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

func copyFile(destination, source string) error {
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

	// //cd to mkdir
	// if err := os.Chdir(tempDir); err != nil {
	// 	log.Fatal(err)
	// }

	// mydir, func_err := os.Getwd()
	// if func_err != nil {
	// 	fmt.Println(func_err)
	// }
	// fmt.Println("mydir", mydir)

	chrootCommand := filepath.Join(tempDir, command)
	fmt.Println("chroot dir", chrootCommand)

	///==== copy binary (what to copy?)
	copyCommand, err := exec.LookPath(command)
	if err != nil {
		panic(err)
	}

	if err := copyFile(chrootCommand, copyCommand); err != nil {
		panic(err)
	}

	//chroot
	if err := syscall.Chroot(tempDir); err != nil {
		panic(err)
	}

	//run command
	chrootCommand = filepath.Join("/", filepath.Base(command))

	cmd := exec.Command(command, args...)
	err = cmd.Run()

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	go io.Copy(os.Stdout, stdout)
	go io.Copy(os.Stderr, stderr)

	if err := cmd.Run(); err != nil {
		fmt.Printf("Run() err: %v \n", err)
		os.Exit(cmd.ProcessState.ExitCode())
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
