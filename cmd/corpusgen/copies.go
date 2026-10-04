package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// urlResourceFiles are copied into urls/testdata for the URL conformance suite.
var urlResourceFiles = []string{
	"IdnaTestV2.json",
	"setters_tests.json",
	"toascii.json",
	"percent-encoding.json",
}

// html5libTrees are copied into parsers/html/testdata for the HTML, tokenizer,
// and encoding conformance suites.
var html5libTrees = []string{"encoding", "tokenizer", "tree-construction"}

func generateURL(upstream, root string) error {
	source := filepath.Join(upstream, "wpt", "url", "resources")
	destination := filepath.Join(root, "urls", "testdata")
	for _, name := range urlResourceFiles {
		if err := copyFile(filepath.Join(source, name), filepath.Join(destination, name)); err != nil {
			return err
		}
	}
	return nil
}

func generateHTML5lib(upstream, root string) error {
	source := filepath.Join(upstream, "html5lib-tests")
	destination := filepath.Join(root, "parsers", "html", "testdata", "html5lib-tests")
	for _, tree := range html5libTrees {
		if err := copyTree(filepath.Join(source, tree), filepath.Join(destination, tree)); err != nil {
			return err
		}
	}
	return nil
}

func copyTree(source, destination string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		return copyFile(path, target)
	})
}

func copyFile(source, destination string) error {
	input, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("copy %s: %w", source, err)
	}
	defer input.Close()
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	output, err := os.Create(destination)
	if err != nil {
		return err
	}
	if _, err := io.Copy(output, input); err != nil {
		output.Close()
		return err
	}
	return output.Close()
}
