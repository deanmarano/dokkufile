package compose

import (
	"testing"
)

func TestDetectPostgresService(t *testing.T) {
	input := `
version: "3"
services:
  db:
    image: postgres:15
  app:
    image: myapp:latest
    depends_on:
      - db
`
	df, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	svc, ok := df.Services["db"]
	if !ok {
		t.Fatal("expected service 'db' to be detected")
	}
	if svc.Type != "postgres" {
		t.Errorf("expected type postgres, got %q", svc.Type)
	}

	app, ok := df.Apps["app"]
	if !ok {
		t.Fatal("expected app 'app' to exist")
	}
	if app.Image != "myapp:latest" {
		t.Errorf("expected image myapp:latest, got %q", app.Image)
	}
}

func TestDetectRedisService(t *testing.T) {
	input := `
version: "3"
services:
  cache:
    image: redis:7-alpine
`
	df, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	svc, ok := df.Services["cache"]
	if !ok {
		t.Fatal("expected service 'cache' to be detected")
	}
	if svc.Type != "redis" {
		t.Errorf("expected type redis, got %q", svc.Type)
	}
}

func TestDetectMySQLService(t *testing.T) {
	input := `
version: "3"
services:
  db:
    image: mysql:8
`
	df, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	svc, ok := df.Services["db"]
	if !ok {
		t.Fatal("expected service 'db' to be detected")
	}
	if svc.Type != "mysql" {
		t.Errorf("expected type mysql, got %q", svc.Type)
	}
}

func TestDetectMariaDBService(t *testing.T) {
	input := `
version: "3"
services:
  db:
    image: mariadb:10
`
	df, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	svc, ok := df.Services["db"]
	if !ok {
		t.Fatal("expected service 'db' to be detected")
	}
	if svc.Type != "mariadb" {
		t.Errorf("expected type mariadb, got %q", svc.Type)
	}
}

func TestDetectMongoService(t *testing.T) {
	input := `
version: "3"
services:
  db:
    image: mongo:6
`
	df, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	svc, ok := df.Services["db"]
	if !ok {
		t.Fatal("expected service 'db' to be detected")
	}
	if svc.Type != "mongo" {
		t.Errorf("expected type mongo, got %q", svc.Type)
	}
}

func TestAppWithPorts(t *testing.T) {
	input := `
version: "3"
services:
  web:
    image: nginx:latest
    ports:
      - "8080:80"
`
	df, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	app, ok := df.Apps["web"]
	if !ok {
		t.Fatal("expected app 'web' to exist")
	}
	if app.Ports["http:8080"] != "80" {
		t.Errorf("expected port http:8080 -> 80, got %v", app.Ports)
	}
}

func TestAppWithEnvironment(t *testing.T) {
	input := `
version: "3"
services:
  web:
    image: nginx:latest
    environment:
      FOO: bar
      BAZ: qux
`
	df, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	app, ok := df.Apps["web"]
	if !ok {
		t.Fatal("expected app 'web' to exist")
	}
	if app.Env["FOO"] != "bar" {
		t.Errorf("expected FOO=bar, got %q", app.Env["FOO"])
	}
	if app.Env["BAZ"] != "qux" {
		t.Errorf("expected BAZ=qux, got %q", app.Env["BAZ"])
	}
}

func TestAppWithVolumes(t *testing.T) {
	input := `
version: "3"
services:
  web:
    image: nginx:latest
    volumes:
      - ./data:/var/www/html
`
	df, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	app, ok := df.Apps["web"]
	if !ok {
		t.Fatal("expected app 'web' to exist")
	}
	if len(app.Storage) != 1 || app.Storage[0] != "./data:/var/www/html" {
		t.Errorf("unexpected storage: %v", app.Storage)
	}
}

func TestAppWithHTTPSPort(t *testing.T) {
	input := `
version: "3"
services:
  web:
    image: nginx:latest
    ports:
      - "443:8443"
`
	df, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	app := df.Apps["web"]
	if app.Ports["https:443"] != "8443" {
		t.Errorf("expected https:443 -> 8443, got %v", app.Ports)
	}
}

func TestAppWithMultiplePorts(t *testing.T) {
	input := `
version: "3"
services:
  web:
    image: nginx:latest
    ports:
      - "80:8080"
      - "443:8443"
`
	df, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	app := df.Apps["web"]
	if app.Ports["http:80"] != "8080" {
		t.Errorf("expected http:80 -> 8080, got %v", app.Ports)
	}
	if app.Ports["https:443"] != "8443" {
		t.Errorf("expected https:443 -> 8443, got %v", app.Ports)
	}
}

func TestDetectClickhouseService(t *testing.T) {
	input := `
version: "3"
services:
  analytics:
    image: clickhouse/clickhouse-server:latest
`
	df, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	svc, ok := df.Services["analytics"]
	if !ok {
		t.Fatal("expected service 'analytics' to be detected")
	}
	if svc.Type != "clickhouse" {
		t.Errorf("expected type clickhouse, got %q", svc.Type)
	}
}

