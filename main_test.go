package main

import (
	"fmt"
	"os"
	"runtime"
	"testing"
)

type pathtestpair struct {
	system string
	path   string
}

var pathtests = []pathtestpair{}

func setup() {
	homedir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("could not set up user home dir")
		return
	}

	pathtests = append(pathtests, pathtestpair{"windows", "/%appdata%/.minecraft/saves"})
	pathtests = append(pathtests, pathtestpair{"darwin", homedir + "/Library/Application Support/minecraft/saves"})
	pathtests = append(pathtests, pathtestpair{"linux", homedir + "/.minecraft/saves"})
}

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	os.Exit(code)
}

func TestGetSavegamePath(t *testing.T) {
	for _, c := range pathtests {
		path, err := getSavegamePath(c.system)
		if err != nil {
			t.Error(err.Error())
		}

		if path != c.path {
			t.Error(
				"For", c.system,
				"expected", c.path,
				"got", path,
			)
		}
	}
}

func TestPathExists(t *testing.T) {
	// todo: generate a unique random path
	if pathExists("/some/random/path") {
		t.Error("Non-existent path exists")
	}

	if !pathExists("/") {
		t.Error("Existing path does not exist")
	}
}

func TestGetListOfSaves(t *testing.T) {
	p, e := getSavegamePath(runtime.GOOS)
	if e != nil {
		t.Error("Could not get savegame path")
	}

	_, err := getListOfSaves(p)
	if err != nil {
		t.Error("Could not get list of saves")
	}
}
