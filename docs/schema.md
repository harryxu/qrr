# Recipe schema (version 1)

Store one recipe per `<name>.yaml` or `<name>.yml` in the configuration directory's
`commands` folder. Names must match filenames and use lowercase letters, digits,
and hyphens, beginning with a letter. CLI command names are reserved: `run`, `list`, `show`, `add`, `edit`,
`rename`, `remove`, `validate`, `completion`, `help`, `prompt`, and `schema`.
Unknown fields, duplicate YAML keys, duplicate parameter names, multiple YAML
documents, and unsupported versions are errors.

```yaml
version: 1
name: greet
description: Print a greeting
type: command
params:
  - name: person
    type: input
    prompt: Your name
    default: World
command:
  - echo
  - 'Hello, {{ .person }}!'
```

## Parameters

Each parameter has `name`, `type`, optional `prompt`, optional `required`, optional
`default`, and either `options` or `options_command` for selection types. Parameter names follow command-name
syntax, and cannot shadow global flags or help.

| Type | Value | CLI form |
| --- | --- | --- |
| `input` | String | `--person 'Jane Doe'` |
| `select` | String from options | `--quality high` |
| `multiselect` | Array of option strings | `--languages en,ja` or `--languages=` |
| `confirm` | Boolean | `--enabled` or `--enabled=false` |

Option entries contain `value` and optional `label`. Values must be nonempty and
unique. Quote numeric-looking values such as `"1440"`. Selection values must occur
in the option list. Multiselect values cannot repeat. Defaults must have the
correct types. Without a default, values start as an empty string, empty array, or
false. Required values cannot be empty; a required confirm must be true.

Comma-separated multi-select values do not support commas inside individual option
values. Input values are not trimmed or evaluated as shell expressions.

### Dynamic options

For `select` and `multiselect`, replace `options` with `options_command`, a
nonempty argv array. The executable can be a path or a command found on PATH.
Arguments are passed literally: no qrr template expansion, variable expansion,
or implicit shell evaluation occurs. Paths are resolved from the invocation's
working directory. Providers inherit the environment and receive EOF on stdin.
For pipelines or multiline scripts, invoke the shell explicitly:

```yaml
params:
  - name: container
    type: select
    prompt: Select a running container
    required: true
    options_command:
      - bash
      - -o
      - pipefail
      - -c
      - |
        docker container ls --filter status=running --format json |
          jq -s 'map({label: (.Names + " · " + .Image), value: .ID})'
```

Stdout must contain exactly one JSON array of objects with a nonempty string
`value` and optional string `label`. Unknown fields, duplicate values, malformed
JSON, and comma-containing multiselect values are rejected. An omitted or empty
label displays the value. For example:

```json
[{"label":"web · nginx:alpine","value":"a1b2c3d4"}]
```

The provider runs once immediately before prompting for an omitted parameter.
The returned options validate the interactive choice and any configured default.
An empty array or a default absent from the current options produces an error.
Single-select menus accept direct keyword filtering; multiselect menus use `/`
to filter, Esc to leave filtering, and Space to toggle a choice.

Explicit parameter flags skip both the provider and the prompt. Non-interactive
execution uses supplied values or defaults and never runs providers. These values
are checked for type, requiredness, and multiselect uniqueness, but cannot be
checked against a live option list. Validation, help, schema output, and shell
completion never execute providers; completion offers no dynamic option values.
Interactive `--dry-run` still runs providers to collect choices, then prints the
recipe's argv without executing it. Use explicit flags or `--non-interactive` for
a dry-run without provider execution.

Providers have a 30-second timeout and a 1 MiB stdout limit. Failures include up
to 16 KiB of captured stderr; successful stderr is discarded. Ctrl+C cancels the
query and its process group. Provider scripts should only retrieve options;
diagnostic text belongs on stderr. Programs such as Bash, Docker, and jq are
external dependencies of the recipe, not dependencies of the qrr executable.

## Execution and templates

`command` is a nonempty argv array. Each entry is rendered independently with Go
`text/template`. Use `{{ .person }}` to reference a parameter and
`{{ join .languages "," }}` to join a multi-select value. Missing parameter references
fail validation. Templates have no file-reading or command-execution functions.

`optional_args` appends conditional argv groups in declaration order:

```yaml
optional_args:
  - when: languages
    args:
      - --languages
      - '{{ join .languages "," }}'
args_tail:
  - --
  - '{{ .url }}'
```

`when` names a parameter and includes the group when its value is nonempty or true.
The final order is `command`, included `optional_args`, then `args_tail`. Template
output is never split into more arguments. A shell is never invoked implicitly.
Use `--` before untrusted positional input when the target program supports it.

Only `type: command` is implemented. Handler fields and other workflow extensions
are rejected. Validation is local and does not check installed programs or access
the network. Use `--dry-run --json --non-interactive` to inspect the rendered argv.
