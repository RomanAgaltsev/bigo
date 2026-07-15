package report

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// TestGoldensValidate validates every golden document against the normative
// schema. Because old goldens are never rewritten except by deliberate
// regeneration, this doubles as the additive-compatibility check: a schema
// change that invalidates an existing document is a compatibility break.
func TestGoldensValidate(t *testing.T) {
	compiler := jsonschema.NewCompiler()
	schema, err := compiler.Compile(filepath.Join("..", "..", "schema", "report.schema.json"))
	if err != nil {
		t.Fatalf("compile schema: %v", err)
	}
	entries, err := os.ReadDir(filepath.Join("testdata", "golden"))
	if err != nil {
		t.Fatalf("read goldens: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no golden documents found")
	}
	for _, e := range entries {
		t.Run(e.Name(), func(t *testing.T) {
			f, err := os.Open(filepath.Join("testdata", "golden", e.Name()))
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = f.Close() }()
			doc, err := jsonschema.UnmarshalJSON(f)
			if err != nil {
				t.Fatal(err)
			}
			if err := schema.Validate(doc); err != nil {
				t.Errorf("%s does not validate: %v", e.Name(), err)
			}
		})
	}
}
