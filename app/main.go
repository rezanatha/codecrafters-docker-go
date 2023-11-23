//go:build linux
// +build linux

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
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

type TokenResponse struct {
	Token       string `json:"token"`
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	IssuedAt    string `json:"issued_at"`
}

type Manifest struct {
	SchemaVersion int    `json:"schemaVersion"`
	MediaType     string `json:"mediaType"`
	Config        struct {
		MediaType string `json:"mediaType"`
		Size      int    `json:"size"`
		Digest    string `json:"digest"`
	}
	Layers []struct {
		MediaType string `json:"mediaType"`
		Size      int    `json:"size"`
		Digest    string `json:"digest"`
	}
}

const (
	getTokenURL         = "https://auth.docker.io/token?service=registry.docker.io&scope=repository:library/%s:pull"
	getImageManifestURL = "https://registry-1.docker.io/v2/library/%s/manifests/latest"
	pullDockerLayerURL  = "https://registry-1.docker.io/v2/library/%s/blobs/%s"
)

func getAuthToken(imageName string) string {
	resp, err := http.Get(fmt.Sprintf(getTokenURL, imageName))
	if err != nil {
		log.Fatal("getAuthToken(): HTTP GET error ", err)
	}
	defer resp.Body.Close()
	var docker_token TokenResponse
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("getAuthToken(): ioutil.ReadAll error ", err)
	}

	err = json.Unmarshal(body, &docker_token)
	if err != nil {
		log.Fatal("getAuthToken(): json.Unmarshal error ", err)
	}
	return docker_token.Token
}

func getImageManifest(token, imageName string) Manifest {
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf(getImageManifestURL, imageName), nil)
	if err != nil {
		panic(err)
	}
	req.Header.Add("Authorization", "Bearer "+token)
	req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	var manifest Manifest
	err = json.Unmarshal(body, &manifest)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	return manifest
}

func pullDockerLayer(imageName string, download_path string) {
	//==== 1. Get token
	var token string = getAuthToken(imageName)

	//===== 2. Fetch image manifest
	var manifest Manifest = getImageManifest(token, imageName)

	//===== 3. Pull layer
	client := &http.Client{}
	for _, layer := range manifest.Layers {
		//fmt.Println("digest", layer.Digest)
		req, err := http.NewRequest("GET", fmt.Sprintf(pullDockerLayerURL, imageName, layer.Digest), nil)
		if err != nil {
			panic(err)
		}
		req.Header.Add("Authorization", "Bearer "+token)
		req.Header.Add("Accept", "application/vnd.docker.distribution.manifest.v2+json")
		resp, err := client.Do(req)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()
		//download
		/*
			1. Create empty file
			2. Download layer
			3. Verify checksum
			4. Extract to our root (new root that has been chroot-ed)
		*/
		layer_path := filepath.Join(download_path, "docker_layer.tar")
		file, err := os.Create(layer_path)
		if err != nil {
			panic(err)
		}
		_, err = io.Copy(file, resp.Body)
		if err != nil {
			panic(err)
		}
		defer file.Close()
		cmd := exec.Command("tar", "-xf", layer_path, "-C", download_path)
		if err := cmd.Run(); err != nil {
			fmt.Println("error doing tar")
			panic(err)
		}
		if err = os.Remove(layer_path); err != nil {
			fmt.Println("error removing tar file")
			panic(err)
		}

	}
}

func main() {
	var imageName string = os.Args[2]
	command := os.Args[3]
	args := os.Args[4:len(os.Args)]

	//==== mkdir
	tempDir, err := os.MkdirTemp("", "chroot_temp")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	//==== pull docker layer
	pullDockerLayer(imageName, tempDir)

	///==== chroot
	if err := syscall.Chroot(tempDir); err != nil {
		log.Fatal(err)
	}

	///==== create dev/null
	os.Mkdir("/dev", 0755)
	devNull, _ := os.Create("/dev/null")
	devNull.Close()

	cmd := exec.Command(command, args...)
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	go io.Copy(os.Stdout, stdout)
	go io.Copy(os.Stderr, stderr)

	//===== Isolate process
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWPID | syscall.CLONE_NEWUTS,
	}

	if err = cmd.Run(); err != nil {
		fmt.Printf("Run() err: %v \n", err)
		exitError, _ := err.(*exec.ExitError)
		os.Exit(exitError.ExitCode())
	}

	os.Exit(0)
}
