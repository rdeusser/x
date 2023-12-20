package main

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
)

func ensureModPath() error {
	_, err := os.ReadFile("go.mod")
	if err != nil {
		return err
	}
	return nil
}

func findTestData() ([]string, error) {
	var paths []string

	if err := ensureModPath(); err != nil {
		return paths, err
	}

	err := fs.WalkDir(os.DirFS("."), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			return nil
		}

		if d.Name() == "testdata" {
			paths = append(paths, filepath.Dir(path))
		}

		return nil
	})
	if err != nil {
		return paths, err
	}

	return paths, nil
}

func updateTestData() error {
	cmd := exec.Command("go", "test", "-v", "-timeout", "2m", "-update", ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func main() {
	var hadError bool

	if err := ensureModPath(); err != nil {
		log.Fatal(err)
	}

	paths, err := findTestData()
	if err != nil {
		log.Fatal(err)
	}

	for _, path := range paths {
		if err := os.Chdir(path); err != nil {
			log.Fatal(err)
		}

		log.Printf("updating testdata for %s", path)

		if err := updateTestData(); err != nil {
			hadError = true
			fmt.Println(err)
		}
	}

	if hadError {
		log.Println("some error(s) occurred in some of the tests")
	} else {
		log.Println("successfully updated testdata!")
	}
}
