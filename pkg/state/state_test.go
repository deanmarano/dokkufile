package state

import (
	"fmt"
	"testing"

	"github.com/deanmarano/dokkufile/pkg/schema"
)

// FakeRunner returns canned output for known command combinations.
type FakeRunner struct {
	Commands map[string]FakeResult
}

type FakeResult struct {
	Output string
	Err    error
}

func (f *FakeRunner) Run(args ...string) (string, error) {
	key := fmt.Sprintf("%v", args)
	if r, ok := f.Commands[key]; ok {
		return r.Output, r.Err
	}
	return "", fmt.Errorf("unexpected command: %v", args)
}

// --- Parser unit tests ---

func TestParseExportLines(t *testing.T) {
	input := `export DATABASE_URL='postgres://localhost/mydb'
export SECRET_KEY='s3cret'
export EMPTY=''
`
	got := parseExportLines(input)
	want := map[string]string{
		"DATABASE_URL": "postgres://localhost/mydb",
		"SECRET_KEY":   "s3cret",
		"EMPTY":        "",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("parseExportLines[%q] = %q, want %q", k, got[k], v)
		}
	}
	if len(got) != len(want) {
		t.Errorf("parseExportLines returned %d keys, want %d", len(got), len(want))
	}
}

func TestParseExportLinesEmpty(t *testing.T) {
	got := parseExportLines("")
	if len(got) != 0 {
		t.Errorf("parseExportLines(\"\") returned %d keys, want 0", len(got))
	}
}

func TestParseScaleOutput(t *testing.T) {
	input := `-----> Scaling for myapp
       Proctype  Count
       web       1
       worker    2
`
	got := parseScaleOutput(input)
	if got["web"] != 1 {
		t.Errorf("scale[web] = %d, want 1", got["web"])
	}
	if got["worker"] != 2 {
		t.Errorf("scale[worker] = %d, want 2", got["worker"])
	}
	if len(got) != 2 {
		t.Errorf("parseScaleOutput returned %d entries, want 2", len(got))
	}
}

func TestParseScaleOutputEmpty(t *testing.T) {
	got := parseScaleOutput("")
	if len(got) != 0 {
		t.Errorf("parseScaleOutput(\"\") returned %d entries, want 0", len(got))
	}
}

func TestParsePortsList(t *testing.T) {
	input := `-----> Port map for myapp
       scheme  host-port  container-port
       http    80         5000
       https   443        5000
`
	got := parsePortsList(input)
	want := map[string]string{
		"http:80":   "5000",
		"https:443": "5000",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("ports[%q] = %q, want %q", k, got[k], v)
		}
	}
	if len(got) != len(want) {
		t.Errorf("parsePortsList returned %d entries, want %d", len(got), len(want))
	}
}

func TestSplitCommaList(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"/host:/container, /data:/data", []string{"/host:/container", "/data:/data"}},
		{"single", []string{"single"}},
		{"", nil},
		{"  a , b , c  ", []string{"a", "b", "c"}},
	}
	for _, tc := range tests {
		got := splitCommaList(tc.input)
		if len(got) != len(tc.want) {
			t.Errorf("splitCommaList(%q) len = %d, want %d", tc.input, len(got), len(tc.want))
			continue
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Errorf("splitCommaList(%q)[%d] = %q, want %q", tc.input, i, got[i], tc.want[i])
			}
		}
	}
}

func TestParseReportField(t *testing.T) {
	input := `=====> myapp
       Domains app vhosts:      example.com www.example.com
       Domains global vhosts:   global.example.com
`
	got := parseReportField(input, "Domains app vhosts")
	want := "example.com www.example.com"
	if got != want {
		t.Errorf("parseReportField = %q, want %q", got, want)
	}
}

func TestParseReportFieldMissing(t *testing.T) {
	got := parseReportField("no match here", "Missing field")
	if got != "" {
		t.Errorf("parseReportField for missing field = %q, want empty", got)
	}
}

