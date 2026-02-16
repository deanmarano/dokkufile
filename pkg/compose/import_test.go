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
	if app.Ports["http"] != "8080:80" {
		t.Errorf("expected port 8080:80, got %q", app.Ports["http"])
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
