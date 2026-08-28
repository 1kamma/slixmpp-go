package xep

import (
	"fmt"
	"testing"

	"github.com/saret/slixmpp-go/xmpp"
)

func TestCatalogUniqueAndLoadable(t *testing.T) {
	client := xmpp.MustNewClient(xmpp.DefaultConfig("catalog@example.org/test", "unused"))
	if err := RegisterAll(client); err != nil {
		t.Fatal(err)
	}

	ids := make(map[int]struct{}, len(Catalog))
	names := make(map[string]struct{}, len(Catalog))
	for _, descriptor := range Catalog {
		if _, duplicate := ids[descriptor.Number]; duplicate {
			t.Fatalf("duplicate XEP ID %d", descriptor.Number)
		}
		ids[descriptor.Number] = struct{}{}
		name := descriptor.Name()
		if _, duplicate := names[name]; duplicate {
			t.Fatalf("duplicate plugin name %q", name)
		}
		names[name] = struct{}{}

		plugin, err := client.Plugins.Load(name)
		if err != nil {
			t.Fatalf("load %s (%s): %v", name, descriptor.Title, err)
		}
		if plugin.Name() != name {
			t.Fatalf("loaded %s as %s", name, plugin.Name())
		}
		if descriptor.Coverage == CoverageCatalog && len(plugin.Features()) != 0 {
			t.Fatalf("catalog-only %s advertises features: %v", name, plugin.Features())
		}
	}

	if len(ids) != len(Catalog) || len(names) != len(Catalog) {
		t.Fatal(fmt.Errorf("catalog uniqueness mismatch"))
	}
}
