---
title: Collection Agent Guide
purpose: Download metadata records, verify local files, and ingest JSONL lines into DuckDB tables.
stage: Ingestion & Conversion
compatible_tools: [download, convert_db]
next: [analysis]
---

# Collection Agent Guide

> **Read this guide when** downloading bibliographic records from OpenAlex or ingesting the raw line-by-line JSONL file into DuckDB database tables.

## Objective
Your goal is to run concurrent downloads to a local JSONL file, initialize the DuckDB schema, and populate relational tables.

## Available MCP Tools
* `download`: Pull matching papers from OpenAlex.
* `convert_db`: Ingest JSONL records into DuckDB.

## Recommended Order
1. Run `download` tool to pull records into `collected_papers.jsonl`.
2. Run `convert_db` to write JSONL data to DuckDB.

## Expected Inputs
* Output path coordinates.
* Input JSONL path overrides.

## Expected Outputs
* Populate database tables (`papers`, `authors`, `institutions`, `countries`, `contributions`).
* Counts of loaded papers and authors.

## Things To Avoid
* Do not trigger downloads before checking query hit counts (ensure count matches expectations).
* Do not interrupt conversion transactions once started.

## Examples

### Good Workflow
```
[download] ──> [convert_db]
```

### Bad Workflow
```
[convert_db] (without running download first or checking file size)
```
*(Ingesting missing or partial JSONL files results in table validation errors).*
