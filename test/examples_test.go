package test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/marcelbrand/binderlm/internal/config"
	"github.com/marcelbrand/binderlm/internal/stitcher"
)

func TestExamplesIntegration(t *testing.T) {
	examples := []struct {
		name       string
		configPath string
		goldenPath string
	}{
		{
			name:       "Basic Example",
			configPath: "../examples/basic/binderlm.yaml",
			goldenPath: "../examples/basic/expected_output.md",
		},
		{
			name:       "Microservices Example",
			configPath: "../examples/microservices/binderlm.yaml",
			goldenPath: "../examples/microservices/expected_output.md",
		},
	}

	for _, ex := range examples {
		t.Run(ex.name, func(t *testing.T) {
			absConfig, err := filepath.Abs(ex.configPath)
			if err != nil {
				t.Fatalf("failed to get abs path for %s: %v", ex.configPath, err)
			}

			cfg, err := config.Load(absConfig)
			if err != nil {
				t.Fatalf("failed to load config %s: %v", absConfig, err)
			}

			assembler := stitcher.NewAssembler(cfg)
			doc, err := assembler.Assemble(context.Background())
			if err != nil {
				t.Fatalf("assembly failed for %s: %v", ex.name, err)
			}

			goldenBytes, err := os.ReadFile(ex.goldenPath)
			if err != nil {
				t.Fatalf("failed to read golden file %s: %v", ex.goldenPath, err)
			}

			if string(doc.Content) != string(goldenBytes) {
				t.Errorf("assembled document does not match golden expected output for %s\nGot:\n%s\nExpected:\n%s",
					ex.name, string(doc.Content), string(goldenBytes))
			}
		})
	}
}
