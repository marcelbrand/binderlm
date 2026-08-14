# Basic Sample Documentation

Aggregated documentation for the basic example repository.

## Table of Contents

* [Overview](#overview)
  * [Introduction](#introduction)
    * [Core Principles](#core-principles)
* [Guides](#guides)
  * [Getting Started & Usage](#getting-started-usage)
    * [CLI Usage](#cli-usage)
      * [Python Client Example](#python-client-example)

---

## Overview

> *Source: `docs/intro.md`*

### Introduction

Welcome to the sample service documentation.

#### Core Principles

- Keep services decoupled.
- Document APIs close to code.

## Guides

> *Source: `docs/usage.md`*

### Getting Started & Usage

Follow these steps to run the service locally.

#### CLI Usage

Run the following shell commands:

```bash
# Clone the repository
git clone https://github.com/example/repo.git

# Run the server
./bin/server --port=8080
```

##### Python Client Example

```python
# Initialize client
client = ExampleClient(api_key="secret")
# Perform query
res = client.query("test")
```

