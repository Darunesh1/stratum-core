---
title: Analysis Agent Guide
purpose: Execute metadata imputation pipelines, extract keywords, and run custom analytical queries.
stage: Imputation & Analysis
compatible_tools: [impute]
next: [completion]
---

# Analysis Agent Guide

> **Read this guide when** running metadata enrichment stages (Crossref/LLM/PDF imputation) or querying the populated database.

## Objective
Your goal is to run imputation routines (Crossref lookups, LLMs, PDF extraction) to fill missing affiliations, and perform data searches.

## Available MCP Tools
* `impute`: Run Crossref, LLM, or PDF imputation.

## Recommended Order
1. Run `impute` with `pipeline: "crossref"` to fill affiliations by DOI.
2. Run `impute` with `pipeline: "llm"` to classify affiliation strings.
3. Run `impute` with `pipeline: "pdf"` to extract affiliations from PDF files.
4. Run analytical SQL select queries to draw statistics.

## Expected Inputs
* Active database target name.
* Limit count for PDF manuscript downloads.

## Expected Outputs
* Populate `country_code` and `institution_id` in `contributions`.
* Populated `country_imputation_audit` records.

## Things To Avoid
* Do not run LLM imputation before running the faster, cheaper Crossref lookup.
* Do not run PDF extraction without specifying limits to avoid high network load.

## Examples

### Good Workflow
```
[impute (crossref)] ──> [impute (llm)] ──> [impute (pdf)]
```

### Bad Workflow
```
[impute (pdf)] (without running crossref first)
```
*(Running expensive PDF extraction first wastes time and calls on metadata that could have been resolved via Crossref APIs).*
