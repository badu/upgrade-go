package main

import (
	"bytes"
	"context"
	"fmt"
	"github.com/google/go-github/v29/github"
	"github.com/hashicorp/go-version"
	"github.com/manifoldco/promptui"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"sort"
	"strings"
)

const (
	GitOrganization = "golang"
	GitRepository   = "dl"
	MasterBranch    = "refs/heads/master"
	TreeType        = "tree"
)

func SelectVersion() (string, error) {
	// go to GitHub to take the versions from folder names
	client := github.NewClient(nil)
	ref, _, err := client.Git.GetRef(context.Background(), GitOrganization, GitRepository, MasterBranch)
	if err != nil {
		log.Fatalf("error getting `%s` from `github.com/%s/%s` : %v", MasterBranch, GitOrganization, GitRepository, err)
	}
	tree, _, err := client.Git.GetTree(context.Background(), GitOrganization, GitRepository, ref.GetObject().GetSHA(), false)
	if err != nil {
		log.Fatalf("error reading tree from `github.com/%s/%s` : %v", GitOrganization, GitRepository, err)
	}
	// create a list of versions by looking at the folder names
	var versions []*version.Version
	for _, entry := range tree.Entries {
		if entry.GetType() == TreeType {
			folderName := entry.GetPath()
			if strings.HasPrefix(folderName, "go") {
				ver, err := version.NewVersion(strings.Replace(folderName, "go", "", -1))
				if err == nil {
					versions = append(versions, ver)
				}
				// otherwise, it's probably `gotip`
			}
		}
	}
	// sort versions
	sort.Sort(sort.Reverse(version.Collection(versions)))
	// create the raw list to be used in prompt
	var versionsRaw []string
	for _, ver := range versions {
		versionsRaw = append(versionsRaw, ver.Original())
	}

	prompt := promptui.Select{Label: "Select Version", Items: versionsRaw}
	_, result, err := prompt.Run()
	if err != nil {
		log.Fatalf("choosing Go version failed : %v", err)
	}
	return result, nil
}

func SelectArch() (string, error) {
	prompt := promptui.Select{
		Label: "Select Architecture",
		Items: []string{"amd64", "386"},
	}
	_, result, err := prompt.Run()
	if err != nil {
		log.Fatalf("choosing architecture failed : %v", err)
	}
	return result, nil
}

func SelectOS() (string, error) {
	prompt := promptui.Select{
		Label: "Select OS",
		Items: []string{"linux", "darwin", "freebsd"}, // no Windows :)
	}
	_, result, err := prompt.Run()
	if err != nil {
		log.Fatalf("choosing operating system failed : %v", err)
	}
	return result, nil
}

func ExistingGo() bool {
	// running command : which go
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd := exec.Command("which", "go")
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	if err := cmd.Run(); err != nil {
		log.Fatalf("error running command `which go` : %s [%v]", errOut.String(), err)
	}
	what, err := out.ReadString('\n')
	if err != nil {
		log.Fatalf("error reading result of command `which go` : %v", err)
	}
	log.Printf("Go is installed in : %s", what)
	return true
}

func GoVersion() {
	// running command : go version
	var out bytes.Buffer
	var errOut bytes.Buffer
	verCmd := exec.Command("go", "version")
	verCmd.Stdout = &out
	verCmd.Stderr = &errOut
	if err := verCmd.Run(); err != nil {
		log.Fatalf("error running command `go version` : %s [%v]", errOut.String(), err)
	}
	what, err := out.ReadString('\n')
	if err != nil {
		log.Fatalf("error reading result of command `go version` : %v", err)
	}
	log.Printf("Go version : %q", what)

}

func DownloadArchive(fromURL, toFilePath string) {
	// running download e.g. https://dl.google.com/go/go1.13.6.linux-amd64.tar.gz
	resp, err := http.Get(fromURL)
	if err != nil {
		log.Fatalf("download failed : %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		log.Fatalf("Something went wrong : http status is %d", resp.StatusCode)
	}
	// Create the file
	out, err := os.Create(toFilePath)
	if err != nil {
		log.Fatalf("creating file failed : %v", err)
	}
	defer out.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	log.Printf("Download complete.")
}

func RemoveJunk() {
	// running command : sudo rm -rf /usr/local/go
	cmd := exec.Command("sudo", "rm", "-rf", "/usr/local/go-old")
	var out bytes.Buffer
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		log.Printf("removing previous Go version failed : %s [%v]", out.String(), err)
	}
}

func RenameExisting() {
	// running command : sudo rm -rf /usr/local/go
	cmd := exec.Command("sudo", "mv", "/usr/local/go", "/usr/local/go-old")
	var out bytes.Buffer
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		log.Printf("removing previous Go version failed : %s [%v]", out.String(), err)
	}
}

func UndoRenameExisting() {
	// running command : sudo rm -rf /usr/local/go
	cmd := exec.Command("sudo", "mv", "/usr/local/go-old", "/usr/local/go")
	var out bytes.Buffer
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		log.Printf("restoring previous Go version failed : %s [%v]", out.String(), err)
	}
}

func ExtractAndInstall(archivePath string) error {
	// running command : sudo tar -C /usr/local -xzf /home/<username>/Downloads/go1.8.1.linux-amd64.tar.gz
	cmd := exec.Command("sudo", "tar", "-C", "/usr/local", "-xzf", archivePath)
	var out bytes.Buffer
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		log.Printf("untar failed : %s [%v]", out.String(), err)
		return err
	}
	return nil
}

func main() {
	var (
		err            error
		arch, ost, ver string
	)
	log.Println("Current Go location")
	if !ExistingGo() {
		log.Fatalf("No current Go found.")
	}
	log.Println("Existing version:")
	GoVersion()
	log.Println("Architecture:")
	if arch, err = SelectArch(); err != nil {
		os.Exit(1)
	}
	log.Println("Operating system:")
	if ost, err = SelectOS(); err != nil {
		os.Exit(1)
	}
	log.Println("Version:")
	if ver, err = SelectVersion(); err != nil {
		os.Exit(1)
	}

	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	tarURL := fmt.Sprintf("https://dl.google.com/go/go%s.%s-%s.tar.gz", ver, ost, arch)
	downloadFile := fmt.Sprintf(usr.HomeDir+"/Downloads/go%s.%s-%s.tar.gz", ver, ost, arch)
	if _, err := os.Stat(downloadFile); os.IsNotExist(err) {
		log.Printf("Downloading %q into %s", tarURL, downloadFile)
		DownloadArchive(tarURL, downloadFile)
	}
	log.Println("Renaming old Go folder (might need undo)")
	RenameExisting()
	log.Println("Extracting downloaded archive into /usr/local folder")
	if err := ExtractAndInstall(downloadFile); err != nil {
		log.Println("We've failed : undo renaming folder to the old version")
		UndoRenameExisting()
		os.Exit(1)
	}
	log.Println("Cleaning up old Go version (folder /usr/local/go-old gets deleted)")
	RemoveJunk()
	log.Println("Currently installed Go version:")
	GoVersion()

}
