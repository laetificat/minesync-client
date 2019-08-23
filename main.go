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
	"time"

	"github.com/pierrre/archivefile/zip"
)

const macosDir string = "/Library/Application Support/minecraft/saves"
const windowsDir string = "/%appdata%/.minecraft/saves"
const linuxDir string = "/.minecraft/saves"
const windows10Dir string = "/%appdata%/Local/Packages/Microsoft.MinecraftUWP_8wekyb3d8bbwe/LocalState/games/com.mojang"

const syncServer string = "127.0.0.1:9999"
const syncServerList string = "127.0.0.1:9998"
const downloadServer string = "127.0.0.1:9997"

type savegame struct {
	os.FileInfo
	path string
}

type syncObject struct {
	Name string
	Data []byte
}

type saveList struct {
	Saves []save
}

type save struct {
	Name             string
	LastModifiedDate time.Time
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

	saves, err := getListOfSavesFromServer()
	if err != nil {
		fmt.Println(err)
		return
	}

	savesToUpload := getSavesToUpload(savegames, saves)
	savesToDownload := getSavesToDownload(savegames, saves)

	//todo: remove after debugging
	fmt.Println("list of saves from server")
	fmt.Printf("%s", saves)
	fmt.Println("")
	fmt.Println("")
	fmt.Println("saves to upload")
	fmt.Printf("%s", savesToUpload)
	fmt.Println("")
	fmt.Println("")
	fmt.Println("saves to download")
	fmt.Printf("%s", savesToDownload)
	fmt.Println("")
	fmt.Println("")

	syncFilesToServer(savesToUpload)
	syncFilesFromServer(savesToDownload)
}

func getSavesToUpload(s []savegame, rs []save) []savegame {
	l := []savegame{}

OUTER:
	for _, v := range s {
		for _, v1 := range rs {
			n := "minesync_" + strings.ReplaceAll(v.FileInfo.Name(), " ", "_") + ".zip"
			if n == v1.Name {
				if v.FileInfo.ModTime().After(v1.LastModifiedDate) {
					l = append(l, v)
				}
				continue OUTER
			}
		}

		l = append(l, v)
	}

	return l
}

func getSavesToDownload(s []savegame, rs []save) []save {
	l := []save{}

OUTER:
	for _, v := range rs {
		for _, v1 := range s {
			n := "minesync_" + strings.ReplaceAll(v1.FileInfo.Name(), " ", "_") + ".zip"
			if n == v.Name {
				if v.LastModifiedDate.After(v1.FileInfo.ModTime()) {
					l = append(l, v)
				}
				continue OUTER
			}
		}

		l = append(l, v)
	}

	return l
}

// Gets the list of saves from the server.
func getListOfSavesFromServer() ([]save, error) {
	sl := saveList{}

	c, err := net.Dial("tcp", syncServerList)
	err = gob.NewDecoder(c).Decode(&sl)

	return sl.Saves, err
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

// Downloads the savegames from the server
func syncFilesFromServer(files []save) {
	fmt.Println("Downloaded saves")
	for _, s := range files {
		c, err := net.Dial("tcp", downloadServer)
		if err != nil {
			fmt.Println(err)
			return
		}

		gob.NewEncoder(c).Encode(s)

		decoded := syncObject{}
		gob.NewDecoder(c).Decode(&decoded)

		// todo: write data to minecraft save folder and unzip
		fmt.Println(decoded.Name)
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
