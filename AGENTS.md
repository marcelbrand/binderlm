# AGENTS.md — binderlm Project Context & Guidelines

## 1. Project Purpose & Summary
`binderlm` is a standalone Go (1.26+) CLI tool that aggregates distributed Markdown documentation across repositories, microservices, and packages into a unified, hierarchically structured context document and synchronizes it directly to Google Drive for consumption in **Google NotebookLM** and enterprise RAG pipelines.

### Key Functionality
1. **Markdown Aggregation**: Traverses directories and explicit file paths with glob patterns (`**/*.md`), respecting exclusions.
2. **AST-Safe Heading Shifting**: Dynamically demotes headings (H1 → H2 → H3, clamped at H6) based on section hierarchy while strictly protecting code fences and math blocks via `goldmark` AST.
3. **Frontmatter Processing**: Extracts and processes YAML frontmatter in `strip` (default), `table`, or `keep` mode.
4. **Provenance Annotations**: Injects source hints (`> *Source: path/to/file.md*`) for accurate citation in NotebookLM.
5. **Automated TOC**: Generates a nested Table of Contents with CommonMark-compatible anchor slugs.
6. **Google Drive Sync**: Idempotently uploads/updates raw `text/markdown` files via Google Drive API v3 without converting to Google Docs format.

---

## 2. Directory & Architecture Map

```
binderlm/
├── cmd/
│   └── binderlm/
│       ├── main.go               # Cobra CLI root & entry point
│       ├── auth.go               # 'auth status' subcommand
│       ├── build.go              # 'build' subcommand (local assembly)
│       ├── login.go              # 'login' subcommand (interactive developer OAuth)
│       ├── logout.go             # 'logout' subcommand (credential cleanup)
│       ├── setup.go              # 'setup' subcommand (interactive credential setup wizard)
│       ├── sync.go               # 'sync' subcommand (Drive upload & upsert)
│       ├── validate.go           # 'validate' subcommand (config & path linting)
│       └── version.go            # 'version' subcommand
├── internal/
│   ├── config/
│   │   ├── config.go             # Schema structs & YAML unmarshaling (gopkg.in/yaml.v3)
│   │   ├── env.go                # Environment variable overrides & .env file loader
│   │   └── validator.go          # Schema semantic validation
│   ├── validator/
│   │   └── validator.go          # Deep path, glob, frontmatter & auth validation engine
│   ├── parser/
│   │   ├── frontmatter.go        # YAML frontmatter extraction & stripping (goldmark-meta)
│   │   ├── heading_shifter.go    # Goldmark AST walker for safe heading demoting
│   │   └── reader.go             # File discovery & globbing (bmatcuk/doublestar/v4)
│   ├── stitcher/
│   │   ├── assembler.go          # Master document aggregation coordinator
│   │   ├── toc.go                # Nested TOC builder & anchor generator
│   │   └── model.go              # In-memory document and section models
│   └── drive/
│       ├── client.go             # Google Drive API v3 client factory
│       ├── auth.go               # 5-tier auth resolution hierarchy & status inspector
│       ├── oauth.go              # Interactive OAuth2 flow & token cache manager
│       ├── paths.go              # Central ~/.config/binderlm/ path & credential loader
│       └── uploader.go           # Idempotent search, create & update operations
├── binderlm.example.yaml         # Reference configuration template
├── README.md                     # User-facing documentation & quick start
├── SPEC.md                       # Comprehensive technical design & PRD
└── AGENTS.md                     # Agent memory & architecture cheatsheet (this file)
```

---

## 3. Technology Stack & Key Dependencies

| Component | Library / Package | Purpose |
| :--- | :--- | :--- |
| **Language & Runtime** | Go 1.26+ | Statically linked binary (`CGO_ENABLED=0`) |
| **CLI Framework** | `github.com/spf13/cobra` | Commands (`build`, `sync`, `validate`, `version`) and flags |
| **Configuration** | `gopkg.in/yaml.v3` | Strict YAML parsing with line numbers |
| **Markdown AST** | `github.com/yuin/goldmark` | CommonMark AST parsing (safe heading transformations) |
| **Frontmatter** | `github.com/yuin/goldmark-meta` | Frontmatter metadata extraction |
| **Path Globbing** | `github.com/bmatcuk/doublestar/v4` | POSIX-compliant glob matching (`**/*.md`) |
| **Drive API & Auth** | `google.golang.org/api/drive/v3`<br>`golang.org/x/oauth2/google` | Google Drive v3 client & Service Account auth |

---

## 4. Configuration Schema (`binderlm.yaml`)

```yaml
version: "1"

output:
  filename: "project_context_latest.md" # Target output filename
  title: "Master Context Document"       # Top-level H1 title
  description: "Aggregated documentation"
  generate_toc: true                    # Auto-generate Table of Contents
  inject_source_hints: true             # Add '> *Source: path/file.md*'
  frontmatter_mode: "strip"             # 'strip' | 'table' | 'keep'
  max_heading_level: 6                  # Heading depth clamp

drive:
  enabled: true
  folder_id: "GOOGLE_DRIVE_FOLDER_ID"   # Can be overridden by GDRIVE_FOLDER_ID env var
  mime_type: "text/markdown"            # Always text/markdown to avoid Doc conversion

sections:
  - title: "Architecture"
    level: 1
    files:
      - "./docs/architecture.md"
  - title: "Services"
    level: 1
    subsections:
      - title: "Auth Service"
        level: 2
        path: "./services/auth/docs"
        pattern: "*.md"
        exclude:
          - "**/_drafts/**"
```

---

## 5. Development Conventions & Rules

1. **AST Safety First**:
   - Never use naive regex or string replacement to shift Markdown headings (`#`), as this corrupts code blocks, inline code, and blockquotes.
   - Always traverse and rewrite headings using the `goldmark` AST.

2. **Google Drive Idempotency**:
   - Search target folder by parent ID, filename, and `trashed = false`.
   - Update if existing file exists, create if missing.
   - Always upload with MIME type `text/markdown` to prevent auto-conversion.

3. **Authentication Hierarchy (5-Tier Resolution)**:
   - 1st: `GOOGLE_APPLICATION_CREDENTIALS_JSON` (in-memory JSON string for CI/CD).
   - 2nd: `GOOGLE_APPLICATION_CREDENTIALS` (filepath to service account JSON).
   - 3rd: Cached Developer OAuth Token (`~/.config/binderlm/token.json` via `binderlm login` / `binderlm setup`).
   - 4th: Global Service Account (`~/.config/binderlm/service_account.json` via `binderlm setup`).
   - 5th: Application Default Credentials (ADC).
   - Flag Override: `-a, --auth <user|sa>` directly forces user OAuth or Service Account.
   - Folder ID: CLI flag `--folder-id` > Config file `drive.folder_id` > Env var `GDRIVE_FOLDER_ID`.

4. **Error Handling**:
   - Provide clear, actionable error messages with file paths and line numbers where available.
   - Return appropriate exit codes (`0` = success, `1` = general error, `2` = auth failure).

5. **Key Documentation References**:
   - Detailed specification and requirements: [SPEC.md](SPEC.md)
   - CLI usage & examples: [README.md](README.md)
   - Example configuration: [binderlm.example.yaml](binderlm.example.yaml)
