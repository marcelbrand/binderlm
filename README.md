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
| `binderlm setup` | | Interactive setup wizard for OAuth client credentials or Service Account keys. |
| `binderlm login` | `--port <port>`<br>`--no-browser`<br>`--client-id <id>`<br>`--client-secret <secret>`<br>`--timeout <duration>` | Initiates interactive OAuth2 browser login for personal Google Drive accounts. |
| `binderlm auth status` | `-a, --auth <mode>` | Inspects active Google Drive authentication method and account. |
| `binderlm logout` | | Deletes locally cached OAuth credentials from disk. |
| `binderlm build` | `-c, --config <path>`<br>`-o, --output <path>`<br>`--stdout` | Assembles local markdown files into a single unified context file. |
| `binderlm sync` | `-c, --config <path>`<br>`-a, --auth <mode>`<br>`-e, --env-file <path>`<br>`--folder-id <id>`<br>`--dry-run`<br>`--keep-local`<br>`-o, --output <path>` | Assembles markdown and uploads/updates the target file in Google Drive. |
| `binderlm validate` | `-c, --config <path>`<br>`--strict`<br>`--check-drive` | Validates configuration schema, file paths, globs, and frontmatter syntax offline. Use `--check-drive` to verify Google Drive credentials and folder access. |
| `binderlm version` | | Prints version and build metadata. |

### Global Flags
- `-a, --auth <mode>`: Authentication mode override: `'user'` (personal OAuth), `'sa'` (Service Account), or `'auto'` (default).
- `-c, --config <path>`: Path to config file (default: `binderlm.yaml` or `.binderlm.yaml`).
- `-e, --env-file <path>`: Path to `.env` file to load environment variables from.
- `-v, --verbose`: Enable verbose debug logging.
- `-h, --help`: Help for any command.

---

## 🔐 Google Drive Setup & Authentication

`binderlm` stores local configurations and credentials centrally in `~/.config/binderlm/`:

```
~/.config/binderlm/
├── client.json           # OAuth Desktop Client ID & Secret
├── token.json            # Cached OAuth User Token (created by 'binderlm login')
└── service_account.json  # Global Service Account Key (optional)
```

---

### 1. Interactive Setup Wizard (`binderlm setup`)

The quickest way to configure credentials on your local machine is the interactive setup wizard:

```bash
# Launch interactive setup wizard
binderlm setup
```

The wizard guides you through:
1. **OAuth2 Developer Account:** Configures your Desktop Client credentials in `~/.config/binderlm/client.json` and initiates browser login.
2. **Service Account Key:** Saves your downloaded Service Account JSON in `~/.config/binderlm/service_account.json`.

---

### 2. Manual Google Cloud Console Setup

