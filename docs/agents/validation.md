---
title: Validation Agent Guide
purpose: Verify search query coverage against anchor DOIs, retrieve random samples, and refine topic codes.
stage: Validation
compatible_tools: [search, get_sample, get_topics, configure_project]
next: [collection]
---

# Validation Agent Guide

> **Read this guide when** verifying search recall against anchor DOIs, checking search precision using random samples, or filtering out noisy disciplines.

## Objective
Your goal is to check if the boolean search query covers anchor papers, fetch samples to check for noise, and refine topic filters.

## Available MCP Tools
* `search` (with `check_anchors: true`): Check coverage.
* `get_sample`: Pull random papers to inspect.
* `get_topics`: Group search results by OpenAlex topics.
* `configure_project`: Add selected topic IDs as filters.

## Recommended Order
1. Run `search` with `check_anchors` set to true.
2. If anchors are missing, call `get_sample` to see if query terms are too narrow.
3. Call `get_topics` to view top research fields.
4. Update configuration topics using `configure_project` to filter out irrelevant disciplines.
5. Sample again to verify precision.

## Expected Inputs
* Active project name.
* Sample size bounds.

## Expected Outputs
* List of missing DOIs (if any).
* Clean topic filters map.

## Things To Avoid
* Do not proceed to downloads if anchor coverage is under 90% without a clear explanation.
* Do not treat missing anchors as automatic failures (they might be missing from OpenAlex indexing). Check indexing first.

## Examples

### Good Workflow
```
[search (anchors=true)] ──> [get_topics] ──> [configure_project (topics)] ──> [get_sample]
```

### Bad Workflow
```
[search (anchors=true)] ──> [download] (with missing anchors or high sample noise)
```
*(Failing to refine topic filters leads to importing unrelated journals and broad disciplines).*
