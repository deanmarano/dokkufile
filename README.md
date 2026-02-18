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
    # --- Deployment source (pick one) ---
    image: myorg/myapp:latest             # deploy from Docker image

    git:                                   # OR deploy via git
      repo: https://github.com/org/app
      branch: main
      keep_git_dir: false

    # --- Domains ---
    domains:
      - myapp.example.com
      - example.com

    # --- Port mappings (scheme:host_port: container_port) ---
    ports:
      http:80: "3000"
      https:443: "3000"

    # --- Environment variables ---
    env:
      NODE_ENV: production
      DATABASE_URL: postgres://...

    # --- Secrets (pulled from host env at apply time) ---
    secrets:
      - SECRET_KEY
      - AWS_ACCESS_KEY_ID

    # --- Linked backing services ---
    links:
      postgres: mydb
      redis: mycache

    # --- Persistent storage (host:container) ---
    storage:
      - /mnt/data/uploads:/app/uploads

    # --- Docker options by phase ---
    docker_options:
      build:
        - --build-arg NODE_ENV=production
      deploy:
        - --gpus all
        - --cap-add=NET_ADMIN
      run:
        - --cap-add=SYS_ADMIN

    # --- SSL ---
    letsencrypt: true                      # auto-provision via Let's Encrypt

    ssl:                                   # OR provide custom cert/key files
      cert_file: /path/to/cert.pem
      key_file: /path/to/key.pem

    # --- Process scaling ---
    scale:
      web: 2
      worker: 1

    # --- Resource limits/reservations per process type ---
    resources:
      web:
        limits:
          cpu: "1.0"
          memory: 512M
          memory_swap: 1G
          nvidia_gpu: "1"
        reservations:
          cpu: "0.5"
          memory: 256M

    # --- Health checks per process type ---
    healthchecks:
      web:
        - path: /health
          port: 3000
          timeout: 10
          attempts: 3
          wait: 30
          initial_delay: 15
          content: ok
        - command: curl -f http://localhost/health

    # --- Cron jobs ---
    cron:
      - command: bundle exec rake cleanup
        schedule: "0 2 * * *"

    # --- Network ---
    network:
      initial_network: my-network
      attach_post_create: shared
      attach_post_deploy: my-network
      bind_all_interfaces: false
      static_web_listener: ""
      tld: ""

    # --- Nginx ---
    nginx:
      hsts: true
      hsts_include_subdomains: true
      hsts_max_age: 31536000
      hsts_preload: false
      properties:
        client-max-body-size: 100m
        proxy-read-timeout: 120s
        proxy-buffer-size: 8k

    # --- Nginx custom template (sigil format) ---
    nginx_template: /path/to/nginx.conf.sigil

    # --- Proxy ---
    proxy:
      enabled: true
      type: nginx                          # nginx, caddy, haproxy, traefik
      caddy:                               # proxy-type-specific properties
        key: value
      haproxy:
        key: value
      traefik:
        key: value

    # --- Builder ---
    builder:
      selected: dockerfile                 # dockerfile, herokuish, pack, nixpacks
      build_dir: ""
      dockerfile_path: Dockerfile.prod
      pack_projecttoml_path: ""
      nixpacks_toml_path: ""
      herokuish_allowed: ""

    # --- Buildpacks (for herokuish/pack builds) ---
    buildpacks:
      - https://github.com/heroku/heroku-buildpack-nodejs.git
      - https://github.com/heroku/heroku-buildpack-ruby.git

    # --- Registry ---
    registry:
      server: registry.example.com
      image_repo: myorg/myapp
      push_on_release: true
      push_extra_tags: latest

    # --- Zero-downtime deploy checks ---
    checks:
      disabled:
        - _all_
      skipped:
        - worker
      wait_to_retire: 30

    # --- Process management ---
    process:
      restart_policy: always               # always, on-failure, on-failure:N, unless-stopped
      procfile_path: Procfile

    # --- Log management ---
    logs:
      max_size: 50m
      vector_image: timberio/vector:latest
      vector_sink: "https://logs.example.com"
      app_label_alias: my-app

    # --- Scheduler ---
    scheduler:
      selected: docker-local
      docker_local_init_process: "true"
      docker_local_parallel_schedule_count: "1"

    # --- Deployment scripts ---
    scripts:
      predeploy: bundle exec rake db:migrate
      postdeploy: bundle exec rake cache:clear

    # --- Auth (dokku-auth plugin) ---
    auth:
      directory: ldap
      protected: authelia

    # --- Mail service link ---
    mail: default

    # --- Maintenance mode ---
    maintenance: false

    # --- App locking ---
    locked: false
