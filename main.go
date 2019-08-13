// The MineSync client
package main

import (
	"encoding/gob"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/pierrre/archivefile/zip"
)

const macosDir string = "/Library/Application Support/minecraft/saves"
const windowsDir string = "/%appdata%/.minecraft/saves"
const linuxDir string = "/.minecraft/saves"
const windows10Dir string = "/%appdata%/Local/Packages/Microsoft.MinecraftUWP_8wekyb3d8bbwe/LocalState/games/com.mojang"

const syncServer string = "127.0.0.1:9999"

type savegame struct {
	os.FileInfo
	path string
}

type syncObject struct {
	Name string
	Data []byte
}

func main() {
	path, err := getSavegamePath(runtime.GOOS)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	if !pathExists(path) {
		fmt.Println("Path does not exist.")
		return
	}

	savegames, err := getListOfSaves(path)
	if err != nil {
		fmt.Println(err.Error())
	}

	syncFilesToServer(savegames)
}

// Returns the savegame path for the system.
func getSavegamePath(currentOs string) (string, error) {
	if currentOs == "windows" {
		path, err := filepath.Abs(windowsDir)
		if err != nil {
			return filepath.Abs(windows10Dir)
		}

		return path, nil
	} else if currentOs == "darwin" {
		dir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Abs(dir + macosDir)
	} else if currentOs == "linux" {
		dir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Abs(dir + linuxDir)
	} else {
		return "", errors.New("could not define OS")
	}
}

// Checks if the path exists.
func pathExists(path string) bool {
	_, e := os.Stat(path)
	return e == nil
}

// Returns a list of savegame items based on the save files found in path.
func getListOfSaves(path string) ([]savegame, error) {
	dir, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer dir.Close()

	fileInfos, err := dir.Readdir(-1)
	if err != nil {
		return nil, err
	}

	savegames := make([]savegame, 0)

	for _, fileInfo := range fileInfos {
		if fileInfo.IsDir() {
			savegames = append(savegames, savegame{fileInfo, path})
		}
	}

	return savegames, nil
}

// Synchronizes the savegames to the server
func syncFilesToServer(files []savegame) {
	for _, f := range files {
		c, err := net.Dial("tcp", syncServer)
		if err != nil {
			fmt.Println(err)
		}

		fmt.Println("Syncing", f.Name(), "to the server...")

		zipFilename := "minesync_" + strings.ReplaceAll(f.Name(), " ", "_") + ".zip"
		zipFile := os.TempDir() + zipFilename
		if err := zipSavegame(f, zipFile); err != nil {
			fmt.Println(err)
			return
		}

		data, err := ioutil.ReadFile(zipFile)
		if err != nil {
			fmt.Println(err)
			return
		}

		gob.NewEncoder(c).Encode(syncObject{zipFilename, data})

		if err := os.Remove(zipFile); err != nil {
			fmt.Println(err)
			return
		}
	}
}

// Creates the zip file for the given savegame
func zipSavegame(f savegame, zipName string) error {
	progress := func(archivePath string) {}

	err := zip.ArchiveFile(f.path+"/"+f.Name(), zipName, progress)
	if err != nil {
		fmt.Println(err)
	}

	return nil
}
