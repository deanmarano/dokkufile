# CLAUDE.md

## What This Project Is

Dokkufile is an Infrastructure-as-Code CLI for [Dokku](https://dokku.com/) servers. It follows a Terraform-like model: define desired state in YAML, diff against live state, and converge. It also installs as a Dokku plugin with a `post-extract` hook that auto-applies `dokkufile.yml` on every `git push`.

## Build & Test

```bash
CGO_ENABLED=0 go build ./...
CGO_ENABLED=0 go test ./...
```

Integration tests require Docker:
```bash
DOKKU_VERSION=0.37.6 ./tests/integration/run.sh
```

Tested against Dokku 0.34.9, 0.35.20, 0.36.11, 0.37.6.

## Architecture

```
main.go                  → cobra root command
cmd/
  inspect.go             → dump live state as YAML/JSON
  plan.go                → diff desired vs live, exit 2 on drift
  apply.go               → execute plan (create/update apps, services, etc.)
  validate.go            → offline schema validation
  import.go              → docker-compose → dokkufile converter
  version.go             → print version (set via ldflags)
pkg/
  schema/schema.go       → Dokkufile struct, validation, Load/Parse
  state/
    state.go             → read live Dokku state via shell commands
    parse.go             → parse dokku command output into structs
  plan/plan.go           → diff two Dokkufiles into a Plan with Steps
  apply/apply.go         → execute Plan steps as dokku commands
  compose/import.go      → docker-compose YAML → Dokkufile conversion
```

## Key Design Decisions

- **Additive-only**: plan/apply never destroy apps or services not in the Dokkufile. Resources not mentioned are left untouched.
- **Scoped reads**: `ReadScoped()` only reads state for apps/services named in the Dokkufile (faster than full `Read()`).
- **Secrets from host env**: The `secrets` field pulls values from the host's environment at apply time, so secrets never appear in the YAML.
- **Phase ordering**: apply creates plugins → services → apps, then configures each app (domains, env, ports, storage, etc.) before deploying.
- **Commands accept both `-f` flag and positional arg**: `plan`, `apply`, and `validate` all support `cmd [file]` and `cmd -f file`. Positional arg overrides the flag.

## Dependencies

Only `cobra` and `yaml.v3`. No ORM, no HTTP client, no test framework beyond stdlib.

## Service Types

15 backing services are supported. The canonical list lives in `pkg/services/services.go` (`services.Types` and `services.PluginURLs`). Both `pkg/state/state.go` and `pkg/compose/import.go` derive from this single source of truth.

The types: clickhouse, couchdb, elasticsearch, mariadb, meilisearch, memcached, mongo, mysql, nats, postgres, rabbitmq, redis, rethinkdb, solr, typesense.

## Testing Patterns

- All packages use table-driven or inline YAML tests with no external fixtures.
- State tests use `FakeRunner` to mock dokku command output.
- Apply tests use `FakeRunner` + `FakeFileRunner` and assert the sequence of commands executed.
- Plan tests construct two `Dokkufile` structs and assert the diff steps.
- Compose tests pass inline YAML strings to `ImportCompose()`.

## Known Limitations

- **SSL round-trip**: `plan` cannot detect cert/key content changes (state reader can't read cert contents back from Dokku).
- **Git repo round-trip**: `plan` cannot detect git repo URL changes (Dokku doesn't expose the source URL).
- **No destructive convergence**: no way to declare "this app should not exist" and have it removed.

## Dokku Plugin Files

- `plugin.toml` — plugin metadata
- `commands` — `dokkufile:inspect`, `dokkufile:plan`, `dokkufile:apply`, `dokkufile:validate`, `dokkufile:version`
- `post-extract` — hook that auto-applies `dokkufile.yml` from app repos
- `dokkufile-bin` — pre-built binary placed by CI release (not tracked in git)