func TestParseAppsList(t *testing.T) {
	input := `=====> My Apps
myapp
otherapp
thirdapp
`
	got := parseAppsList(input)
	want := []string{"myapp", "otherapp", "thirdapp"}
	if len(got) != len(want) {
		t.Fatalf("parseAppsList len = %d, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("parseAppsList[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestParseServiceList(t *testing.T) {
	input := `NAME        VERSION  STATUS
mydb        15       running
cache       7        running
`
	got := parseServiceList(input)
	want := []string{"mydb", "cache"}
	if len(got) != len(want) {
		t.Fatalf("parseServiceList len = %d, want %d", len(got), len(want))
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("parseServiceList[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

// --- Full integration test with FakeRunner ---

func TestDokkuReaderRead(t *testing.T) {
	fake := &FakeRunner{
		Commands: map[string]FakeResult{
			// apps:list
			fmt.Sprintf("%v", []string{"apps:list"}): {
				Output: "=====> My Apps\nmyapp\n",
			},
			// git:report for image
			fmt.Sprintf("%v", []string{"git:report", "myapp", "--git-source-image"}): {
				Output: "docker.io/library/nginx:latest\n",
			},
			// domains:report
			fmt.Sprintf("%v", []string{"domains:report", "myapp", "--domains-app-vhosts"}): {
				Output: "example.com www.example.com\n",
			},
			// ports:list
			fmt.Sprintf("%v", []string{"ports:list", "myapp"}): {
				Output: "-----> Port map for myapp\n       scheme  host-port  container-port\n       http    80         5000\n       https   443        5000\n",
			},
			// config:export
			fmt.Sprintf("%v", []string{"config:export", "myapp"}): {
				Output: "export DATABASE_URL='postgres://localhost/mydb'\nexport SECRET_KEY='s3cret'\n",
			},
			// storage:report
			fmt.Sprintf("%v", []string{"storage:report", "myapp"}): {
				Output: "=====> myapp\n       Storage build mounts:   \n       Storage deploy mounts:  /host/data:/container/data, /host/logs:/container/logs\n       Storage run mounts:     \n",
			},
			// docker-options:report
			fmt.Sprintf("%v", []string{"docker-options:report", "myapp"}): {
				Output: "=====> myapp\n       Docker options build:   \n       Docker options deploy:  --restart=always\n       Docker options run:     --cap-add=NET_ADMIN, -e FOO=bar\n",
			},
			// ps:scale
			fmt.Sprintf("%v", []string{"ps:scale", "myapp"}): {
				Output: "-----> Scaling for myapp\n       Proctype  Count\n       web       2\n       worker    1\n",
			},
			// letsencrypt:active
			fmt.Sprintf("%v", []string{"letsencrypt:active", "myapp"}): {
				Output: "",
				Err:    nil, // exit 0 = active
			},
			// service lists
			fmt.Sprintf("%v", []string{"postgres:list"}): {
				Output: "NAME        VERSION  STATUS\nmydb        15       running\n",
			},
			fmt.Sprintf("%v", []string{"redis:list"}): {
				Output: "NAME        VERSION  STATUS\nmycache     7        running\n",
			},
			fmt.Sprintf("%v", []string{"mysql:list"}): {
				Output: "",
				Err:    fmt.Errorf("plugin not installed"),
			},
			fmt.Sprintf("%v", []string{"mariadb:list"}): {
				Output: "",
				Err:    fmt.Errorf("plugin not installed"),
			},
			fmt.Sprintf("%v", []string{"mongo:list"}): {
				Output: "",
				Err:    fmt.Errorf("plugin not installed"),
			},
			// linked checks
			fmt.Sprintf("%v", []string{"postgres:linked", "mydb", "myapp"}): {
				Output: "",
				Err:    nil, // linked
			},
			fmt.Sprintf("%v", []string{"redis:linked", "mycache", "myapp"}): {
				Output: "",
				Err:    nil, // linked
			},
		},
	}

	reader := &DokkuReader{Runner: fake}
	df, err := reader.Read()
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}

	// Check version
	if df.Version != "1" {
		t.Errorf("Version = %q, want %q", df.Version, "1")
	}

	// Check apps
	app, ok := df.Apps["myapp"]
	if !ok {
		t.Fatal("expected app 'myapp' to exist")
	}

	if app.Image != "docker.io/library/nginx:latest" {
		t.Errorf("Image = %q, want %q", app.Image, "docker.io/library/nginx:latest")
	}

	if len(app.Domains) != 2 || app.Domains[0] != "example.com" || app.Domains[1] != "www.example.com" {
		t.Errorf("Domains = %v, want [example.com www.example.com]", app.Domains)
	}

	if app.Ports["http:80"] != "5000" || app.Ports["https:443"] != "5000" {
		t.Errorf("Ports = %v, unexpected", app.Ports)
	}

	if app.Env["DATABASE_URL"] != "postgres://localhost/mydb" {
		t.Errorf("Env[DATABASE_URL] = %q", app.Env["DATABASE_URL"])
	}

	if len(app.Storage) != 2 || app.Storage[0] != "/host/data:/container/data" {
		t.Errorf("Storage = %v, unexpected", app.Storage)
	}

	if len(app.DockerOptions.Deploy) != 1 || app.DockerOptions.Deploy[0] != "--restart=always" {
		t.Errorf("DockerOptions.Deploy = %v", app.DockerOptions.Deploy)
	}
	if len(app.DockerOptions.Run) != 2 || app.DockerOptions.Run[0] != "--cap-add=NET_ADMIN" {
		t.Errorf("DockerOptions.Run = %v", app.DockerOptions.Run)
	}

	if app.Scale["web"] != 2 || app.Scale["worker"] != 1 {
		t.Errorf("Scale = %v", app.Scale)
	}

	if !app.LetsEncrypt {
		t.Error("LetsEncrypt should be true")
	}

	if app.Links["postgres"] != "mydb" || app.Links["redis"] != "mycache" {
		t.Errorf("Links = %v", app.Links)
	}

	// Check services
	if df.Services["mydb"].Type != "postgres" {
		t.Errorf("Service mydb type = %q, want postgres", df.Services["mydb"].Type)
	}
	if df.Services["mycache"].Type != "redis" {
		t.Errorf("Service mycache type = %q, want redis", df.Services["mycache"].Type)
	}
}

func TestDokkuReaderNoApps(t *testing.T) {
	fake := &FakeRunner{
		Commands: map[string]FakeResult{
			fmt.Sprintf("%v", []string{"apps:list"}): {
				Output: "=====> My Apps\n",
			},
			fmt.Sprintf("%v", []string{"postgres:list"}): {
				Output: "", Err: fmt.Errorf("not installed"),
			},
			fmt.Sprintf("%v", []string{"redis:list"}): {
				Output: "", Err: fmt.Errorf("not installed"),
			},
			fmt.Sprintf("%v", []string{"mysql:list"}): {
				Output: "", Err: fmt.Errorf("not installed"),
			},
			fmt.Sprintf("%v", []string{"mariadb:list"}): {
				Output: "", Err: fmt.Errorf("not installed"),
			},
			fmt.Sprintf("%v", []string{"mongo:list"}): {
				Output: "", Err: fmt.Errorf("not installed"),
			},
		},
	}
	reader := &DokkuReader{Runner: fake}
	df, err := reader.Read()
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	if len(df.Apps) != 0 {
		t.Errorf("expected 0 apps, got %d", len(df.Apps))
	}
	if len(df.Services) != 0 {
		t.Errorf("expected 0 services, got %d", len(df.Services))
	}
}

func TestDokkuReaderLetsEncryptInactive(t *testing.T) {
	fake := &FakeRunner{
		Commands: map[string]FakeResult{
			fmt.Sprintf("%v", []string{"apps:list"}): {
				Output: "=====> My Apps\nweb\n",
			},
			fmt.Sprintf("%v", []string{"git:report", "web", "--git-source-image"}):     {Output: ""},
			fmt.Sprintf("%v", []string{"domains:report", "web", "--domains-app-vhosts"}): {Output: ""},
			fmt.Sprintf("%v", []string{"ports:list", "web"}):                             {Output: ""},
			fmt.Sprintf("%v", []string{"config:export", "web"}):                          {Output: ""},
			fmt.Sprintf("%v", []string{"storage:report", "web"}):                         {Output: ""},
			fmt.Sprintf("%v", []string{"docker-options:report", "web"}):                  {Output: ""},
			fmt.Sprintf("%v", []string{"ps:scale", "web"}):                               {Output: ""},
			fmt.Sprintf("%v", []string{"letsencrypt:active", "web"}): {
				Output: "",
				Err:    fmt.Errorf("exit status 1"), // not active
			},
			fmt.Sprintf("%v", []string{"postgres:list"}): {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"redis:list"}):    {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mysql:list"}):    {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mariadb:list"}):  {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mongo:list"}):    {Output: "", Err: fmt.Errorf("not installed")},
		},
	}
	reader := &DokkuReader{Runner: fake}
	df, err := reader.Read()
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	app := df.Apps["web"]
	if app.LetsEncrypt {
		t.Error("LetsEncrypt should be false")
	}
}

func TestDokkuReaderUnlinkedService(t *testing.T) {
	fake := &FakeRunner{
		Commands: map[string]FakeResult{
			fmt.Sprintf("%v", []string{"apps:list"}): {
				Output: "=====> My Apps\nweb\n",
			},
			fmt.Sprintf("%v", []string{"git:report", "web", "--git-source-image"}):     {Output: ""},
			fmt.Sprintf("%v", []string{"domains:report", "web", "--domains-app-vhosts"}): {Output: ""},
			fmt.Sprintf("%v", []string{"ports:list", "web"}):                             {Output: ""},
			fmt.Sprintf("%v", []string{"config:export", "web"}):                          {Output: ""},
			fmt.Sprintf("%v", []string{"storage:report", "web"}):                         {Output: ""},
			fmt.Sprintf("%v", []string{"docker-options:report", "web"}):                  {Output: ""},
			fmt.Sprintf("%v", []string{"ps:scale", "web"}):                               {Output: ""},
			fmt.Sprintf("%v", []string{"letsencrypt:active", "web"}):                     {Output: "", Err: fmt.Errorf("exit status 1")},
			fmt.Sprintf("%v", []string{"postgres:list"}): {
				Output: "NAME        VERSION  STATUS\nmydb        15       running\n",
			},
			fmt.Sprintf("%v", []string{"postgres:linked", "mydb", "web"}): {
				Output: "",
				Err:    fmt.Errorf("not linked"), // not linked
			},
			fmt.Sprintf("%v", []string{"redis:list"}):   {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mysql:list"}):    {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mariadb:list"}):  {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mongo:list"}):    {Output: "", Err: fmt.Errorf("not installed")},
		},
	}
	reader := &DokkuReader{Runner: fake}
	df, err := reader.Read()
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	app := df.Apps["web"]
	if len(app.Links) != 0 {
		t.Errorf("expected 0 links, got %v", app.Links)
	}
	// But the service should still exist
	if df.Services["mydb"].Type != "postgres" {
		t.Errorf("Service mydb should exist with type postgres")
	}
}

// Verify that the Reader interface is satisfied.
var _ Reader = (*DokkuReader)(nil)
// Verify the unused import is used
var _ = schema.Dokkufile{}
