package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/vazrupe/go-acb/acb"
)

func main() {
	defaultDir := ""
	saveDir := flag.String("save", defaultDir, "extract dir")
	force := flag.Bool("f", false, "if an existing destination file cannot be opened, remove it and try again")

	flag.Parse()
	files := flag.Args()

	var saveRoot string
	for _, filename := range files {
		f, err := acb.LoadCriAcbFile(filename)
		if err != nil {
			fmt.Printf("Error: %s Open Failed (%s)\n", filename, err)
		}

		name := filepath.Base(filename)
		if (*saveDir) == defaultDir {
			ext := filepath.Ext(filename)
			saveRoot = filename[:len(filename)-len(ext)]
		} else {
			ext := filepath.Ext(filename)
			dirName := name[:len(name)-len(ext)]
			saveRoot = filepath.Join(*saveDir, dirName)
		}

		if _, err := os.Stat(saveRoot); err == nil {
			if !*force {
				fmt.Printf("Exists: directory `%s`. skip\n", saveRoot)
				continue
			}
			RemoveDir(saveRoot)
		}
		i := SaveAcb(saveRoot, f)
		fmt.Printf("Extract: %s -> %s (%d files)\n", name, saveRoot, i)
	}
}

// RemoveDir is remove all files on dir
func RemoveDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		err = os.RemoveAll(filepath.Join(dir, name))
		if err != nil {
			return err
		}
	}
	return nil
}

// SaveAcb is extract AcbFile on target dir
func SaveAcb(savedir string, a *acb.CriAcbFile) int {
	i := 0
	for _, cue := range a.Cue {
		file, ok := a.InternalAwb.Files[uint16(cue.CueID)]
		if !ok {
			continue
		}

		savename := fmt.Sprintf("%s%s", cue.CueName, cue.GetFileExtension())
		savePath := filepath.Join(savedir, savename)
		os.MkdirAll(savedir, os.ModeDir)

		f, err := os.Create(savePath)
		if err == nil {
			i++
		}
		defer f.Close()
		f.Write(file.Data)
	}
	return i
}
