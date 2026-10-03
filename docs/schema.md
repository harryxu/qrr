# Recipe schema (version 1)

Store one recipe per `<name>.yaml` or `<name>.yml` in the configuration directory's
`commands` folder. Names must match filenames and use lowercase letters, digits,
and hyphens, beginning with a letter. CLI command names are reserved: `run`, `list`, `show`, `add`, `edit`,
`remove`, `validate`, `completion`, `help`, `prompt`, and `schema`.
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
`default`, and `options` for selection types. Parameter names follow command-name
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
