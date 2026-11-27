package apply

import (
	"reflect"
	"testing"
)

func TestEnvUpdateCommandsLeavesSecretsAlone(t *testing.T) {
	desired := map[string]string{"BASE_URL": "https://new"}
	actual := map[string]string{"BASE_URL": "https://old", "AUTHELIA_DOMAIN": "auth.example", "DATABASE_URL": "postgres://x", "STALE": "1"}
	got := envUpdateCommands("app", desired, actual, []string{"AUTHELIA_DOMAIN", "DATABASE_URL"})
	want := [][]string{
		{"config:set", "--no-restart", "app", "BASE_URL=https://new"},
		{"config:unset", "--no-restart", "app", "STALE"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestEnvUpdateCommandsWithoutSecretsUnsetsExtras(t *testing.T) {
	got := envUpdateCommands("app", map[string]string{}, map[string]string{"X": "1"}, nil)
	want := [][]string{{"config:unset", "--no-restart", "app", "X"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