func TestDetectNatsService(t *testing.T) {
	input := `
version: "3"
services:
  mq:
    image: nats:2.10
`
	df, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	svc, ok := df.Services["mq"]
	if !ok {
		t.Fatal("expected service 'mq' to be detected")
	}
	if svc.Type != "nats" {
		t.Errorf("expected type nats, got %q", svc.Type)
	}
}

func TestAppWithHealthcheck(t *testing.T) {
	input := `
version: "3"
services:
  web:
    image: myapp:latest
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost/health"]
      interval: 30s
      timeout: 10s
      retries: 3
`
	df, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	app := df.Apps["web"]
	if app.Healthchecks == nil {
		t.Fatal("expected healthchecks to be set")
	}
	hcs := app.Healthchecks["web"]
	if len(hcs) != 1 {
		t.Fatalf("expected 1 healthcheck, got %d", len(hcs))
	}
	if hcs[0].Command != "curl -f http://localhost/health" {
		t.Errorf("expected healthcheck command, got %q", hcs[0].Command)
	}
	if hcs[0].Attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", hcs[0].Attempts)
	}
	if hcs[0].Timeout != 10 {
		t.Errorf("expected 10s timeout, got %d", hcs[0].Timeout)
	}
	if hcs[0].Wait != 30 {
		t.Errorf("expected 30s wait, got %d", hcs[0].Wait)
	}
}

func TestAppWithDeployReplicas(t *testing.T) {
	input := `
version: "3"
services:
  web:
    image: myapp:latest
    deploy:
      replicas: 3
`
	df, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	app := df.Apps["web"]
	if app.Scale["web"] != 3 {
		t.Errorf("expected scale web=3, got %v", app.Scale)
	}
}

func TestAppWithDeployResources(t *testing.T) {
	input := `
version: "3"
services:
  web:
    image: myapp:latest
    deploy:
      resources:
        limits:
          cpus: "0.5"
          memory: 512M
        reservations:
          cpus: "0.25"
          memory: 256M
`
	df, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	app := df.Apps["web"]
	if app.Resources == nil {
		t.Fatal("expected resources to be set")
	}
	rc := app.Resources["web"]
	if rc.Limits.CPU != "0.5" {
		t.Errorf("expected limits cpu=0.5, got %q", rc.Limits.CPU)
	}
	if rc.Limits.Memory != "512M" {
		t.Errorf("expected limits memory=512M, got %q", rc.Limits.Memory)
	}
	if rc.Reservations.CPU != "0.25" {
		t.Errorf("expected reservations cpu=0.25, got %q", rc.Reservations.CPU)
	}
	if rc.Reservations.Memory != "256M" {
		t.Errorf("expected reservations memory=256M, got %q", rc.Reservations.Memory)
	}
}

func TestAppLinksToService(t *testing.T) {
	input := `
version: "3"
services:
  db:
    image: postgres:15
  app:
    image: myapp:latest
    depends_on:
      - db
`
	df, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	app, ok := df.Apps["app"]
	if !ok {
		t.Fatal("expected app 'app' to exist")
	}
	if app.Links["postgres"] != "db" {
		t.Errorf("expected link postgres->db, got %v", app.Links)
	}
}

func TestAppWithRestartPolicy(t *testing.T) {
	input := `
version: "3"
services:
  web:
    image: myapp:latest
    restart: always
`
	df, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	app := df.Apps["web"]
	if app.Process == nil {
		t.Fatal("expected process config to be set")
	}
	if app.Process.RestartPolicy != "always" {
		t.Errorf("expected restart policy 'always', got %q", app.Process.RestartPolicy)
	}
}

func TestAppWithBuildDockerfile(t *testing.T) {
	input := `
version: "3"
services:
  web:
    build:
      context: .
      dockerfile: Dockerfile.prod
`
	df, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	app := df.Apps["web"]
	if app.Builder == nil {
		t.Fatal("expected builder config to be set")
	}
	if app.Builder.DockerfilePath != "Dockerfile.prod" {
		t.Errorf("expected dockerfile path 'Dockerfile.prod', got %q", app.Builder.DockerfilePath)
	}
	if app.Builder.Selected != "dockerfile" {
		t.Errorf("expected builder selected 'dockerfile', got %q", app.Builder.Selected)
	}
}

func TestAppWithCapAdd(t *testing.T) {
	input := `
version: "3"
services:
  web:
    image: myapp:latest
    cap_add:
      - NET_ADMIN
`
	df, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	app := df.Apps["web"]
	if len(app.DockerOptions.Deploy) != 1 || app.DockerOptions.Deploy[0] != "--cap-add=NET_ADMIN" {
		t.Errorf("expected docker option --cap-add=NET_ADMIN, got %v", app.DockerOptions.Deploy)
	}
}

