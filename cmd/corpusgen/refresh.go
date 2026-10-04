package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

// cssParsingFiles lists the raw files vendored from the css-parsing-tests
// corpus.
var cssParsingFiles = []string{
	"An+B.json",
	"blocks_contents.json",
	"color_function_4.json",
	"color_functions_5.json",
	"color_hexadecimal_3.json",
	"color_hexadecimal_4.json",
	"color_hsl_3.json",
	"color_hsl_4.json",
	"color_hwb_4.json",
	"color_keywords_3.json",
	"color_keywords_4.json",
	"color_lab_4.json",
	"color_lch_4.json",
	"color_oklab_4.json",
	"color_oklch_4.json",
	"component_value_list.json",
	"declaration_list.json",
	"one_component_value.json",
	"one_declaration.json",
	"one_rule.json",
	"rule_list.json",
	"stylesheet.json",
	"stylesheet_bytes.json",
	"README.rst",
	"LICENSE",
}

const cssParsingBase = "https://raw.githubusercontent.com/CourtBouillon/css-parsing-tests/main"

func refreshCorpora(upstream, corpus string) error {
	if corpus == "all" || corpus == "css-parsing-tests" {
		if err := refreshFiles(filepath.Join(upstream, "css-parsing-tests"), cssParsingBase, cssParsingFiles); err != nil {
			return err
		}
	}
	return nil
}

func refreshFiles(directory, base string, names []string) error {
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return err
	}
	for _, name := range names {
		if err := downloadFile(base+"/"+name, filepath.Join(directory, name)); err != nil {
			return fmt.Errorf("%s: %w", name, err)
		}
	}
	return nil
}

func downloadFile(url, target string) error {
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: %s", url, response.Status)
	}
	temporary := target + ".tmp"
	output, err := os.Create(temporary)
	if err != nil {
		return err
	}
	if _, err := io.Copy(output, response.Body); err != nil {
		output.Close()
		return err
	}
	if err := output.Close(); err != nil {
		return err
	}
	return os.Rename(temporary, target)
}
