# Domain Docs

How engineering skills consume this repo's domain documentation.

## Before exploring, read these

- `GLOSSARY.md` at the repo root.
- ADRs in `docs/adr/` that concern the area being explored.

If these files do not exist, proceed silently. The `/domain-modeling`
skill creates them when terms or decisions are actually resolved.

## File structure

This repo uses a single-context layout:

- `GLOSSARY.md`: shared domain vocabulary.
- `docs/adr/`: architecture decision records, named
  `NNNN-<decision-slug>.md`.

## Use the glossary's vocabulary

When naming a domain concept in an issue title, refactor proposal,
hypothesis, or test name, use the term defined in `GLOSSARY.md`.

If a needed concept is absent, reconsider whether the proposed term
fits the project. Record a real vocabulary gap for `/domain-modeling`.

## Flag ADR conflicts

If a proposal contradicts an existing ADR, identify the ADR explicitly
and explain why the decision should be reopened.