```

### Global Config

Server-wide defaults applied via `--global` flag. These serve as defaults for all apps.

```yaml
global:
  domains:
    - dokku.example.com
  nginx:
    properties:
      client-max-body-size: 50m
  proxy:
    type: nginx
  network:
    initial_network: bridge
  builder:
    selected: herokuish
  registry:
    server: registry.example.com
  logs:
    max_size: 100m
  scheduler:
    selected: docker-local
```

### Mail Services

```yaml
mail_services:
  default:
    provider: ""
    config:
      key: value
```

### Auth Directories

```yaml
auth_directories:
  ldap:
    provider: LDAP
    config:
      server: ldap://ldap.example.com
```

### Auth Frontends

```yaml
auth_frontends:
  authelia:
    provider: Authelia SSO
    directory: ldap
    protected_apps:
      - myapp
    oidc_enabled: true
    oidc_clients:
      - id: myapp-client
        secret: supersecret
        redirect_uri: https://myapp.example.com/callback
```

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

# Scope to a single app (and its linked services/plugins)
dokku dokkufile:plan dokkufile.yml --app myapp

# JSON output (for CI/scripting)
dokku dokkufile:plan dokkufile.yml --format json
```

Exits with code **2** when drift is detected — useful in CI to fail on configuration drift:

```bash
dokku dokkufile:plan dokkufile.yml || echo "Drift detected!"
```

### apply

Apply changes to converge live state:

```bash
dokku dokkufile:apply dokkufile.yml

# Scope to a single app (and its linked services/plugins)
dokku dokkufile:apply dokkufile.yml --app myapp

# Dry run (print commands without executing)
dokku dokkufile:apply dokkufile.yml --dry-run
```

### validate

Check a dokkufile offline without connecting to the server:

```bash
dokku dokkufile:validate dokkufile.yml
```

Validates:
- YAML/JSON syntax
- `image` and `git.repo` are mutually exclusive
- `letsencrypt` and `ssl` are mutually exclusive
- SSL requires both `cert_file` and `key_file`
- Service link types match declared services
- Mail and auth references point to declared resources

### import

Convert a `docker-compose.yml` to dokkufile format:

```bash
dokku dokkufile:import -f docker-compose.yml
```

Automatically detects backing services (postgres, redis, etc.) from image names and converts them to dokkufile services with links.

Prints warnings to stderr for:
- **Skipped fields** with reasons (e.g., `command` → "use a Procfile instead")
- **Unrecognized fields** that were ignored

Supported compose fields: `image`, `ports`, `environment`, `volumes`, `depends_on`, `healthcheck`, `deploy` (replicas, resources), `restart`, `build`, `cap_add`, `cap_drop`, `networks`, `logging`, `entrypoint`, `extra_hosts`, `tmpfs`, `sysctls`, `shm_size`, `user`, `stop_grace_period`, `labels`, `privileged`, `dns`, `dns_search`.

### version

```bash
dokku dokkufile:version
```

## Environment Variables

Env vars are **omitted by default** from `inspect` output because they typically contain secrets (API keys, database passwords, etc.). Use `--include-env` to include them.

When a dokkufile has no `env` section, plan/apply leave the app's env vars untouched. Only add `env` to your dokkufile if you want to manage env vars declaratively.

### Secrets

The `secrets` field lets you reference environment variables by name. At apply time, values are pulled from the host's environment — so secrets never appear in the YAML file:

```yaml
apps:
  myapp:
    secrets:
      - DATABASE_URL
      - STRIPE_SECRET_KEY
```

## Git Push Workflow

The recommended workflow:

1. Bootstrap: `dokku dokkufile:inspect myapp > dokkufile.yml`
2. Commit `dokkufile.yml` to your app repo
3. Make infrastructure changes by editing `dokkufile.yml`
4. `git push` applies config changes automatically before build

The `post-extract` hook runs after code extraction but before the build, so build-time settings (docker build args, buildpacks, builder) take effect.

## Standalone CLI Usage

You can also use dokkufile as a standalone binary (without the Dokku plugin):

```bash
# Build from source
CGO_ENABLED=0 go build -o dokkufile .

# Use directly on a Dokku server
./dokkufile inspect myapp
./dokkufile plan dokkufile.yml
./dokkufile apply dokkufile.yml
./dokkufile validate dokkufile.yml
./dokkufile import -f docker-compose.yml
```

## Supported Dokku Versions

Tested against Dokku 0.34.9, 0.35.20, 0.36.11, and 0.37.6.

## Development

### Build & Test

```bash
CGO_ENABLED=0 go build ./...
CGO_ENABLED=0 go test ./...
```

### Integration Tests

```bash
cd tests/integration
DOKKU_VERSION=0.37.6 ./run.sh
```

Requires Docker and Go. Spins up a Dokku container and runs end-to-end tests.

## License

MIT