func TestAppWithCapDrop(t *testing.T) {
	input := `
version: "3"
services:
  web:
    image: myapp:latest
    cap_drop:
      - SYS_ADMIN
`
	df, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	app := df.Apps["web"]
	if len(app.DockerOptions.Deploy) != 1 || app.DockerOptions.Deploy[0] != "--cap-drop=SYS_ADMIN" {
		t.Errorf("expected docker option --cap-drop=SYS_ADMIN, got %v", app.DockerOptions.Deploy)
	}
}

func TestAppWithNetwork(t *testing.T) {
	input := `
version: "3"
services:
  web:
    image: myapp:latest
    networks:
      - mynet
`
	df, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	app := df.Apps["web"]
	if app.Network == nil {
		t.Fatal("expected network config to be set")
	}
	if app.Network.InitialNetwork != "mynet" {
		t.Errorf("expected initial network 'mynet', got %q", app.Network.InitialNetwork)
	}
}

func TestAppWithLoggingMaxSize(t *testing.T) {
	input := `
version: "3"
services:
  web:
    image: myapp:latest
    logging:
      options:
        max-size: 50m
`
	df, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	app := df.Apps["web"]
	if app.Logs == nil {
		t.Fatal("expected logs config to be set")
	}
	if app.Logs.MaxSize != "50m" {
		t.Errorf("expected max size '50m', got %q", app.Logs.MaxSize)
	}
}

func TestAppWithHealthcheckStartPeriod(t *testing.T) {
	input := `
version: "3"
services:
  web:
    image: myapp:latest
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 15s
`
	df, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	app := df.Apps["web"]
	hcs := app.Healthchecks["web"]
	if len(hcs) != 1 {
		t.Fatalf("expected 1 healthcheck, got %d", len(hcs))
	}
	if hcs[0].InitialDelay != 15 {
		t.Errorf("expected initial delay 15, got %d", hcs[0].InitialDelay)
	}
}

func TestAppWithDependsOnMap(t *testing.T) {
	input := `
version: "3"
services:
  db:
    image: postgres:15
  app:
    image: myapp:latest
    depends_on:
      db:
        condition: service_healthy
`
	df, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	app, ok := df.Apps["app"]
	if !ok {
		t.Fatal("expected app 'app' to exist")
	}
	if app.Links["postgres"] != "db" {
		t.Errorf("expected link postgres->db, got %v", app.Links)
	}
}

func TestAppWithBuildNoImage(t *testing.T) {
	input := `
version: "3"
services:
  web:
    build:
      context: .
      dockerfile: Dockerfile
`
	df, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, ok := df.Apps["web"]
	if !ok {
		t.Fatal("expected app 'web' to exist")
	}
}

func TestServiceImageVersion(t *testing.T) {
	input := `
version: "3"
services:
  db:
    image: postgres:15
  app:
    image: myapp:latest
`
	df, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	svc, ok := df.Services["db"]
	if !ok {
		t.Fatal("expected service 'db' to be detected")
	}
	if svc.ImageVersion != "15" {
		t.Errorf("expected image version '15', got %q", svc.ImageVersion)
	}
}

func TestServiceImageVersionAlpineSuffix(t *testing.T) {
	input := `
version: "3"
services:
  cache:
    image: redis:7-alpine
`
	df, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	svc := df.Services["cache"]
	if svc.ImageVersion != "7" {
		t.Errorf("expected image version '7', got %q", svc.ImageVersion)
	}
}

func TestServiceImageVersionLatestIgnored(t *testing.T) {
	input := `
version: "3"
services:
  db:
    image: postgres:latest
`
	df, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	svc := df.Services["db"]
	if svc.ImageVersion != "" {
		t.Errorf("expected empty image version for 'latest', got %q", svc.ImageVersion)
	}
}

func TestServiceImageVersionNoTag(t *testing.T) {
	input := `
version: "3"
services:
  db:
    image: postgres
`
	df, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	svc := df.Services["db"]
	if svc.ImageVersion != "" {
		t.Errorf("expected empty image version for no tag, got %q", svc.ImageVersion)
	}
}

func TestAutoAddPlugins(t *testing.T) {
	input := `
version: "3"
services:
  db:
    image: postgres:15
  cache:
    image: redis:7
  app:
    image: myapp:latest
`
	df, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if df.Plugins == nil {
		t.Fatal("expected plugins to be set")
	}
	pgPlugin, ok := df.Plugins["postgres"]
	if !ok {
		t.Fatal("expected postgres plugin entry")
	}
	if pgPlugin.URL != "https://github.com/dokku/dokku-postgres.git" {
		t.Errorf("unexpected postgres plugin URL: %q", pgPlugin.URL)
	}
	redisPlugin, ok := df.Plugins["redis"]
	if !ok {
		t.Fatal("expected redis plugin entry")
	}
	if redisPlugin.URL != "https://github.com/dokku/dokku-redis.git" {
		t.Errorf("unexpected redis plugin URL: %q", redisPlugin.URL)
	}
}
