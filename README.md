# Dokkufile

Infrastructure as Code for [Dokku](https://dokku.com). Define your app's desired state in a declarative YAML file and let dokkufile converge it.

## Install

```bash
dokku plugin:install https://github.com/deanmarano/dokkufile.git dokkufile
```

To update:

```bash
dokku plugin:update dokkufile
```

## Quick Start

1. Generate a dokkufile from your existing app:

```bash
dokku dokkufile:inspect myapp > dokkufile.yml
```

2. Commit `dokkufile.yml` to your app's git repo.

3. On the next `git push`, dokkufile automatically validates and applies your config before the build starts.

That's it. Your app's infrastructure is now version-controlled.

## How It Works

Each app gets its own `dokkufile.yml` in its git repo. When you `git push` to dokku, a `post-extract` hook:

1. Detects `dokkufile.yml` in the repo
2. Validates the schema
3. Diffs desired state against live server state
4. Applies any changes (domains, ports, storage, etc.)
5. Build proceeds with the updated config

This means build-time config like docker options, buildpacks, and builder selection are applied before the build, so they take effect immediately.

If validation or apply fails, the deploy is aborted.

## Schema

```yaml
version: "1"

services:
  myapp-db:
    type: postgres
  myapp-redis:
    type: redis

apps:
  myapp:
    # Image-based deploy (mutually exclusive with git push)
    image: myorg/myapp:latest

    # Domains
    domains:
      - myapp.example.com
      - example.com

    # Port mappings (scheme:host_port: container_port)
    ports:
      http:80: "3000"
      https:443: "3000"

    # Linked backing services
    links:
      postgres: myapp-db
      redis: myapp-redis

    # Persistent storage (host:container)
    storage:
      - /mnt/data/uploads:/app/uploads

    # Docker options by phase
    docker_options:
      build:
        - --build-arg NODE_ENV=production
      deploy:
        - --gpus all
      run:
        - --cap-add=SYS_ADMIN

    # Let's Encrypt SSL
    letsencrypt: true

    # Process scaling
    scale:
      web: 1
      worker: 2

    # Network attachments
    network:
      attach_post_create: shared
      attach_post_deploy: my-network

    # Nginx properties
    nginx:
      properties:
        client-max-body-size: 100m

    # Proxy config
    proxy:
      enabled: true

    # Mail service link
    mail: default

    # Builder selection
    builder:
      selected: dockerfile

    # Buildpacks (for herokuish/pack builds)
    buildpacks:
      - https://github.com/heroku/heroku-buildpack-nodejs.git

    # Deploy checks
    checks:
      disabled:
        - _all_

    # App locking
    locked: false

    # Maintenance mode
    maintenance: false

# Mail services
mail_services:
  default:
    provider: ""

# Auth directories (dokku-auth plugin)
auth_directories:
  ldap:
    provider: LDAP

# Auth frontends (dokku-auth plugin)
auth_frontends:
  authelia:
    provider: Authelia SSO
```

Only declare what you want to manage. Omitted fields are left untouched on the server.

## CLI Commands

### inspect

Dump live server state as a dokkufile:

```bash
# Single app (safe for git - no secrets)
dokku dokkufile:inspect myapp

# Single app with env vars (contains secrets!)
dokku dokkufile:inspect myapp --include-env

# All apps
dokku dokkufile:inspect --global

# Full server backup (all apps + secrets)
dokku dokkufile:inspect --global --include-env

# JSON output
dokku dokkufile:inspect myapp --format json
```

Inspect automatically:
- Filters internal dokku env vars (`DOKKU_*`, `GIT_REV`)
- Removes default values (checks, scheduler, process restart policy)
- Cleans docker options that duplicate link/storage config
- Extracts `dokku.mail.*` network attachments into `mail` field
- Includes linked services, mail services, and auth config in the output

### plan

Preview changes without applying:

```bash
dokku dokkufile:plan dokkufile.yml
```

### apply

Apply changes to converge live state:

```bash
dokku dokkufile:apply dokkufile.yml

# Dry run (print commands without executing)
dokku dokkufile:apply dokkufile.yml --dry-run
```

### validate

Check a dokkufile without connecting to the server:

```bash
dokku dokkufile:validate dokkufile.yml
```

### version

```bash
dokku dokkufile:version
```

## Environment Variables

Env vars are **omitted by default** from `inspect` output because they typically contain secrets (API keys, database passwords, etc.). Use `--include-env` to include them.

When a dokkufile has no `env` section, plan/apply leave the app's env vars untouched. Only add `env` to your dokkufile if you want to manage env vars declaratively.

## Git Push Workflow

The recommended workflow:

1. Bootstrap: `dokku dokkufile:inspect myapp > dokkufile.yml`
2. Commit `dokkufile.yml` to your app repo
3. Make infrastructure changes by editing `dokkufile.yml`
4. `git push` applies config changes automatically before build

The `post-extract` hook runs after code extraction but before the build, so build-time settings (docker build args, buildpacks, builder) take effect.

## Import from docker-compose

Generate a dokkufile from an existing `docker-compose.yml`:

```bash
dokku dokkufile:import -f docker-compose.yml
```

## Development

### Build & Test

```bash
CGO_ENABLED=0 go build ./...
CGO_ENABLED=0 go test ./...
```

### Integration Tests

```bash
cd tests/integration
./run.sh
```

Requires a running dokku server with postgres and redis plugins.

## License

MIT
