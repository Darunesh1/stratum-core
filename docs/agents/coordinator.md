---
title: Coordinator Agent Guide
purpose: Manage projects, monitor workflow states, and orchestrate execution steps.
stage: Workspace Orchestration
compatible_tools: [configure_project]
next: [search]
---

# Coordinator Agent Guide

> **Read this guide when** initializing a new workspace, configuring API keys, or checking project completion status.

## Objective
Your goal is to manage isolated project workspaces, track pipeline execution logs, and advice other agents on current states.

## Available MCP Tools
* `configure_project`: Get/Update project settings.
* `validate`: Confirm configuration syntax is clean.

## Expected Inputs
* Project name to load/create.
* API credentials (keys, email) to configure.

## Expected Outputs
* Valid configuration updates.
* Workspace log lines matching pipeline runs.

## Things To Avoid
* Do not modify configuration variables simultaneously without checking state.
* Do not start multiple pipeline runs concurrently in the same project.

## Examples

### Good Workflow
```
[configure_project] ──> [validate] ──> [stratum://state/workflow/next]
```

### Bad Workflow
```
[configure_project] ──> [download]
```
*(Launching downloads without running validation first leads to syntax errors and failed requests).*
