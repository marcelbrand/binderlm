package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Default config filenames to search for if none is specified
var DefaultConfigFiles = []string{
	"binderlm.yaml",
	".binderlm.yaml",
	"binderlm.yml",
	".binderlm.yml",
}

// Config represents the top-level configuration for binderlm.
type Config struct {
	Version  string          `yaml:"version"`
	Output   OutputConfig    `yaml:"output"`
	Drive    DriveConfig     `yaml:"drive"`
	Sections []SectionConfig `yaml:"sections"`

	// BaseDir stores the directory containing the config file, used to resolve relative paths
	BaseDir string `yaml:"-"`
}

// OutputConfig configures the master document generation.
type OutputConfig struct {
	Filename          string `yaml:"filename"`
	Title             string `yaml:"title"`
	Description       string `yaml:"description"`
	GenerateTOC       *bool  `yaml:"generate_toc"`
	InjectSourceHints *bool  `yaml:"inject_source_hints"`
	FrontmatterMode   string `yaml:"frontmatter_mode"` // "strip" | "table" | "keep"
	MaxHeadingLevel   int    `yaml:"max_heading_level"`
}

// IsGenerateTOC returns true if TOC generation is enabled (default true).
func (o *OutputConfig) IsGenerateTOC() bool {
	if o.GenerateTOC == nil {
		return true
	}
	return *o.GenerateTOC
}

// IsInjectSourceHints returns true if source provenance annotations are enabled (default true).
func (o *OutputConfig) IsInjectSourceHints() bool {
	if o.InjectSourceHints == nil {
		return true
	}
	return *o.InjectSourceHints
}

// DriveConfig configures Google Drive synchronization.
type DriveConfig struct {
	Enabled  bool   `yaml:"enabled"`
	FolderID string `yaml:"folder_id"`
	MimeType string `yaml:"mime_type"`
}

// SectionConfig configures an individual section or subsection in the document.
type SectionConfig struct {
	Title       string          `yaml:"title"`
	Level       int             `yaml:"level"`
	Files       []string        `yaml:"files,omitempty"`
	Path        string          `yaml:"path,omitempty"`
	Pattern     string          `yaml:"pattern,omitempty"`
	Recursive   bool            `yaml:"recursive,omitempty"`
	Exclude     []string        `yaml:"exclude,omitempty"`
	Subsections []SectionConfig `yaml:"subsections,omitempty"`
}

// DefaultConfig returns a new Config with standard default values.
func DefaultConfig() *Config {
	trueVal := true
	return &Config{
		Version: "1",
		Output: OutputConfig{
			Filename:          "project_context_latest.md",
			Title:             "Master Context Document",
			Description:       "",
			GenerateTOC:       &trueVal,
			InjectSourceHints: &trueVal,
			FrontmatterMode:   "strip",
			MaxHeadingLevel:   6,
		},
		Drive: DriveConfig{
			Enabled:  false,
			FolderID: "",
			MimeType: "text/markdown",
		},
		Sections: []SectionConfig{},
	}
}

// Load reads and parses a binderlm configuration from the specified path or discovers a default file.
func Load(configPath string) (*Config, error) {
	resolvedPath, err := resolveConfigPath(configPath)
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(resolvedPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file at %s: %w", resolvedPath, err)
	}

	cfg := DefaultConfig()
	absPath, err := filepath.Abs(resolvedPath)
	if err != nil {
		absPath = resolvedPath
	}
	cfg.BaseDir = filepath.Dir(absPath)

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse YAML in %s: %w", resolvedPath, err)
	}

	// Apply environment variable overrides
	ApplyEnvOverrides(cfg)

	// Validate config integrity
	if err := Validate(cfg); err != nil {
		return nil, fmt.Errorf("configuration error in %s: %w", resolvedPath, err)
	}

	return cfg, nil
}

// resolveConfigPath returns the provided path or discovers a default config file in the current directory.
func resolveConfigPath(path string) (string, error) {
	if path != "" {
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("config file not found: %s", path)
		}
		return path, nil
	}

	for _, file := range DefaultConfigFiles {
		if _, err := os.Stat(file); err == nil {
			return file, nil
		}
	}

	return "", fmt.Errorf("no config file found. Provide --config or create 'binderlm.yaml'")
}
