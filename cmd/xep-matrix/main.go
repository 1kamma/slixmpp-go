// Command xep-matrix renders the compiled XEP implementation catalog as
// Markdown. It is used to keep docs/xep-support.md synchronized with code.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/1kamma/slixmpp-go/xep"
)

func main() {
	fmt.Fprintln(os.Stdout, "# XEP support matrix")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "This file is generated from `xep.Catalog` by `go run ./cmd/xep-matrix`.")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "Coverage meanings:")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "- **operational** — typed implementation plus client behavior and protocol tests within the documented scope.")
	fmt.Fprintln(os.Stdout, "- **client** — typed client operations and/or incoming event handling; some optional portions of the XEP may remain.")
	fmt.Fprintln(os.Stdout, "- **payload** — typed XML builders/parsers; application code supplies the surrounding workflow.")
	fmt.Fprintln(os.Stdout, "- **catalog** — the Slixmpp plugin name exists for migration/discovery, but loading it advertises no wire support.")
	fmt.Fprintln(os.Stdout)
	fmt.Fprintln(os.Stdout, "| XEP | Slixmpp plugin | Title | Coverage | Namespaces | Notes |")
	fmt.Fprintln(os.Stdout, "|---|---|---|---|---|---|")
	for _, descriptor := range xep.Catalog {
		namespaces := strings.Join(descriptor.Namespaces, "<br>")
		if namespaces == "" {
			namespaces = "—"
		}
		notes := descriptor.Notes
		if notes == "" {
			notes = "Metadata compatibility only; use `xmpp.Node` for custom XML."
		}
		fmt.Fprintf(os.Stdout, "| %s | `%s` | %s | **%s** | %s | %s |\n",
			descriptor.XEP(), descriptor.Name(), escape(descriptor.Title), descriptor.Coverage, escape(namespaces), escape(notes))
	}
}

func escape(value string) string {
	return strings.ReplaceAll(value, "|", "\\|")
}
