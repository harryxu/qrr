# Documentation

qrr (Quick Recipe Runner) is a standalone Go CLI that turns complex commands
into reusable YAML shortcuts with interactive prompts.

Start with the usage guide to run a recipe, then use the schema when
writing your own.

| Task | Read |
| --- | --- |
| Build qrr and run the included greeting | [Usage: build and quick start](usage.md#build-and-install) |
| Find where recipes are stored | [Usage: configuration](usage.md#configuration) |
| Search, rename, or edit from the selector | [Usage: interactive selector](usage.md#choose-a-recipe-interactively) |
| Create, inspect, validate, or remove recipes | [Usage: manage recipes](usage.md#manage-recipes) |
| Turn an existing command into a recipe with an agent | [Usage: AI authoring](usage.md#create-recipes-with-an-ai-agent) |
| Use flags, defaults, dynamic choices, or dry-runs | [Usage: parameters and execution](usage.md#parameters-and-execution) |
| Write YAML and understand argv rendering | [Recipe schema](schema.md) |
| Adapt working recipes | [Example recipes](../examples/commands) |
| Understand dependency roles and upgrade checks | [Dependencies](dependencies.md) |

## Maintainers and agents

Read [AGENTS.md](../AGENTS.md) for project constraints, package boundaries, and
required verification. The repository-specific workflow guides are:

- [Issue tracker](agents/issue-tracker.md): local specs, tickets, and wayfinding.
- [Triage labels](agents/triage-labels.md): the five triage roles and tracker strings.
- [Domain docs](agents/domain.md): glossary and architecture decision conventions.

The [schema](schema.md) describes implemented behavior and is embedded in
`qrr schema`. Proposed features in implementation plans are not configuration
contracts. See [current limits](usage.md#current-limits) before relying on handlers,
workflows, or download failure recovery.
