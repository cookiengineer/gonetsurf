package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
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

const wptContentsURL = "https://api.github.com/repos/web-platform-tests/wpt/contents"
const wptTreeBase = "https://api.github.com/repos/web-platform-tests/wpt/git/trees/"
const wptCommitURL = "https://api.github.com/repos/web-platform-tests/wpt/commits/master"
const wptRawBase = "https://raw.githubusercontent.com/web-platform-tests/wpt/"

var httpClient = &http.Client{Timeout: 60 * time.Second}

func refreshCorpora(upstream, corpus string) error {
	if corpus == "all" || corpus == "css-parsing-tests" {
		if err := refreshFiles(filepath.Join(upstream, "css-parsing-tests"), cssParsingBase, cssParsingFiles); err != nil {
			return err
		}
	}
	if corpus == "all" || corpus == "wpt" {
		if err := refreshWPT(filepath.Join(upstream, "wpt")); err != nil {
			return err
		}
	}
	return nil
}

// wptTreeResponse is the subset of the GitHub tree API used to discover the
// parsing suites that are vendored for offline conformance runs.
type wptContentsEntry struct {
	Name string `json:"name"`
	Type string `json:"type"`
	SHA  string `json:"sha"`
}

type wptTreeResponse struct {
	SHA       string `json:"sha"`
	Truncated bool   `json:"truncated"`
	Tree      []struct {
		Path string `json:"path"`
		Type string `json:"type"`
	} `json:"tree"`
}

type wptCommitResponse struct {
	SHA string `json:"sha"`
}

func refreshWPT(directory string) error {
	cssTree, err := wptCSSTree()
	if err != nil {
		return err
	}
	commit, err := wptCommit()
	if err != nil {
		return err
	}
	treeURL := wptTreeBase + cssTree + "?recursive=1"
	response, err := httpClient.Get(treeURL)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: %s", treeURL, response.Status)
	}
	var tree wptTreeResponse
	if err := json.NewDecoder(response.Body).Decode(&tree); err != nil {
		return fmt.Errorf("decode WPT tree: %w", err)
	}
	if tree.Truncated {
		return fmt.Errorf("WPT css tree is truncated")
	}
	paths := make([]string, 0, 1400)
	for _, entry := range tree.Tree {
		if entry.Type != "blob" {
			continue
		}
		if !strings.Contains(entry.Path, "/parsing/") || !strings.HasSuffix(entry.Path, ".html") {
			continue
		}
		if !strings.HasPrefix(entry.Path, "selectors/parsing/") &&
			!strings.HasSuffix(entry.Path, "-valid.html") &&
			!strings.HasSuffix(entry.Path, "-invalid.html") {
			continue
		}
		paths = append(paths, "css/"+entry.Path)
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return fmt.Errorf("WPT tree contains no css/*/parsing HTML files")
	}
	jobs := make(chan string)
	failures := make(chan error, len(paths))
	var wait sync.WaitGroup
	for worker := 0; worker < 8; worker++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for path := range jobs {
				target := filepath.Join(directory, filepath.FromSlash(path))
				if err := downloadFile(wptRawBase+commit.SHA+"/"+path, target); err != nil {
					failures <- err
				}
			}
		}()
	}
	for _, path := range paths {
		jobs <- path
	}
	close(jobs)
	wait.Wait()
	close(failures)
	if err := <-failures; err != nil {
		return err
	}
	return writeWPTSource(directory, commit.SHA, cssTree)
}

func wptCSSTree() (string, error) {
	response, err := httpClient.Get(wptContentsURL)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: %s", wptContentsURL, response.Status)
	}
	var entries []wptContentsEntry
	if err := json.NewDecoder(response.Body).Decode(&entries); err != nil {
		return "", fmt.Errorf("decode WPT contents: %w", err)
	}
	for _, entry := range entries {
		if entry.Name == "css" && entry.Type == "dir" {
			return entry.SHA, nil
		}
	}
	return "", fmt.Errorf("WPT repository has no css directory")
}

func wptCommit() (wptCommitResponse, error) {
	response, err := httpClient.Get(wptCommitURL)
	if err != nil {
		return wptCommitResponse{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return wptCommitResponse{}, fmt.Errorf("GET %s: %s", wptCommitURL, response.Status)
	}
	var commit wptCommitResponse
	if err := json.NewDecoder(response.Body).Decode(&commit); err != nil {
		return wptCommitResponse{}, fmt.Errorf("decode WPT commit: %w", err)
	}
	if commit.SHA == "" {
		return wptCommitResponse{}, fmt.Errorf("WPT commit response has no SHA")
	}
	return commit, nil
}

func writeWPTSource(directory, revision, cssTree string) error {
	content := fmt.Sprintf(`Web Platform Tests (WPT)
========================

Upstream: https://github.com/web-platform-tests/wpt
Commit:   %s
CSS tree: %s
License:  BSD-3-Clause (WPT)
Fetched:  %s

Vendored subsets:
  css/*/parsing/            CSS parsing tables, including nested parsing dirs
  css/selectors/parsing/    Selectors Level 4 validity tables

Parsing files are vendored for offline conformance and for property registry
generation. The normalized corpora consumed by `+"`go test`"+` are produced by
`+"`cmd/corpusgen`"+`.
`, revision, cssTree, time.Now().UTC().Format("2006-01-02"))
	return os.WriteFile(filepath.Join(directory, "SOURCE"), []byte(content), 0o644)
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
	response, err := httpClient.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("GET %s: %s", url, response.Status)
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
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
