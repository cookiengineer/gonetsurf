// Command netsurfctl provides maintenance and diagnostic subcommands for the
// gonetsurf browser: dependency policy checks, cache management, and inspection
// helpers.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"

	"gonetsurf/urls"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: netsurfctl <deps-check|urlparse> [args]")
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
