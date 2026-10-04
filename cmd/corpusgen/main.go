// Command corpusgen turns the raw upstream test sources vendored under
// testdata/upstream into the normalized JSON corpora consumed by `go test`.
//
// It is offline by default. Pass -refresh to re-download the raw sources.
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	refresh := flag.Bool("refresh", false, "re-download raw upstream sources")
	corpus := flag.String("corpus", "all", "corpus to process: all or a corpus name")
	upstream := flag.String("upstream", "testdata/upstream", "directory of raw upstream sources")
	root := flag.String("root", ".", "repository root")
	flag.Parse()

	if *refresh {
		if err := refreshCorpora(*upstream, *corpus); err != nil {
			fmt.Fprintln(os.Stderr, "corpusgen: refresh:", err)
			os.Exit(1)
		}
	}
	if err := generateCorpora(*upstream, *root, *corpus); err != nil {
		fmt.Fprintln(os.Stderr, "corpusgen:", err)
		os.Exit(1)
	}
}
