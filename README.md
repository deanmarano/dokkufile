# Dokkufile

Infrastructure as Code for [Dokku](https://dokku.com). Define your server's desired state in a declarative YAML file and let the CLI converge it.

## Schema

```yaml
version: "1"

services:
  nextcloud-db:
    type: postgres
  nextcloud-redis:
    type: redis

apps:
  nextcloud:
    image: nextcloud:stable
    domains:
      - nextcloud.example.com
    ports:
      http: "80:80"
    env:
      NEXTCLOUD_ADMIN_USER: admin
    secrets:
      - NEXTCLOUD_ADMIN_PASSWORD
    links:
      postgres: nextcloud-db
      redis: nextcloud-redis
    storage:
      - /var/lib/dokku/data/storage/nextcloud:/var/www/html
    docker_options:
      deploy:
        - "--memory=2g"
    letsencrypt: true
    auth:
      integration: ldap
      group: nextcloud-users
    dns:
      zone: example.com
    mail: default
    healthcheck:
      path: /status.php
      timeout: 60
    scale:
      web: 1
```

Supports YAML (`.yml`/`.yaml`) and JSON (`.json`). Defaults to `Dokkufile.yml` in the current directory.

## CLI

```
dokkufile plan  [-f Dokkufile.yml]       # diff desired vs live, print plan
dokkufile apply [-f Dokkufile.yml]       # execute plan
dokkufile inspect [--format yaml|json]   # dump live state as Dokkufile
dokkufile import [-f docker-compose.yml] # scaffold Dokkufile from compose
```

## Development

### Prerequisites

- Go 1.21+
- Node.js 20+ (for E2E tests)

### Build & Test

```bash
go build ./...
CGO_ENABLED=0 go test ./...
```

### E2E Tests

```bash
cd tests/e2e
npm install
npx playwright test  # requires a running dokku server
```

### Import from docker-compose

```bash
dokkufile import -f docker-compose.yml > Dokkufile.yml
```

## How It Works

1. **Plan** — Reads the Dokkufile and queries the live dokku server. Computes a diff (new apps, removed apps, changed configuration).
2. **Apply** — Executes the plan by shelling out to `dokku` commands (create apps, set config, link services, etc).
3. **Inspect** — Reads the live server state and outputs it as a Dokkufile, useful for bootstrapping.
4. **Import** — Parses a `docker-compose.yml` and generates a Dokkufile scaffold, detecting backing services (postgres, redis, etc).

## License

MIT
