# binderlm

> **Git-to-NotebookLM Markdown Aggregator & Sync Tool**  
> Consolidate decentralized markdown documentation across repositories, microservices, and packages into a unified, hierarchically structured context document and synchronize it directly to Google Drive for Google NotebookLM.

[![Go Version](https://img.shields.io/badge/go-1.22+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

---

## 📖 Overview

In modern software architectures, documentation is ideally maintained close to the code (*Docs as Code*) across microservices, libraries, and frontend applications. However, LLM-based research assistants like **Google NotebookLM** and enterprise RAG workflows deliver the best insights when provided with a coherent, centralized master context.

**`binderlm`** bridges this gap:
1. **Aggregates** distributed Markdown files using explicit paths and glob patterns.
2. **Normalizes & Shakes Headings** dynamically (H1 &rarr; H2 &rarr; H3) based on section hierarchy while strictly preserving code fences.
3. **Processes YAML Frontmatter**, stripping raw headers or rendering clean metadata tables and fallback titles.
4. **Injects Source Provenance** annotations to ensure exact citations in NotebookLM.
5. **Generates an Automated Table of Contents (TOC)** with anchor navigation.
6. **Syncs to Google Drive** idempotently using Google Drive API v3 (upserting raw `text/markdown` to preserve formatting).

---

## 🚀 Quick Start

### Installation

```bash
# Install via Go CLI
go install github.com/your-org/binderlm/cmd/binderlm@latest
```

### Basic Commands

```bash
# Build and assemble markdown locally (preview output)
binderlm build --config binderlm.yaml

# Build and sync the aggregated markdown to Google Drive
binderlm sync --config binderlm.yaml

# Dry run sync to verify changes without uploading
binderlm sync --config binderlm.yaml --dry-run
```

---

## ⚙️ Configuration (`binderlm.yaml`)

Define source files, directory globs, section hierarchy, and Google Drive upload targets in `binderlm.yaml`:

```yaml
version: "1"

# Output settings
output:
  filename: "project_context_latest.md"
  generate_toc: true
  inject_source_hints: true
  frontmatter_mode: "strip" # Options: strip | table | keep

# Google Drive Target Configuration
drive:
  enabled: true
  folder_id: "1A2B3C4D5E6F7G8H9I0J_GOOGLE_DRIVE_FOLDER_ID" # Or via GDRIVE_FOLDER_ID env var

# Sections and Document Structure
sections:
  - title: "Architecture & Overview"
    level: 1
    files:
      - "./docs/architecture.md"
      - "./docs/guidelines.md"

  - title: "Common Library"
    level: 1
    path: "./packages/common/docs"
    pattern: "**/*.md"
    recursive: true

  - title: "Microservices"
    level: 1
    subsections:
      - title: "Auth Service"
        level: 2
        path: "./services/auth/docs"
        pattern: "*.md"
      - title: "Payment Service"
        level: 2
        path: "./services/payment/docs"
        pattern: "*.md"

  - title: "Frontend"
    level: 1
    path: "./frontend/docs"
    pattern: "**/*.md"
    recursive: true
```

---

## 🛠️ CLI Reference

| Command | Flags | Description |
| :--- | :--- | :--- |
| `binderlm build` | `-c, --config <path>`<br>`-o, --output <path>` | Assembles local markdown files into a single unified context file. |
| `binderlm sync` | `-c, --config <path>`<br>`--dry-run`<br>`--folder-id <id>` | Assembles markdown and uploads/updates the target file in Google Drive. |
| `binderlm validate` | `-c, --config <path>` | Validates configuration, file paths, and frontmatter syntax. |
| `binderlm version` | | Prints version and build metadata. |

### Global Flags
- `-c, --config string`: Path to config file (default: `binderlm.yaml` or `.binderlm.yaml`).
- `-v, --verbose`: Enable verbose debug logging.
- `--help`: Help for any command.

---

## 🔐 Google Drive Authentication

`binderlm` supports Google Service Account authentication for automated CI/CD and local environments:

1. **Service Account JSON File**:
   Set `GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json`.
2. **Environment Variable (CI/CD Secret)**:
   Set `GOOGLE_APPLICATION_CREDENTIALS_JSON='{"type": "service_account", ...}'`.

Ensure the Service Account email has **Editor** permissions on the target Google Drive folder (`folder_id`).

---

## 🤖 CI/CD Integration (GitHub Actions)

Automatically compile and update your NotebookLM documentation source on every push:

```yaml
name: Sync Docs to NotebookLM

on:
  push:
    branches: [ main ]
    paths:
      - 'docs/**'
      - '**/docs/**'
      - '*.md'
      - 'binderlm.yaml'

jobs:
  stitch-and-sync:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Build & Sync with binderlm
        env:
          GOOGLE_APPLICATION_CREDENTIALS_JSON: ${{ secrets.GDRIVE_SERVICE_ACCOUNT_KEY }}
          GDRIVE_FOLDER_ID: ${{ secrets.GDRIVE_FOLDER_ID }}
        run: |
          go run ./cmd/binderlm sync --config binderlm.yaml
```

---

## 🏗️ Architecture

```
cmd/
  binderlm/
    main.go           # CLI Entrypoint (Cobra root & flags)
    build.go          # 'build' subcommand (local assembly)
    version.go        # 'version' subcommand
internal/
  config/
    config.go         # YAML parser & defaults
    env.go            # Environment variable overrides
    validator.go      # Configuration semantic validation
  parser/
    frontmatter.go    # YAML frontmatter extraction & modes (strip, table, keep)
    heading_shifter.go# AST-safe markdown heading demotion/shifting
    reader.go         # Directory traversal, globbing & exclusions
  stitcher/
    assembler.go      # Section collector, ordering & assembly
    model.go          # Document and section data models
    toc.go            # TOC generator & slug disambiguation
```

For in-depth architectural details, requirements, and technical specifications, refer to [SPEC.md](SPEC.md).

---

## 🗺️ Roadmap

- [x] Initial Architecture & Specification ([SPEC.md](SPEC.md))
- [x] **Phase 1**: Core Parser, Heading Shifter & Assembler (Local `build`)
- [ ] **Phase 2**: Google Drive API v3 Client & Idempotent Upsert (`sync`)
- [ ] **Phase 3**: Multi-repository / Git submodule support & Validation CLI
- [ ] **Phase 4**: Automated GitHub Actions and Release pipeline

---

## 📄 License

Distributed under the Apache 2.0 License. See [LICENSE](LICENSE) for more information.
