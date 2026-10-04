# Issue tracker: Local Markdown

Issues and specs for this repo live as Markdown files in `.scratch/`.

## Conventions

- One feature per directory: `.scratch/<feature-slug>/`.
- The spec is `.scratch/<feature-slug>/spec.md`.
- Implementation issues are one file per ticket at
  `.scratch/<feature-slug>/issues/<NN>-<slug>.md`, numbered from `01`.
- Triage state is recorded as a `Status:` line near the top of each issue
  file. See `triage-labels.md` for the role strings.
- Comments and conversation history append to the bottom of the file
  under a `## Comments` heading.

## Publishing to the issue tracker

Create the spec or individual issue files at the paths above,
creating directories as needed.

## Fetching a relevant ticket

Read the file at the referenced path. The user will normally pass the
path or issue number directly. Resolve numbers within the relevant
feature directory.

## Wayfinding operations

Used by `/wayfinder`. The map is a file with one child file per ticket.

- Map: `.scratch/<effort>/map.md`, containing Notes, Decisions-so-far,
  and Fog.
- Child ticket: `.scratch/<effort>/issues/NN-<slug>.md`, numbered from
  `01`, with the question in the body. A `Type:` line records
  `research`, `prototype`, `grilling`, or `task`.
- Status: use `open` for unclaimed work, `claimed` while being worked
  on, and `resolved` when answered. These are wayfinding lifecycle
  states, separate from triage roles.
- Blocking: a `Blocked by: NN, NN` line near the top. A ticket is
  unblocked when every listed ticket is `resolved`.
- Frontier: scan the effort's issues for open, unblocked tickets;
  the lowest number wins.
- Claim: set `Status: claimed` and save before starting work.
- Resolve: append the answer under `## Answer`, set `Status: resolved`,
  and append a context pointer (gist and link) to the map's
  Decisions-so-far.
