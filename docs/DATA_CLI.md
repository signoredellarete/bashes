# Data CLI

This document describes the command-line utilities for validating and normalizing a Bashes JSON datastore.

Validate a Bashes JSON datastore:

```bash
go run ./cmd/bashes-data validate /path/to/hosts.json
```

Write a normalized copy of a datastore to another file:

```bash
go run ./cmd/bashes-data migrate input-hosts.json output-hosts.json
```

[← Back to README](../README.md)