#### For Personal Accounts (OAuth 2.0 Desktop App):
1. In the [Google Cloud Console](https://console.cloud.google.com/), navigate to **APIs & Services** &rarr; **Credentials**.
2. Click **+ Create Credentials** &rarr; **OAuth client ID**.
3. Set **Application type** to **Desktop App** (Desktop-Anwendung).
4. In **OAuth consent screen**:
   * If in *Testing* status, add your `@gmail.com` address under **Test users**.
   * *(Recommended)* Set publishing status to **In Production** (External) so your OAuth refresh token never expires.

#### For CI/CD & Teams (Service Accounts):
1. Go to **IAM & Admin** &rarr; **Service Accounts** &rarr; **Create Service Account**.
2. Go to the **Keys** tab &rarr; **Add Key** &rarr; **Create new key (JSON)**.
3. In Google Drive, share your target folder with the Service Account email as **Editor** (uncheck "Notify people").
4. Copy the **Folder ID** from the folder's browser address bar.

---

### 3. Developer Commands Reference

```bash
# Interactive setup wizard
binderlm setup

# Log in with your personal Google account
binderlm login

# Check active authentication status
binderlm auth status

# Sync directly using your personal Drive storage quota
binderlm sync --config binderlm.yaml

# Log out and clear cached credentials
binderlm logout
```

> [!TIP]
> **Explicit `--env-file` Flag:** 
> Any `binderlm` command accepts `-e, --env-file <path>` (e.g. `binderlm sync -e .env`) to safely load custom environment variables without global shell exports.

---

## ⚡ The Hybrid Personal Drive + CI/CD Workflow Pattern

Google Drive API enforces a **0 MB storage quota** for Service Accounts on personal (`@gmail.com`) Google Drive accounts. When a Service Account creates a new file, it triggers a `storageQuotaExceeded` error. However, when a file is **updated** (`Files.Update`), Google charges storage quota to the **file owner** rather than the editor.

`binderlm` enables a seamless hybrid pattern that bypasses this limitation without requiring manual file uploads in the Google Drive web UI:

```
┌────────────────────────────────────────────────────────────────────────┐
│ 1. Local Developer Initial Setup (One-Time)                            │
│                                                                        │
│  $ binderlm login   ──► Authenticates with personal Google Account     │
│  $ binderlm sync    ──► Creates target file in Google Drive            │
│                         (Owned by developer, uses personal 15GB quota) │
│  Drive Web UI       ──► Share folder with Service Account as 'Editor'  │
└──────────────────────────────────┬─────────────────────────────────────┘
                                   │
                                   ▼
┌────────────────────────────────────────────────────────────────────────┐
│ 2. Automated CI/CD Execution (On Every Git Push)                       │
│                                                                        │
│  GitHub Actions / GitLab CI                                            │
│  $ binderlm sync    ──► Authenticates with Service Account Key Secret  │
│                         Detects existing file & executes Files.Update  │
│                         (Zero quota errors, zero browser interactions) │
└────────────────────────────────────────────────────────────────────────┘
```

---

### 5. Environment Variables & Auth Hierarchy

`binderlm` automatically resolves credentials in the following order (or explicitly override via `-a, --auth <user|sa>`):
1. `GOOGLE_APPLICATION_CREDENTIALS_JSON` — In-memory JSON string (ideal for CI/CD secrets).
2. `GOOGLE_APPLICATION_CREDENTIALS` — Local file path to Service Account JSON key.
3. Cached OAuth User Token (`~/.config/binderlm/token.json` via `binderlm login` / `binderlm setup`).
4. Global Service Account Key (`~/.config/binderlm/service_account.json` via `binderlm setup`).
5. Google Application Default Credentials (ADC).

| Variable | Description |
| :--- | :--- |
| `GOOGLE_APPLICATION_CREDENTIALS` | Local file path to the Service Account JSON key (e.g. `./credentials.json`). |
| `GOOGLE_APPLICATION_CREDENTIALS_JSON` | Raw JSON string content of the Service Account key (ideal for CI/CD secrets). |
| `GDRIVE_FOLDER_ID` | Optional default folder ID override (can also be configured in `binderlm.yaml` or `--folder-id`). |
| `GDRIVE_OAUTH_CLIENT_ID` | Optional custom Google OAuth Client ID override. |
| `GDRIVE_OAUTH_CLIENT_SECRET` | Optional custom Google OAuth Client Secret override. |
| `BINDERLM_AUTH_MODE` | Optional default authentication mode override (`user`, `sa`, or `auto`). |
| `BINDERLM_CONFIG_DIR` | Optional custom configuration directory override (default: `~/.config/binderlm`). |

---

## 🐳 Docker Usage

`binderlm` is available as a lightweight, secure container image via GitHub Container Registry (`ghcr.io`):

```bash
# Display help
docker run --rm ghcr.io/marcelbrand/binderlm:latest --help

# Validate documentation structure in current repository
docker run --rm -v "$(pwd):/workspace" -w /workspace \
  ghcr.io/marcelbrand/binderlm:latest validate -c binderlm.yaml

# Assemble local master context markdown file
docker run --rm -v "$(pwd):/workspace" -w /workspace \
  ghcr.io/marcelbrand/binderlm:latest build -c binderlm.yaml -o context.md

# Synchronize directly to Google Drive using Service Account credentials
docker run --rm -v "$(pwd):/workspace" -w /workspace \
  -e GOOGLE_APPLICATION_CREDENTIALS_JSON="${GDRIVE_SERVICE_ACCOUNT_KEY}" \
  ghcr.io/marcelbrand/binderlm:latest sync -c binderlm.yaml
```

---

## 🤖 CI/CD Integration (GitHub Actions)

### 1. Using the Official GitHub Action

You can easily integrate `binderlm` into your GitHub Actions workflow using the official action without needing to set up Go or install dependencies:

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
  workflow_dispatch:

jobs:
  stitch-and-sync:
    name: Aggregate & Sync Documentation
    runs-on: ubuntu-latest
    steps:
      - name: Checkout Repository
        uses: actions/checkout@v4

      - name: Validate Docs Configuration
        uses: marcelbrand/binderlm@v1
        with:
          command: validate
          config: binderlm.yaml
          strict: true

      - name: Compile and Sync to Google Drive
        uses: marcelbrand/binderlm@v1
        with:
          command: sync
          config: binderlm.yaml
          google-application-credentials-json: ${{ secrets.GOOGLE_APPLICATION_CREDENTIALS_JSON }}
```

### Action Inputs

| Input | Description | Default |
| :--- | :--- | :--- |
| `command` | Command to run (`sync`, `build`, `validate`). | `sync` |
| `config` | Path to `binderlm.yaml` configuration file. | `binderlm.yaml` |
| `folder-id` | Google Drive target folder ID override. | `""` |
| `auth-mode` | Auth mode override (`auto`, `sa`, `user`). | `auto` |
| `dry-run` | Run in dry-run mode without modifying Google Drive. | `false` |
| `keep-local` | Retain the assembled markdown file after syncing. | `false` |
| `output` | Output filepath for assembled markdown (`build` or `sync`). | `""` |
| `strict` | Treat warnings as fatal errors in `validate`. | `false` |
| `check-drive` | Verify Google Drive access during `validate`. | `false` |
| `google-application-credentials-json` | Service account JSON secret for Google Drive API. | `""` |
| `image` | Custom Docker image tag to execute. | `ghcr.io/marcelbrand/binderlm:latest` |

---

## 🏗️ Architecture

```
cmd/
  binderlm/
    main.go           # CLI Entrypoint (Cobra root & flags)
    auth.go           # 'auth status' subcommand
    build.go          # 'build' subcommand (local assembly)
    login.go          # 'login' subcommand (interactive developer OAuth)
    logout.go         # 'logout' subcommand (credential cleanup)
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
    auth.go           # 5-tier auth resolution hierarchy & status inspector
    client.go         # Google Drive v3 client factory
    oauth.go          # Interactive OAuth2 flow & token cache manager
    paths.go          # Central ~/.config/binderlm/ credential storage
    uploader.go       # Idempotent file search, create & update
```

For in-depth architectural details, requirements, and technical specifications, refer to [SPEC.md](SPEC.md).

---

## 🗺️ Roadmap

- [x] Initial Architecture & Specification ([SPEC.md](SPEC.md))
- [x] **Phase 1**: Core Parser, Heading Shifter & Assembler (Local `build`)
- [x] **Phase 2**: Google Drive API v3 Client, Idempotent Upsert & Dry-Run (`sync`)
- [x] **Phase 3**: Configuration & Path Validation Subcommand (`validate`), Unit & Golden Tests, CI/CD Workflows
- [x] **Phase 4**: Interactive Setup Wizard (`binderlm setup`), Developer OAuth Login (`binderlm login`), Mode Overrides (`--auth`), and Hybrid CI/CD Documentation
- [x] **Phase 5**: Docker Containerization, GitHub Actions Integration & Automated Release Pipeline

---

## 📄 License

Distributed under the Apache 2.0 License. See [LICENSE](LICENSE) for more information.
