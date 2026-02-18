package compose

import (
	"strings"
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
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

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
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

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
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

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
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

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
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

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
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

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
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

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
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

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
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

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
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

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
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

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
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

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
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

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
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

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
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

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
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

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
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

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
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

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
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

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
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

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
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

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
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

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
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

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
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

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
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

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
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

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
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

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
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

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
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

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
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

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

// --- New field mapping tests ---

func TestAppWithExtraHosts(t *testing.T) {
	input := `
version: "3"
services:
  web:
    image: myapp:latest
    extra_hosts:
      - "host1:10.0.0.1"
      - "host2:10.0.0.2"
`
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

	app := df.Apps["web"]
	expected := []string{"--add-host=host1:10.0.0.1", "--add-host=host2:10.0.0.2"}
	if len(app.DockerOptions.Deploy) != 2 {
		t.Fatalf("expected 2 docker options, got %v", app.DockerOptions.Deploy)
	}
	for i, opt := range expected {
		if app.DockerOptions.Deploy[i] != opt {
			t.Errorf("expected %q, got %q", opt, app.DockerOptions.Deploy[i])
		}
	}
}

func TestAppWithTmpfsString(t *testing.T) {
	input := `
version: "3"
services:
  web:
    image: myapp:latest
    tmpfs: /run
`
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

	app := df.Apps["web"]
	if len(app.DockerOptions.Deploy) != 1 || app.DockerOptions.Deploy[0] != "--tmpfs=/run" {
		t.Errorf("expected docker option --tmpfs=/run, got %v", app.DockerOptions.Deploy)
	}
}

func TestAppWithTmpfsList(t *testing.T) {
	input := `
version: "3"
services:
  web:
    image: myapp:latest
    tmpfs:
      - /run
      - /tmp
`
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

	app := df.Apps["web"]
	if len(app.DockerOptions.Deploy) != 2 {
		t.Fatalf("expected 2 docker options, got %v", app.DockerOptions.Deploy)
	}
	if app.DockerOptions.Deploy[0] != "--tmpfs=/run" {
		t.Errorf("expected --tmpfs=/run, got %q", app.DockerOptions.Deploy[0])
	}
	if app.DockerOptions.Deploy[1] != "--tmpfs=/tmp" {
		t.Errorf("expected --tmpfs=/tmp, got %q", app.DockerOptions.Deploy[1])
	}
}

func TestAppWithSysctlsMap(t *testing.T) {
	input := `
version: "3"
services:
  web:
    image: myapp:latest
    sysctls:
      net.core.somaxconn: "1024"
`
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

	app := df.Apps["web"]
	if len(app.DockerOptions.Deploy) != 1 || app.DockerOptions.Deploy[0] != "--sysctl=net.core.somaxconn=1024" {
		t.Errorf("expected docker option --sysctl=net.core.somaxconn=1024, got %v", app.DockerOptions.Deploy)
	}
}

func TestAppWithSysctlsList(t *testing.T) {
	input := `
version: "3"
services:
  web:
    image: myapp:latest
    sysctls:
      - "net.core.somaxconn=1024"
`
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

	app := df.Apps["web"]
	if len(app.DockerOptions.Deploy) != 1 || app.DockerOptions.Deploy[0] != "--sysctl=net.core.somaxconn=1024" {
		t.Errorf("expected docker option --sysctl=net.core.somaxconn=1024, got %v", app.DockerOptions.Deploy)
	}
}

func TestAppWithShmSize(t *testing.T) {
	input := `
version: "3"
services:
  web:
    image: myapp:latest
    shm_size: 256m
`
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

	app := df.Apps["web"]
	if len(app.DockerOptions.Deploy) != 1 || app.DockerOptions.Deploy[0] != "--shm-size=256m" {
		t.Errorf("expected docker option --shm-size=256m, got %v", app.DockerOptions.Deploy)
	}
}

func TestAppWithUser(t *testing.T) {
	input := `
version: "3"
services:
  web:
    image: myapp:latest
    user: "1000:1000"
`
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

	app := df.Apps["web"]
	if len(app.DockerOptions.Deploy) != 1 || app.DockerOptions.Deploy[0] != "--user=1000:1000" {
		t.Errorf("expected docker option --user=1000:1000, got %v", app.DockerOptions.Deploy)
	}
}

func TestAppWithStopGracePeriod(t *testing.T) {
	input := `
version: "3"
services:
  web:
    image: myapp:latest
    stop_grace_period: 30s
`
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

	app := df.Apps["web"]
	if len(app.DockerOptions.Deploy) != 1 || app.DockerOptions.Deploy[0] != "--stop-timeout=30" {
		t.Errorf("expected docker option --stop-timeout=30, got %v", app.DockerOptions.Deploy)
	}
}

func TestAppWithLabelsMap(t *testing.T) {
	input := `
version: "3"
services:
  web:
    image: myapp:latest
    labels:
      com.example.description: "My app"
`
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

	app := df.Apps["web"]
	if len(app.DockerOptions.Deploy) != 1 || app.DockerOptions.Deploy[0] != "--label=com.example.description=My app" {
		t.Errorf("expected docker option --label=com.example.description=My app, got %v", app.DockerOptions.Deploy)
	}
}

func TestAppWithLabelsList(t *testing.T) {
	input := `
version: "3"
services:
  web:
    image: myapp:latest
    labels:
      - "com.example.description=My app"
`
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

	app := df.Apps["web"]
	if len(app.DockerOptions.Deploy) != 1 || app.DockerOptions.Deploy[0] != "--label=com.example.description=My app" {
		t.Errorf("expected docker option --label=com.example.description=My app, got %v", app.DockerOptions.Deploy)
	}
}

func TestAppWithPrivileged(t *testing.T) {
	input := `
version: "3"
services:
  web:
    image: myapp:latest
    privileged: true
`
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

	app := df.Apps["web"]
	if len(app.DockerOptions.Deploy) != 1 || app.DockerOptions.Deploy[0] != "--privileged" {
		t.Errorf("expected docker option --privileged, got %v", app.DockerOptions.Deploy)
	}
}

func TestAppWithDns(t *testing.T) {
	input := `
version: "3"
services:
  web:
    image: myapp:latest
    dns:
      - 8.8.8.8
      - 8.8.4.4
`
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

	app := df.Apps["web"]
	if len(app.DockerOptions.Deploy) != 2 {
		t.Fatalf("expected 2 docker options, got %v", app.DockerOptions.Deploy)
	}
	if app.DockerOptions.Deploy[0] != "--dns=8.8.8.8" {
		t.Errorf("expected --dns=8.8.8.8, got %q", app.DockerOptions.Deploy[0])
	}
	if app.DockerOptions.Deploy[1] != "--dns=8.8.4.4" {
		t.Errorf("expected --dns=8.8.4.4, got %q", app.DockerOptions.Deploy[1])
	}
}

func TestAppWithDnsSearch(t *testing.T) {
	input := `
version: "3"
services:
  web:
    image: myapp:latest
    dns_search:
      - example.com
      - local.test
`
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	df := result.Dokkufile

	app := df.Apps["web"]
	if len(app.DockerOptions.Deploy) != 2 {
		t.Fatalf("expected 2 docker options, got %v", app.DockerOptions.Deploy)
	}
	if app.DockerOptions.Deploy[0] != "--dns-search=example.com" {
		t.Errorf("expected --dns-search=example.com, got %q", app.DockerOptions.Deploy[0])
	}
	if app.DockerOptions.Deploy[1] != "--dns-search=local.test" {
		t.Errorf("expected --dns-search=local.test, got %q", app.DockerOptions.Deploy[1])
	}
}

// --- Warning tests ---

func TestWarningsForSkippedFields(t *testing.T) {
	input := `
version: "3"
services:
  web:
    image: myapp:latest
    command: node server.js
    env_file: .env
`
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Warnings) != 2 {
		t.Fatalf("expected 2 warnings, got %d: %v", len(result.Warnings), result.Warnings)
	}

	foundCommand := false
	foundEnvFile := false
	for _, w := range result.Warnings {
		if strings.Contains(w, `"command"`) && strings.Contains(w, "Procfile") {
			foundCommand = true
		}
		if strings.Contains(w, `"env_file"`) && strings.Contains(w, "cannot resolve") {
			foundEnvFile = true
		}
	}
	if !foundCommand {
		t.Errorf("expected warning about command field, got %v", result.Warnings)
	}
	if !foundEnvFile {
		t.Errorf("expected warning about env_file field, got %v", result.Warnings)
	}
}

func TestWarningsForUnknownFields(t *testing.T) {
	input := `
version: "3"
services:
  web:
    image: myapp:latest
    foo_bar: something
`
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", len(result.Warnings), result.Warnings)
	}
	if !strings.Contains(result.Warnings[0], `"foo_bar"`) || !strings.Contains(result.Warnings[0], "unrecognized") {
		t.Errorf("expected unrecognized warning for foo_bar, got %q", result.Warnings[0])
	}
}

func TestNoWarningsForHandledFields(t *testing.T) {
	input := `
version: "3"
services:
  web:
    image: myapp:latest
    ports:
      - "8080:80"
    environment:
      FOO: bar
`
	result, err := ImportCompose([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(result.Warnings) != 0 {
		t.Errorf("expected no warnings, got %v", result.Warnings)
	}
}
