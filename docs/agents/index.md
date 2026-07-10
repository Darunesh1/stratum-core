---
title: Agent System Index
purpose: Coordinate multi-agent search, validation, collection, and enrichment workflows.
stage: Overview
compatible_tools: []
next: [search]
---

# Agent System Index

> **Read this guide when** first connecting to Stratum to understand the platform's multi-agent coordination schema, standard workflow steps, and execution guidelines.

## Objective
Your goal is to assist the user in constructing a representative publication corpus for a target technology. Optimize for dataset quality (high recall, controlled noise), not just downloading papers.

## Standard Workflow
The system is executed in a linear, validation-first cycle:
1. **Initialize Project**: Create workspace files and config db.
2. **Upload References**: Import Web of Science or query spreadsheets.
3. **Extract Keywords**: Run TF-IDF frequency calculations to discover query terms.
4. **Configure Search**: Build boolean keywords and query strings.
5. **Validate**: Syntax-check boolean keywords and verify OpenAlex topic codes.
6. **Search / Estimate**: Calculate target page and hit count, check anchor coverage.
7. **Sample Validation**: Inspect random sample papers to estimate precision.
8. **Topics Review**: Review and select topic ID filters.
9. **Refine**: Loop to refine filters if anchors are missing or sample noise is high.
10. **Download**: Pull raw metadata records concurrently to JSONL.
11. **Convert DB**: Ingest records into DuckDB tables.
12. **Impute**: Trigger Crossref DOI, LLM affiliation, and PDF extraction loops.
13. **Analysis**: Execute analytics SQL queries and extract keywords.

## Multi-Agent Roles
*   **Coordinator**: Orchestrates the active project workspace state and logs progress.
*   **Search Agent**: Identifies query search keys and runs topics discovery.
*   **Validation Agent**: Checks anchor paper coverage and evaluates random samples.
*   **Collection Agent**: Handles downloads and loads files into DuckDB.
*   **Analysis Agent**: Computes TF-IDF terms, run SQL queries, and analyzes imputations.

## Examples

### Good Workflow
```
[Validate] ──> [Review Sample] ──> [Download] ──> [Convert DB] ──> [Impute]
```

### Bad Workflow
```
[Download] ──> [Convert DB] ──> [Review Sample]
```
*(Downloading before validating the query causes unnecessary resource consumption and high noise rates).*
