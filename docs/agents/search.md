---
title: Search Agent Guide
purpose: Parse reference lists, perform TF-IDF frequency lookups, and configure boolean search filters.
stage: Search Configuration
compatible_tools: [upload_file, extract_query_and_anchors, configure_project, search]
next: [validation]
---

# Search Agent Guide

> **Read this guide when** parsing an uploaded reference spreadsheet, extracting key phrases, or defining the initial query filters.

## Objective
Your goal is to extract key search phrases from user uploaded papers, write search keywords, and find matching topic filters.

## Available MCP Tools
* `upload_file`: Upload WOS/CSV reference file.
* `extract_query_and_anchors`: Compute TF-IDF scores and DOIs.
* `configure_project`: Update project keywords and topics config.
* `search`: Test query hit count.

## Recommended Order
1. Call `upload_file` to import the reference sheet.
2. Call `extract_query_and_anchors` specifying the title, abstract, and DOI columns.
3. Update config using `configure_project`.
4. Call `search` to check hit count.

## Expected Inputs
* Reference spreadsheet path.
* Title, abstract, and DOI column names.

## Expected Outputs
* High-frequency TF-IDF terms.
* Total search hit count estimates.

## Things To Avoid
* Do not write boolean queries manually if a reference file is available.
* Do not omit TF-IDF frequency checks when identifying search keys.

## Examples

### Good Workflow
```
[upload_file] ──> [extract_query_and_anchors] ──> [configure_project] ──> [search]
```

### Bad Workflow
```
[configure_project] ──> [search] (without extracting query terms first)
```
*(Writing keywords purely on assumption misses key phrasing variants and lowers final dataset recall).*
