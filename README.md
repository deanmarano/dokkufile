# dokkufile

Infrastructure as Code for [Dokku](https://dokku.com). Define your app's desired state in a declarative YAML file and let dokkufile converge it.

## Prerequisites

- [Dokku](https://dokku.com) 0.34+

## Installation

```bash
dokku plugin:install https://github.com/deanmarano/dokkufile.git dokkufile
```

To update:

```bash
dokku plugin:update dokkufile
```

## Quick Start

```bash
# Generate a dokkufile from your existing app
dokku dokkufile:inspect myapp > dokkufile.yml

# Commit dokkufile.yml to your app's git repo
# On the next git push, dokkufile validates and applies your config automatically

# Preview changes without applying
dokku dokkufile:plan dokkufile.yml

# Apply changes manually
dokku dokkufile:apply dokkufile.yml
```

## Commands

| Command | Description |
|---|---|
| `dokkufile:inspect <app> [--include-env] [--global]` | Dump live server state as a dokkufile |
| `dokkufile:plan <file> [--app <app>] [--format json]` | Preview changes without applying |
| `dokkufile:apply <file> [--app <app>] [--dry-run] [--no-checkpoint] [--clear-checkpoint]` | Apply changes to converge live state |
| `dokkufile:validate <file>` | Check a dokkufile offline |
| `dokkufile:import -f <compose-file>` | Convert docker-compose.yml to dokkufile |
| `dokkufile:version` | Show version |

## How It Works

Each app gets its own `dokkufile.yml` in its git repo. When you `git push` to dokku, a `post-extract` hook:

1. Detects `dokkufile.yml` in the repo
2. Validates the schema
3. Diffs desired state against live server state
4. Applies any changes (domains, ports, storage, etc.)
5. Build proceeds with the updated config

Build-time config like docker options, buildpacks, and builder selection are applied before the build, so they take effect immediately. If validation or apply fails, the deploy is aborted.

## Schema Reference

Only declare what you want to manage. Omitted fields are left untouched on the server.

```yaml
version: "1"
```

### Services

Backing services (databases, caches, etc.) backed by [dokku service plugins](https://dokku.com/docs/community/plugins/). 15 types supported: postgres, redis, mysql, mariadb, mongo, memcached, rabbitmq, elasticsearch, clickhouse, couchdb, meilisearch, nats, rethinkdb, solr, typesense.

```yaml
services:
  mydb:
    type: postgres
    image_version: "15"       # optional: pin service image version
  mycache:
    type: redis
```

### Plugins

Dokku plugins to install. Referenced service types auto-add their plugin, but you can also declare plugins explicitly.

```yaml
plugins:
  postgres:
    url: https://github.com/dokku/dokku-postgres.git
    committish: v1.0.0        # optional: pin to a tag/branch/commit
```

### Apps

```yaml
apps:
  myapp:
    # Deployment source (pick one)
    image: myorg/myapp:latest
    git:
      repo: https://github.com/org/app
      branch: main

    # Domains and ports
    domains:
      - myapp.example.com
    ports:
      http:80: "3000"
      https:443: "3000"

    # Environment variables
    env:
      NODE_ENV: production
    secrets:                       # config keys managed directly in Dokku
      - SECRET_KEY

    # Linked services
    links:
      postgres: mydb
      redis: mycache

    # Storage
    storage:
      - /mnt/data/uploads:/app/uploads

    # Docker options by phase
    docker_options:
      build:
        - --build-arg NODE_ENV=production
      deploy:
        - --gpus all

    # SSL
    letsencrypt: true
    # OR custom cert:
    ssl:
      cert_file: /path/to/cert.pem
      key_file: /path/to/key.pem

    # Scaling and resources
    scale:
      web: 2
      worker: 1
    resources:
      web:
        limits:
          cpu: "1.0"
          memory: 512M

    # Health checks
    healthchecks:
      web:
        - path: /health
          port: 3000
          timeout: 10

    # Cron jobs
    cron:
      - command: bundle exec rake cleanup
        schedule: "0 2 * * *"

    # Nginx
    nginx:
      hsts: true
      properties:
        client-max-body-size: 100m
    nginx_template: /path/to/nginx.conf.sigil

    # Proxy
    proxy:
      enabled: true
      type: nginx

    # Builder
    builder:
      selected: dockerfile
      dockerfile_path: Dockerfile.prod
    buildpacks:
      - https://github.com/heroku/heroku-buildpack-nodejs.git

    # Network
    network:
      initial_network: my-network
      attach_post_deploy: my-network

    # Registry
    registry:
      server: registry.example.com
      image_repo: myorg/myapp
      push_on_release: true

    # Process management
    process:
      restart_policy: always
    checks:
      wait_to_retire: 30

    # Deployment scripts
    scripts:
      predeploy: bundle exec rake db:migrate
      postdeploy: bundle exec rake cache:clear

    # SSO (dokku-sso plugin)
    auth:
      directory: ldap
      protected: authelia

    # Mail service link
    mail: default

    # Maintenance and locking
    maintenance: false
    locked: false
```

### Global Config

Server-wide defaults applied via `--global` flag:

```yaml
global:
  domains:
    - dokku.example.com
  nginx:
    properties:
      client-max-body-size: 50m
  proxy:
    type: nginx
```

## Checkpoint and Resume

If an apply fails partway through, dokkufile saves a `.dokkufile-checkpoint.json` file recording which steps completed. On the next apply, completed steps are skipped and execution resumes where it left off.

- Checkpoints are validated against the Dokkufile's content hash — if you change the Dokkufile, the stale checkpoint is discarded and a full apply runs
- Use `--no-checkpoint` to disable checkpointing
- Use `--clear-checkpoint` to delete a checkpoint manually
- On a successful apply, the checkpoint file is automatically cleaned up

## Environment Variables

Env vars are omitted by default from `inspect` output because they typically contain secrets. Use `--include-env` to include them.

The `secrets` field lists config keys that are managed directly in Dokku (via `dokku config:set`). Apply never reads or writes their values — it only excludes them from env drift detection so they aren't flagged or unset. Set them once with `dokku config:set myapp SECRET_KEY=value` and dokkufile will leave them alone:

```yaml
apps:
  myapp:
    secrets:
      - DATABASE_URL
      - STRIPE_SECRET_KEY
```

## Supported Dokku Versions

Tested against Dokku 0.34.9, 0.35.20, 0.36.11, and 0.37.6.

## Development

```bash
CGO_ENABLED=0 go build ./...
CGO_ENABLED=0 go test ./...

# Integration tests (requires Docker and Go)
cd tests/integration
DOKKU_VERSION=0.37.6 ./run.sh
```

## License

MIT
