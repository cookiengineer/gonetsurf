// Command netsurfctl provides maintenance and diagnostic subcommands for the
// gonetsurf browser: dependency policy checks, cache management, and inspection
// helpers.
package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"

	"gonetsurf/protocols/http1"
	"gonetsurf/urls"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: netsurfctl <deps-check|urlparse|fetch> [args]")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "deps-check":
		if err := depsCheck(); err != nil {
			fmt.Fprintln(os.Stderr, "deps-check failed:", err)
			os.Exit(1)
		}
		fmt.Println("deps-check: ok")
	case "urlparse":
		if err := urlParse(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "urlparse failed:", err)
			os.Exit(1)
		}
	case "fetch":
		if err := fetch(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "fetch failed:", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n", os.Args[1])
		os.Exit(2)
	}
}

// allowedModulePrefixes lists the module prefixes permitted at runtime. Only
// the standard library, the project's own module, and golang.org/x packages
// may appear in the dependency graph.
var allowedModulePrefixes = []string{
	"golang.org/x/",
}

func depsCheck() error {
	command := exec.Command("go", "list", "-deps",
		"-f", "{{with .Module}}{{.Path}}{{end}}", "./...")
	output, err := command.Output()
	if err != nil {
		return fmt.Errorf("go list: %w", err)
	}
	modules := make(map[string]struct{})
	for _, line := range strings.Split(string(output), "\n") {
		module := strings.TrimSpace(line)
		if module == "" {
			continue
		}
		modules[module] = struct{}{}
	}
	var violations []string
	for module := range modules {
		if isAllowedModule(module) {
			continue
		}
		violations = append(violations, module)
	}
	if len(violations) > 0 {
		sort.Strings(violations)
		return fmt.Errorf("disallowed modules: %s", strings.Join(violations, ", "))
	}
	return nil
}

func isAllowedModule(module string) bool {
	if module == "gonetsurf" || strings.HasPrefix(module, "gonetsurf/") {
		return true
	}
	for _, prefix := range allowedModulePrefixes {
		if strings.HasPrefix(module, prefix) {
			return true
		}
	}
	return false
}

func fetch(arguments []string) error {
	if len(arguments) != 1 {
		return fmt.Errorf("usage: netsurfctl fetch <url>")
	}
	status, body, err := fetchBody(arguments[0])
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "status: %d\n", status)
	_, err = os.Stdout.Write(body)
	return err
}

// fetchBody retrieves a URL and returns its status code and body.
func fetchBody(rawURL string) (int, []byte, error) {
	target, err := urls.Parse(rawURL)
	if err != nil {
		return 0, nil, err
	}
	switch target.Scheme {
	case "http", "https":
	default:
		return 0, nil, fmt.Errorf("unsupported scheme %q", target.Scheme)
	}
	client := &http1.Client{Transport: &http1.Transport{}}
	response, err := client.Do(context.Background(), &http1.Request{Method: "GET", URL: target, Header: http1.Header{}})
	if err != nil {
		return 0, nil, err
	}
	defer response.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 8<<20))
	if err != nil {
		return 0, nil, err
	}
	return response.StatusCode, body, nil
}

func urlParse(arguments []string) error {
	if len(arguments) != 1 {
		return fmt.Errorf("usage: netsurfctl urlparse <url>")
	}
	parsed, err := urls.Parse(arguments[0])
	if err != nil {
		return err
	}
	fmt.Printf("href:     %s\n", parsed.String())
	fmt.Printf("scheme:   %s\n", parsed.Scheme)
	fmt.Printf("origin:   %s\n", parsed.OriginString())
	fmt.Printf("site:     %s\n", parsed.Site())
	fmt.Printf("host:     %s\n", parsed.HostWithPort())
	fmt.Printf("path:     %s\n", parsed.Pathname())
	fmt.Printf("query:    %s\n", parsed.Search())
	fmt.Printf("fragment: %s\n", parsed.Hash())
	return nil
}
