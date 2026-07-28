# Changelog

## 0.2.1 - 2026-07-28

### Changed
- `plugin:install`/`plugin:update` now install from the newest published versioned release (via GitHub's `/releases/latest/download/` redirect) instead of the rolling `latest` prerelease, so `dokkufile:version` reports the release tag.

## 0.2.0 - 2026-07-28

### Added
- `--app` flag for `plan` and `apply` commands to scope operations to a single app and its linked services/plugins
- `FilterByApp` method on `Dokkufile` struct (shared by inspect, plan, and apply)
- Full schema reference in README covering all ~30 app fields, global config, mail/auth services, and plugins
- Three example dokkufiles: `examples/rails-postgres.yml`, `examples/node-redis.yml`, `examples/multi-app.yml`
- `CLAUDE.md` development guide for AI-assisted sessions
- 18 schema validation unit tests covering every `Validate()` rule
- `pkg/services` package as single source of truth for 15 service types and plugin URLs
- 10 new compose import field mappings: `extra_hosts`, `tmpfs`, `sysctls`, `shm_size`, `user`, `stop_grace_period`, `labels`, `privileged`, `dns`, `dns_search`
- Custom unmarshal types for `tmpfs` (string/list), `sysctls` (map/list), `labels` (map/list)
- Import warning system: prints skipped fields with reasons and unrecognized fields to stderr
- `ImportResult` return type wrapping `Dokkufile` + `Warnings`
- Documented all CLI flags (`--include-env`, `--format`, `--dry-run`, `--global`, `--app`, exit code 2)
- Documented supported Dokku versions (0.34.9, 0.35.20, 0.36.11, 0.37.6)

### Changed
- Service types deduplicated from 3 hardcoded lists into shared `pkg/services` package
- `ImportCompose` returns `*ImportResult` instead of `*schema.Dokkufile`
- `filterByApp` moved from unexported in `cmd/inspect.go` to exported `FilterByApp` method on `schema.Dokkufile`

### Fixed
- Nested `dokku` calls no longer inherit `DOKKU_APP_NAME` when dokkufile runs as a plugin. It leaked into the state reader and applier, so nested commands treated the app as implicit and misparsed positional args (`ps:scale altoids web=1` → "Missing count for process type altoids"; `domains:report altoids --domains-app-vhosts` → "Invalid flag passed"). This produced phantom drift on domains/scale/git in `plan` and aborted `apply` mid-run — only via the plugin/git-push path, not standalone.
- `plan` no longer renders composite-field changes (cron, scale, git, docker_options, dns, letsencrypt, network) as a bare `"" → ""`, which read like a no-op. Shows `<field> changed` when there is no before/after string.
- `dokkufile-bin` added to `.gitignore`

## Earlier Development

### Core Features
- Declarative YAML-based Infrastructure as Code for Dokku servers
- `inspect` command: dump live server state as YAML/JSON with `--global`, `--include-env`, `--format` flags
- `plan` command: diff desired vs live state, exit code 2 on drift, `--format json`
- `apply` command: execute plan to converge state with `--dry-run` support
- `validate` command: offline schema validation
- `import` command: convert docker-compose.yml to dokkufile format
- `version` command: print version/commit/date set via ldflags

### App Configuration
- Image-based and git-based deployments (mutually exclusive)
- Domains, ports (with scheme detection), environment variables, secrets (from host env)
- Service links, persistent storage, docker options (build/deploy/run phases)
- Let's Encrypt and custom SSL certificate support
- Process scaling, resource limits/reservations (CPU, memory, GPU)
- Health checks, cron jobs, zero-downtime deploy checks
- Network configuration, nginx properties, nginx template (sigil)
- Proxy support: nginx, caddy, haproxy, traefik with per-property diffing
- Builder selection: dockerfile, herokuish, pack, nixpacks with sub-plugin paths
- Buildpacks, registry config, deployment scripts (predeploy/postdeploy)
- Process management (restart policy, procfile path), log management, scheduler config
- App locking, maintenance mode, auth (directory + frontend with OIDC)

### Backing Services
- 15 service types: postgres, redis, mysql, mariadb, mongo, memcached, rabbitmq, elasticsearch, clickhouse, couchdb, meilisearch, nats, rethinkdb, solr, typesense
- Service image version pinning
- Auto-detection from docker-compose image names

### Server Management
- Global config: domains, nginx, proxy, network, builder, registry, logs, scheduler
- Plugin management: install/uninstall/update detection
- Mail services and auth directories/frontends

### Dokku Plugin
- Installs as a Dokku plugin with `plugin:install`
- `post-extract` hook auto-applies `dokkufile.yml` on `git push`
- Fails deploy on validation or apply errors
- Scoped state reading for faster plan/apply/inspect

### Infrastructure
- Integration tests against Dokku 0.34.9, 0.35.20, 0.36.11, 0.37.6
- CI builds for linux/amd64 and linux/arm64
- Minimal dependencies: only cobra and yaml.v3
