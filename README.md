# binderlm

> **Git-to-NotebookLM Markdown Aggregator & Sync Tool**  
> Consolidate decentralized markdown documentation across repositories, microservices, and packages into a unified, hierarchically structured context document and synchronize it directly to Google Drive for Google NotebookLM.

[![Go Version](https://img.shields.io/badge/go-1.26+-blue.svg)](https://golang.org)
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
| `binderlm build` | `-c, --config <path>`<br>`-o, --output <path>`<br>`--stdout` | Assembles local markdown files into a single unified context file. |
| `binderlm sync` | `-c, --config <path>`<br>`--folder-id <id>`<br>`--dry-run`<br>`--keep-local` | Assembles markdown and uploads/updates the target file in Google Drive. |
| `binderlm validate` | `-c, --config <path>`<br>`--strict`<br>`--check-drive` | Validates configuration schema, file paths, globs, frontmatter syntax, and Drive auth. |
| `binderlm version` | | Prints version and build metadata. |

### Global Flags
- `-c, --config string`: Path to config file (default: `binderlm.yaml` or `.binderlm.yaml`).
- `-v, --verbose`: Enable verbose debug logging.
- `--help`: Help for any command.

---

## 🔐 Google Drive Setup & Authentication

Follow these steps to configure Google Drive sync with a Google Service Account:

### 1. Enable Google Drive API in GCP
1. Go to the [Google Cloud Console](https://console.cloud.google.com/).
2. Create a new project or select an existing project.
3. Navigate to **APIs & Services** &rarr; **Library**.
4. Search for **Google Drive API** and click **Enable**.

### 2. Create a Service Account & Download Key
1. Go to **IAM & Admin** &rarr; **Service Accounts**.
2. Click **Create Service Account**, give it a name (e.g. `binderlm-sync`), and click **Done**.
3. Click on the newly created service account &rarr; select the **Keys** tab.
4. Click **Add Key** &rarr; **Create new key** &rarr; choose **JSON** format.
5. Download the JSON key file (e.g. `credentials.json`). See [`credentials.example.json`](credentials.example.json) for reference.

### 3. Share Target Google Drive Folder with the Service Account
1. Open Google Drive in your browser.
2. Create or navigate to the folder where you want your NotebookLM sources stored.
3. Click **Share** &rarr; paste the Service Account email address (e.g. `binderlm-sync@your-project.iam.gserviceaccount.com`).
4. Grant **Editor** permissions and uncheck "Notify people" (service accounts cannot receive emails).
5. Copy the **Folder ID** from the folder URL (the string after `/folders/` in your browser address bar).

### 4. Environment Variables

| Variable | Description |
| :--- | :--- |
| `GOOGLE_APPLICATION_CREDENTIALS` | Local file path to the Service Account JSON key (e.g. `./credentials.json`). |
| `GOOGLE_APPLICATION_CREDENTIALS_JSON` | Raw JSON string content of the Service Account key (ideal for CI/CD secrets). |
| `GDRIVE_FOLDER_ID` | Optional default folder ID override (can also be configured in `binderlm.yaml` or `--folder-id`). |

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
          go-version: '1.26'

      - name: Validate & Sync with binderlm
        env:
          GOOGLE_APPLICATION_CREDENTIALS_JSON: ${{ secrets.GDRIVE_SERVICE_ACCOUNT_KEY }}
          GDRIVE_FOLDER_ID: ${{ secrets.GDRIVE_FOLDER_ID }}
        run: |
          go run ./cmd/binderlm validate --config binderlm.yaml
          go run ./cmd/binderlm sync --config binderlm.yaml
```

---

## 🏗️ Architecture

```
cmd/
  binderlm/
    main.go           # CLI Entrypoint (Cobra root & flags)
    build.go          # 'build' subcommand (local assembly)
    sync.go           # 'sync' subcommand (Google Drive upsert & dry-run)
    validate.go       # 'validate' subcommand (static & deep linting)
    version.go        # 'version' subcommand
internal/
  config/
    config.go         # YAML parser & defaults
    env.go            # Environment variable overrides
    validator.go      # Configuration semantic validation
  validator/
    validator.go      # Deep path, glob, frontmatter & auth validation
  parser/
    frontmatter.go    # YAML frontmatter extraction & modes (strip, table, keep)
    heading_shifter.go# AST-safe markdown heading demotion/shifting
    reader.go         # Directory traversal, globbing & exclusions
  stitcher/
    assembler.go      # Section collector, ordering & assembly
    model.go          # Document and section data models
    toc.go            # TOC generator & slug disambiguation
  drive/
    auth.go           # Service Account auth resolution (JSON & file)
    client.go         # Google Drive v3 client factory
    uploader.go       # Idempotent file search, create & update
```

For in-depth architectural details, requirements, and technical specifications, refer to [SPEC.md](SPEC.md).

---

## 🗺️ Roadmap

- [x] Initial Architecture & Specification ([SPEC.md](SPEC.md))
- [x] **Phase 1**: Core Parser, Heading Shifter & Assembler (Local `build`)
- [x] **Phase 2**: Google Drive API v3 Client, Idempotent Upsert & Dry-Run (`sync`)
- [x] **Phase 3**: Configuration & Path Validation Subcommand (`validate`), Unit & Golden Tests, CI/CD Workflows
- [ ] **Phase 4**: Interactive Developer OAuth Login (`binderlm login`) for personal Google Drive accounts
- [ ] **Phase 5**: Automated GitHub Actions and Release pipeline

---

## 📄 License

Distributed under the Apache 2.0 License. See [LICENSE](LICENSE) for more information.
