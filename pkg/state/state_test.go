package state

import (
	"fmt"
	"os"
	"strings"
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

func TestDokkuReaderGitConfig(t *testing.T) {
	fake := &FakeRunner{
		Commands: map[string]FakeResult{
			fmt.Sprintf("%v", []string{"apps:list"}): {Output: "=====> My Apps\nweb\n"},
			fmt.Sprintf("%v", []string{"git:report", "web", "--git-source-image"}): {Output: ""},
			fmt.Sprintf("%v", []string{"git:report", "web"}): {
				Output: "=====> web\n       Git deploy branch:     main\n       Git keep git dir:      true\n",
			},
			fmt.Sprintf("%v", []string{"domains:report", "web", "--domains-app-vhosts"}): {Output: ""},
			fmt.Sprintf("%v", []string{"ports:list", "web"}):            {Output: ""},
			fmt.Sprintf("%v", []string{"config:export", "web"}):         {Output: ""},
			fmt.Sprintf("%v", []string{"storage:report", "web"}):        {Output: ""},
			fmt.Sprintf("%v", []string{"docker-options:report", "web"}): {Output: ""},
			fmt.Sprintf("%v", []string{"ps:scale", "web"}):              {Output: ""},
			fmt.Sprintf("%v", []string{"letsencrypt:active", "web"}):    {Output: "", Err: fmt.Errorf("exit status 1")},
			fmt.Sprintf("%v", []string{"postgres:list"}):                {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"redis:list"}):                   {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mysql:list"}):                   {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mariadb:list"}):                 {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mongo:list"}):                   {Output: "", Err: fmt.Errorf("not installed")},
		},
	}
	reader := &DokkuReader{Runner: fake}
	df, err := reader.Read()
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	app := df.Apps["web"]
	if app.Git == nil {
		t.Fatal("expected Git config to be set")
	}
	if app.Git.Branch != "main" {
		t.Errorf("Git.Branch = %q, want %q", app.Git.Branch, "main")
	}
	if !app.Git.KeepGitDir {
		t.Error("Git.KeepGitDir should be true")
	}
}

func TestDokkuReaderNetworkConfig(t *testing.T) {
	fake := &FakeRunner{
		Commands: map[string]FakeResult{
			fmt.Sprintf("%v", []string{"apps:list"}): {Output: "=====> My Apps\nweb\n"},
			fmt.Sprintf("%v", []string{"git:report", "web", "--git-source-image"}): {Output: ""},
			fmt.Sprintf("%v", []string{"domains:report", "web", "--domains-app-vhosts"}): {Output: ""},
			fmt.Sprintf("%v", []string{"ports:list", "web"}):            {Output: ""},
			fmt.Sprintf("%v", []string{"config:export", "web"}):         {Output: ""},
			fmt.Sprintf("%v", []string{"storage:report", "web"}):        {Output: ""},
			fmt.Sprintf("%v", []string{"docker-options:report", "web"}): {Output: ""},
			fmt.Sprintf("%v", []string{"ps:scale", "web"}):              {Output: ""},
			fmt.Sprintf("%v", []string{"network:report", "web"}): {
				Output: "=====> web\n       Network attach post create:  mynet\n       Network attach post deploy:  \n       Network bind all interfaces:  true\n       Network initial network:     \n       Network static web listener:  \n       Network tld:                  \n",
			},
			fmt.Sprintf("%v", []string{"letsencrypt:active", "web"}): {Output: "", Err: fmt.Errorf("exit status 1")},
			fmt.Sprintf("%v", []string{"postgres:list"}):             {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"redis:list"}):                {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mysql:list"}):                {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mariadb:list"}):              {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mongo:list"}):                {Output: "", Err: fmt.Errorf("not installed")},
		},
	}
	reader := &DokkuReader{Runner: fake}
	df, err := reader.Read()
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	app := df.Apps["web"]
	if app.Network == nil {
		t.Fatal("expected Network config to be set")
	}
	if app.Network.AttachPostCreate != "mynet" {
		t.Errorf("Network.AttachPostCreate = %q, want %q", app.Network.AttachPostCreate, "mynet")
	}
	if !app.Network.BindAllInterfaces {
		t.Error("Network.BindAllInterfaces should be true")
	}
}

func TestDokkuReaderNginxProxyConfig(t *testing.T) {
	fake := &FakeRunner{
		Commands: map[string]FakeResult{
			fmt.Sprintf("%v", []string{"apps:list"}): {Output: "=====> My Apps\nweb\n"},
			fmt.Sprintf("%v", []string{"git:report", "web", "--git-source-image"}): {Output: ""},
			fmt.Sprintf("%v", []string{"domains:report", "web", "--domains-app-vhosts"}): {Output: ""},
			fmt.Sprintf("%v", []string{"ports:list", "web"}):            {Output: ""},
			fmt.Sprintf("%v", []string{"config:export", "web"}):         {Output: ""},
			fmt.Sprintf("%v", []string{"storage:report", "web"}):        {Output: ""},
			fmt.Sprintf("%v", []string{"docker-options:report", "web"}): {Output: ""},
			fmt.Sprintf("%v", []string{"ps:scale", "web"}):              {Output: ""},
			fmt.Sprintf("%v", []string{"nginx:report", "web"}): {
				Output: "=====> web\n       Nginx hsts:                    true\n       Nginx hsts include subdomains:  true\n       Nginx hsts max age:             31536000\n       Nginx hsts preload:             false\n",
			},
			fmt.Sprintf("%v", []string{"proxy:report", "web"}): {
				Output: "=====> web\n       Proxy enabled:  true\n       Proxy type:     nginx\n",
			},
			fmt.Sprintf("%v", []string{"letsencrypt:active", "web"}): {Output: "", Err: fmt.Errorf("exit status 1")},
			fmt.Sprintf("%v", []string{"postgres:list"}):             {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"redis:list"}):                {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mysql:list"}):                {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mariadb:list"}):              {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mongo:list"}):                {Output: "", Err: fmt.Errorf("not installed")},
		},
	}
	reader := &DokkuReader{Runner: fake}
	df, err := reader.Read()
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	app := df.Apps["web"]
	if app.Nginx == nil {
		t.Fatal("expected Nginx config to be set")
	}
	if !app.Nginx.HSTS {
		t.Error("Nginx.HSTS should be true")
	}
	if !app.Nginx.HSTSIncludeSubdomains {
		t.Error("Nginx.HSTSIncludeSubdomains should be true")
	}
	if app.Nginx.HSTSMaxAge != 31536000 {
		t.Errorf("Nginx.HSTSMaxAge = %d, want 31536000", app.Nginx.HSTSMaxAge)
	}
	if app.Proxy == nil {
		t.Fatal("expected Proxy config to be set")
	}
	if !app.Proxy.Enabled {
		t.Error("Proxy.Enabled should be true")
	}
	if app.Proxy.Type != "nginx" {
		t.Errorf("Proxy.Type = %q, want %q", app.Proxy.Type, "nginx")
	}
}

func TestDokkuReaderSSLPresent(t *testing.T) {
	fake := &FakeRunner{
		Commands: map[string]FakeResult{
			fmt.Sprintf("%v", []string{"apps:list"}): {Output: "=====> My Apps\nweb\n"},
			fmt.Sprintf("%v", []string{"git:report", "web", "--git-source-image"}): {Output: ""},
			fmt.Sprintf("%v", []string{"domains:report", "web", "--domains-app-vhosts"}): {Output: ""},
			fmt.Sprintf("%v", []string{"ports:list", "web"}):            {Output: ""},
			fmt.Sprintf("%v", []string{"config:export", "web"}):         {Output: ""},
			fmt.Sprintf("%v", []string{"storage:report", "web"}):        {Output: ""},
			fmt.Sprintf("%v", []string{"docker-options:report", "web"}): {Output: ""},
			fmt.Sprintf("%v", []string{"ps:scale", "web"}):              {Output: ""},
			fmt.Sprintf("%v", []string{"certs:report", "web"}): {
				Output: "=====> web\n       Ssl cert present:  true\n",
			},
			fmt.Sprintf("%v", []string{"letsencrypt:active", "web"}): {Output: "", Err: fmt.Errorf("exit status 1")},
			fmt.Sprintf("%v", []string{"postgres:list"}):             {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"redis:list"}):                {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mysql:list"}):                {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mariadb:list"}):              {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mongo:list"}):                {Output: "", Err: fmt.Errorf("not installed")},
		},
	}
	reader := &DokkuReader{Runner: fake}
	df, err := reader.Read()
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	app := df.Apps["web"]
	if app.SSL == nil {
		t.Fatal("expected SSL config to be set when cert is present")
	}
}

func TestParseAppJSON(t *testing.T) {
	input := `{
		"healthchecks": {
			"web": [{"path": "/health", "timeout": 10}]
		},
		"cron": [
			{"command": "rake db:backup", "schedule": "@daily"}
		]
	}`
	hc, cron, scripts := parseAppJSON(input)
	if len(hc) != 1 {
		t.Fatalf("expected 1 healthcheck process type, got %d", len(hc))
	}
	if len(hc["web"]) != 1 || hc["web"][0].Path != "/health" || hc["web"][0].Timeout != 10 {
		t.Errorf("healthcheck = %+v, unexpected", hc["web"])
	}
	if len(cron) != 1 || cron[0].Command != "rake db:backup" || cron[0].Schedule != "@daily" {
		t.Errorf("cron = %+v, unexpected", cron)
	}
	if scripts != nil {
		t.Error("expected nil scripts")
	}
}

func TestParseAppJSONEmpty(t *testing.T) {
	hc, cron, _ := parseAppJSON("{}")
	if len(hc) != 0 {
		t.Errorf("expected no healthchecks, got %d", len(hc))
	}
	if len(cron) != 0 {
		t.Errorf("expected no cron, got %d", len(cron))
	}
}

func TestParseAppJSONInvalid(t *testing.T) {
	hc, cron, _ := parseAppJSON("not json")
	if hc != nil || cron != nil {
		t.Error("expected nil for invalid JSON")
	}
}

func TestParseAppJSONWithScripts(t *testing.T) {
	input := `{
		"scripts": {
			"dokku": {
				"predeploy": "rake db:migrate",
				"postdeploy": "rake cache:clear"
			}
		}
	}`
	_, _, scripts := parseAppJSON(input)
	if scripts == nil {
		t.Fatal("expected scripts to be set")
	}
	if scripts.Predeploy != "rake db:migrate" {
		t.Errorf("Predeploy = %q, want %q", scripts.Predeploy, "rake db:migrate")
	}
	if scripts.Postdeploy != "rake cache:clear" {
		t.Errorf("Postdeploy = %q, want %q", scripts.Postdeploy, "rake cache:clear")
	}
}

func TestParseReportConfigFields(t *testing.T) {
	input := `=====> myservice
       Provider:       smtp
       Config host:    smtp.example.com
       Config port:    587
       Config user:
`
	got := parseReportConfigFields(input)
	if got["host"] != "smtp.example.com" {
		t.Errorf("Config host = %q, want smtp.example.com", got["host"])
	}
	if got["port"] != "587" {
		t.Errorf("Config port = %q, want 587", got["port"])
	}
	// Empty value should be excluded
	if _, ok := got["user"]; ok {
		t.Error("empty config values should be excluded")
	}
	if len(got) != 2 {
		t.Errorf("expected 2 config fields, got %d", len(got))
	}
}

func TestParseOIDCClients(t *testing.T) {
	input := `ID          SECRET     REDIRECT_URI
client1     secret1    https://example.com/callback
client2     secret2    https://other.com/callback
`
	got := parseOIDCClients(input)
	if len(got) != 2 {
		t.Fatalf("expected 2 OIDC clients, got %d", len(got))
	}
	if got[0].ID != "client1" || got[0].Secret != "secret1" || got[0].RedirectURI != "https://example.com/callback" {
		t.Errorf("client 0 = %+v, unexpected", got[0])
	}
	if got[1].ID != "client2" {
		t.Errorf("client 1 ID = %q, want client2", got[1].ID)
	}
}

func TestParseOIDCClientsEmpty(t *testing.T) {
	got := parseOIDCClients("ID  SECRET  REDIRECT_URI\n")
	if len(got) != 0 {
		t.Errorf("expected 0 clients, got %d", len(got))
	}
}

// FakeFileRunner returns canned file contents for testing.
type FakeFileRunner struct {
	Files map[string]FakeResult
}

func (f *FakeFileRunner) ReadFile(path string) (string, error) {
	if r, ok := f.Files[path]; ok {
		return r.Output, r.Err
	}
	return "", fmt.Errorf("file not found: %s", path)
}

func (f *FakeFileRunner) WriteFile(path string, content []byte, perm os.FileMode) error {
	return nil
}

func TestDokkuReaderNginxTemplate(t *testing.T) {
	fake := &FakeRunner{
		Commands: map[string]FakeResult{
			fmt.Sprintf("%v", []string{"apps:list"}): {Output: "=====> My Apps\nweb\n"},
			fmt.Sprintf("%v", []string{"git:report", "web", "--git-source-image"}): {Output: "nginx:latest\n"},
			fmt.Sprintf("%v", []string{"domains:report", "web", "--domains-app-vhosts"}): {Output: ""},
			fmt.Sprintf("%v", []string{"ports:list", "web"}):            {Output: ""},
			fmt.Sprintf("%v", []string{"config:export", "web"}):         {Output: ""},
			fmt.Sprintf("%v", []string{"storage:report", "web"}):        {Output: ""},
			fmt.Sprintf("%v", []string{"docker-options:report", "web"}): {Output: ""},
			fmt.Sprintf("%v", []string{"ps:scale", "web"}):              {Output: ""},
			fmt.Sprintf("%v", []string{"letsencrypt:active", "web"}):    {Output: "", Err: fmt.Errorf("exit status 1")},
			fmt.Sprintf("%v", []string{"postgres:list"}):                {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"redis:list"}):                   {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mysql:list"}):                   {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mariadb:list"}):                 {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mongo:list"}):                   {Output: "", Err: fmt.Errorf("not installed")},
		},
	}
	fileRunner := &FakeFileRunner{
		Files: map[string]FakeResult{
			"/home/dokku/web/nginx.conf.sigil": {Output: "server { listen 80; }"},
			"/home/dokku/web/app.json":         {Output: "", Err: fmt.Errorf("not found")},
		},
	}
	reader := &DokkuReader{Runner: fake, FileRunner: fileRunner}
	df, err := reader.Read()
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	app := df.Apps["web"]
	if app.NginxTemplate != "server { listen 80; }" {
		t.Errorf("NginxTemplate = %q, want %q", app.NginxTemplate, "server { listen 80; }")
	}
}

func TestDokkuReaderAppJSON(t *testing.T) {
	fake := &FakeRunner{
		Commands: map[string]FakeResult{
			fmt.Sprintf("%v", []string{"apps:list"}): {Output: "=====> My Apps\nweb\n"},
			fmt.Sprintf("%v", []string{"git:report", "web", "--git-source-image"}): {Output: ""},
			fmt.Sprintf("%v", []string{"domains:report", "web", "--domains-app-vhosts"}): {Output: ""},
			fmt.Sprintf("%v", []string{"ports:list", "web"}):            {Output: ""},
			fmt.Sprintf("%v", []string{"config:export", "web"}):         {Output: ""},
			fmt.Sprintf("%v", []string{"storage:report", "web"}):        {Output: ""},
			fmt.Sprintf("%v", []string{"docker-options:report", "web"}): {Output: ""},
			fmt.Sprintf("%v", []string{"ps:scale", "web"}):              {Output: ""},
			fmt.Sprintf("%v", []string{"letsencrypt:active", "web"}):    {Output: "", Err: fmt.Errorf("exit status 1")},
			fmt.Sprintf("%v", []string{"postgres:list"}):                {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"redis:list"}):                   {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mysql:list"}):                   {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mariadb:list"}):                 {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mongo:list"}):                   {Output: "", Err: fmt.Errorf("not installed")},
		},
	}
	fileRunner := &FakeFileRunner{
		Files: map[string]FakeResult{
			"/home/dokku/web/nginx.conf.sigil": {Output: "", Err: fmt.Errorf("not found")},
			"/home/dokku/web/app.json": {
				Output: `{"healthchecks":{"web":[{"path":"/health","timeout":5}]},"cron":[{"command":"rake cleanup","schedule":"@hourly"}]}`,
			},
		},
	}
	reader := &DokkuReader{Runner: fake, FileRunner: fileRunner}
	df, err := reader.Read()
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	app := df.Apps["web"]
	if len(app.Healthchecks) != 1 {
		t.Fatalf("expected 1 healthcheck proc type, got %d", len(app.Healthchecks))
	}
	if app.Healthchecks["web"][0].Path != "/health" {
		t.Errorf("healthcheck path = %q, want /health", app.Healthchecks["web"][0].Path)
	}
	if len(app.Cron) != 1 || app.Cron[0].Command != "rake cleanup" {
		t.Errorf("cron = %+v, unexpected", app.Cron)
	}
}

func TestDokkuReaderMailServices(t *testing.T) {
	fake := &FakeRunner{
		Commands: map[string]FakeResult{
			fmt.Sprintf("%v", []string{"apps:list"}): {Output: "=====> My Apps\n"},
			fmt.Sprintf("%v", []string{"postgres:list"}): {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"redis:list"}):    {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mysql:list"}):    {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mariadb:list"}):  {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mongo:list"}):    {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mail:list"}): {
				Output: "NAME        VERSION  STATUS\nmymail      1        running\n",
			},
			fmt.Sprintf("%v", []string{"mail:info", "mymail"}): {
				Output: "=====> mymail\n       Provider:       smtp\n       Config host:    smtp.example.com\n       Config port:    587\n",
			},
		},
	}
	reader := &DokkuReader{Runner: fake}
	df, err := reader.Read()
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	if len(df.MailServices) != 1 {
		t.Fatalf("expected 1 mail service, got %d", len(df.MailServices))
	}
	svc := df.MailServices["mymail"]
	if svc.Provider != "smtp" {
		t.Errorf("Provider = %q, want smtp", svc.Provider)
	}
	if svc.Config["host"] != "smtp.example.com" {
		t.Errorf("Config host = %q, want smtp.example.com", svc.Config["host"])
	}
}

func TestDokkuReaderAuthDirectories(t *testing.T) {
	fake := &FakeRunner{
		Commands: map[string]FakeResult{
			fmt.Sprintf("%v", []string{"apps:list"}): {Output: "=====> My Apps\n"},
			fmt.Sprintf("%v", []string{"postgres:list"}): {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"redis:list"}):    {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mysql:list"}):    {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mariadb:list"}):  {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mongo:list"}):    {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"auth:list"}): {
				Output: "NAME        VERSION  STATUS\nmydir       1        running\n",
			},
			fmt.Sprintf("%v", []string{"auth:info", "mydir"}): {
				Output: "=====> mydir\n       Provider:       ldap\n       Config url:     ldap://example.com\n",
			},
		},
	}
	reader := &DokkuReader{Runner: fake}
	df, err := reader.Read()
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	if len(df.AuthDirectories) != 1 {
		t.Fatalf("expected 1 auth directory, got %d", len(df.AuthDirectories))
	}
	dir := df.AuthDirectories["mydir"]
	if dir.Provider != "ldap" {
		t.Errorf("Provider = %q, want ldap", dir.Provider)
	}
	if dir.Config["url"] != "ldap://example.com" {
		t.Errorf("Config url = %q", dir.Config["url"])
	}
}

func TestDokkuReaderAuthFrontends(t *testing.T) {
	fake := &FakeRunner{
		Commands: map[string]FakeResult{
			fmt.Sprintf("%v", []string{"apps:list"}): {Output: "=====> My Apps\n"},
			fmt.Sprintf("%v", []string{"postgres:list"}): {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"redis:list"}):    {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mysql:list"}):    {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mariadb:list"}):  {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mongo:list"}):    {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"auth:frontend:list"}): {
				Output: "NAME        VERSION  STATUS\nmyfe        1        running\n",
			},
			fmt.Sprintf("%v", []string{"auth:frontend:info", "myfe"}): {
				Output: "=====> myfe\n       Provider:       oauth2\n       Directory:      mydir\n       Protected apps: webapp api\n",
			},
			fmt.Sprintf("%v", []string{"auth:oidc:list", "myfe"}): {
				Output: "ID          SECRET     REDIRECT_URI\nclient1     secret1    https://example.com/callback\n",
			},
		},
	}
	reader := &DokkuReader{Runner: fake}
	df, err := reader.Read()
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	if len(df.AuthFrontends) != 1 {
		t.Fatalf("expected 1 auth frontend, got %d", len(df.AuthFrontends))
	}
	fe := df.AuthFrontends["myfe"]
	if fe.Provider != "oauth2" {
		t.Errorf("Provider = %q, want oauth2", fe.Provider)
	}
	if fe.Directory != "mydir" {
		t.Errorf("Directory = %q, want mydir", fe.Directory)
	}
	if len(fe.ProtectedApps) != 2 || fe.ProtectedApps[0] != "webapp" {
		t.Errorf("ProtectedApps = %v, unexpected", fe.ProtectedApps)
	}
	if !fe.OIDCEnabled {
		t.Error("OIDCEnabled should be true")
	}
	if len(fe.OIDCClients) != 1 || fe.OIDCClients[0].ID != "client1" {
		t.Errorf("OIDCClients = %+v, unexpected", fe.OIDCClients)
	}
}

func TestParseResourceReport(t *testing.T) {
	input := `=====> myapp
       web limit cpu:              1
       web limit memory:           512m
       web reservation memory:     256m
       worker limit cpu:           2
       worker limit nvidia gpu:    1
`
	got := parseResourceReport(input)
	if len(got) != 2 {
		t.Fatalf("expected 2 process types, got %d", len(got))
	}
	if got["web"].Limits.CPU != "1" {
		t.Errorf("web limit cpu = %q, want 1", got["web"].Limits.CPU)
	}
	if got["web"].Limits.Memory != "512m" {
		t.Errorf("web limit memory = %q, want 512m", got["web"].Limits.Memory)
	}
	if got["web"].Reservations.Memory != "256m" {
		t.Errorf("web reservation memory = %q, want 256m", got["web"].Reservations.Memory)
	}
	if got["worker"].Limits.CPU != "2" {
		t.Errorf("worker limit cpu = %q, want 2", got["worker"].Limits.CPU)
	}
	if got["worker"].Limits.NvidiaGPU != "1" {
		t.Errorf("worker limit nvidia gpu = %q, want 1", got["worker"].Limits.NvidiaGPU)
	}
}

func TestParseResourceReportEmpty(t *testing.T) {
	got := parseResourceReport("")
	if len(got) != 0 {
		t.Errorf("expected 0 entries, got %d", len(got))
	}
}

func TestDokkuReaderResourcesAndChecks(t *testing.T) {
	fake := &FakeRunner{
		Commands: map[string]FakeResult{
			fmt.Sprintf("%v", []string{"apps:list"}): {Output: "=====> My Apps\nweb\n"},
			fmt.Sprintf("%v", []string{"git:report", "web", "--git-source-image"}):      {Output: ""},
			fmt.Sprintf("%v", []string{"domains:report", "web", "--domains-app-vhosts"}): {Output: ""},
			fmt.Sprintf("%v", []string{"ports:list", "web"}):            {Output: ""},
			fmt.Sprintf("%v", []string{"config:export", "web"}):         {Output: ""},
			fmt.Sprintf("%v", []string{"storage:report", "web"}):        {Output: ""},
			fmt.Sprintf("%v", []string{"docker-options:report", "web"}): {Output: ""},
			fmt.Sprintf("%v", []string{"ps:scale", "web"}):              {Output: ""},
			fmt.Sprintf("%v", []string{"resource:report", "web"}): {
				Output: "=====> web\n       web limit cpu:          2\n       web limit memory:       1024m\n       web reservation memory: 512m\n",
			},
			fmt.Sprintf("%v", []string{"checks:report", "web"}): {
				Output: "=====> web\n       Checks disabled list:   worker\n       Checks skipped list:    \n       Checks wait to retire:  30\n",
			},
			fmt.Sprintf("%v", []string{"builder:report", "web"}): {
				Output: "=====> web\n       Builder selected:   herokuish\n       Builder build dir:  src\n",
			},
			fmt.Sprintf("%v", []string{"registry:report", "web"}): {
				Output: "=====> web\n       Registry server:           registry.example.com\n       Registry image repo:        myorg/myapp\n       Registry push on release:   true\n       Registry push extra tags:   latest\n",
			},
			fmt.Sprintf("%v", []string{"maintenance:report", "web"}): {
				Output: "=====> web\n       Maintenance enabled:  true\n",
			},
			fmt.Sprintf("%v", []string{"letsencrypt:active", "web"}): {Output: "", Err: fmt.Errorf("exit status 1")},
			fmt.Sprintf("%v", []string{"postgres:list"}):             {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"redis:list"}):                {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mysql:list"}):                {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mariadb:list"}):              {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mongo:list"}):                {Output: "", Err: fmt.Errorf("not installed")},
		},
	}
	reader := &DokkuReader{Runner: fake}
	df, err := reader.Read()
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	app := df.Apps["web"]

	// Resources
	if len(app.Resources) != 1 {
		t.Fatalf("expected 1 resource proc type, got %d", len(app.Resources))
	}
	if app.Resources["web"].Limits.CPU != "2" {
		t.Errorf("resource web limit cpu = %q, want 2", app.Resources["web"].Limits.CPU)
	}
	if app.Resources["web"].Limits.Memory != "1024m" {
		t.Errorf("resource web limit memory = %q, want 1024m", app.Resources["web"].Limits.Memory)
	}
	if app.Resources["web"].Reservations.Memory != "512m" {
		t.Errorf("resource web reservation memory = %q, want 512m", app.Resources["web"].Reservations.Memory)
	}

	// Checks
	if app.Checks == nil {
		t.Fatal("expected Checks to be set")
	}
	if len(app.Checks.Disabled) != 1 || app.Checks.Disabled[0] != "worker" {
		t.Errorf("Checks.Disabled = %v, want [worker]", app.Checks.Disabled)
	}
	if app.Checks.WaitToRetire != 30 {
		t.Errorf("Checks.WaitToRetire = %d, want 30", app.Checks.WaitToRetire)
	}

	// Builder
	if app.Builder == nil {
		t.Fatal("expected Builder to be set")
	}
	if app.Builder.Selected != "herokuish" {
		t.Errorf("Builder.Selected = %q, want herokuish", app.Builder.Selected)
	}
	if app.Builder.BuildDir != "src" {
		t.Errorf("Builder.BuildDir = %q, want src", app.Builder.BuildDir)
	}

	// Registry
	if app.Registry == nil {
		t.Fatal("expected Registry to be set")
	}
	if app.Registry.Server != "registry.example.com" {
		t.Errorf("Registry.Server = %q, want registry.example.com", app.Registry.Server)
	}
	if app.Registry.ImageRepo != "myorg/myapp" {
		t.Errorf("Registry.ImageRepo = %q, want myorg/myapp", app.Registry.ImageRepo)
	}
	if !app.Registry.PushOnRelease {
		t.Error("Registry.PushOnRelease should be true")
	}
	if app.Registry.PushExtraTags != "latest" {
		t.Errorf("Registry.PushExtraTags = %q, want latest", app.Registry.PushExtraTags)
	}

	// Maintenance
	if !app.Maintenance {
		t.Error("Maintenance should be true")
	}
}

func TestParseNginxProperties(t *testing.T) {
	input := `=====> web
       Nginx hsts:                    true
       Nginx client max body size:    50m
       Nginx proxy read timeout:      120s
       Nginx proxy buffer size:       16k
       Nginx underscore in headers:   on
`
	got := parseNginxProperties(input)
	if got["client-max-body-size"] != "50m" {
		t.Errorf("client-max-body-size = %q, want 50m", got["client-max-body-size"])
	}
	if got["proxy-read-timeout"] != "120s" {
		t.Errorf("proxy-read-timeout = %q, want 120s", got["proxy-read-timeout"])
	}
	if got["proxy-buffer-size"] != "16k" {
		t.Errorf("proxy-buffer-size = %q, want 16k", got["proxy-buffer-size"])
	}
	if got["underscore-in-headers"] != "on" {
		t.Errorf("underscore-in-headers = %q, want on", got["underscore-in-headers"])
	}
}

func TestDokkuReaderProcessAndLocking(t *testing.T) {
	fake := &FakeRunner{
		Commands: map[string]FakeResult{
			fmt.Sprintf("%v", []string{"apps:list"}): {Output: "=====> My Apps\nweb\n"},
			fmt.Sprintf("%v", []string{"git:report", "web", "--git-source-image"}):      {Output: ""},
			fmt.Sprintf("%v", []string{"domains:report", "web", "--domains-app-vhosts"}): {Output: ""},
			fmt.Sprintf("%v", []string{"ports:list", "web"}):            {Output: ""},
			fmt.Sprintf("%v", []string{"config:export", "web"}):         {Output: ""},
			fmt.Sprintf("%v", []string{"storage:report", "web"}):        {Output: ""},
			fmt.Sprintf("%v", []string{"docker-options:report", "web"}): {Output: ""},
			fmt.Sprintf("%v", []string{"ps:scale", "web"}):              {Output: ""},
			fmt.Sprintf("%v", []string{"ps:report", "web"}): {
				Output: "=====> web\n       Ps restart policy:   on-failure:3\n       Ps procfile path:    Procfile.web\n",
			},
			fmt.Sprintf("%v", []string{"apps:locked", "web"}): {Output: "", Err: nil},
			fmt.Sprintf("%v", []string{"letsencrypt:active", "web"}): {Output: "", Err: fmt.Errorf("exit status 1")},
			fmt.Sprintf("%v", []string{"postgres:list"}):             {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"redis:list"}):                {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mysql:list"}):                {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mariadb:list"}):              {Output: "", Err: fmt.Errorf("not installed")},
			fmt.Sprintf("%v", []string{"mongo:list"}):                {Output: "", Err: fmt.Errorf("not installed")},
		},
	}
	reader := &DokkuReader{Runner: fake}
	df, err := reader.Read()
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}
	app := df.Apps["web"]

	// Process
	if app.Process == nil {
		t.Fatal("expected Process to be set")
	}
	if app.Process.RestartPolicy != "on-failure:3" {
		t.Errorf("Process.RestartPolicy = %q, want on-failure:3", app.Process.RestartPolicy)
	}
	if app.Process.ProcfilePath != "Procfile.web" {
		t.Errorf("Process.ProcfilePath = %q, want Procfile.web", app.Process.ProcfilePath)
	}

	// Locking
	if !app.Locked {
		t.Error("Locked should be true")
	}
}

func TestParseBuildpacksList(t *testing.T) {
	input := `=====> myapp buildpack urls
  1. https://github.com/heroku/heroku-buildpack-nodejs
  2. https://github.com/heroku/heroku-buildpack-ruby
`
	got := parseBuildpacksList(input)
	if len(got) != 2 {
		t.Fatalf("expected 2 buildpacks, got %d", len(got))
	}
	if got[0] != "https://github.com/heroku/heroku-buildpack-nodejs" {
		t.Errorf("buildpack[0] = %q", got[0])
	}
	if got[1] != "https://github.com/heroku/heroku-buildpack-ruby" {
		t.Errorf("buildpack[1] = %q", got[1])
	}
}

func TestParsePluginList(t *testing.T) {
	input := `  00_dokku-standard    0.37.6    enabled    dokku core standard plugin
  app-json             0.37.6    enabled    dokku core app-json plugin
  letsencrypt          0.23.0    enabled    Auto-renewal of SSL certs
  disabled-plugin      1.0.0     disabled   A disabled plugin
`
	got := parsePluginList(input)
	// Core plugins (with "dokku core" in description) should be filtered out
	if _, ok := got["00_dokku-standard"]; ok {
		t.Error("core plugin 00_dokku-standard should be filtered out")
	}
	if _, ok := got["app-json"]; ok {
		t.Error("core plugin app-json should be filtered out")
	}
	// Non-core enabled plugins should be included
	if _, ok := got["letsencrypt"]; !ok {
		t.Error("expected letsencrypt in plugins")
	}
	// Disabled plugins should be excluded
	if _, ok := got["disabled-plugin"]; ok {
		t.Error("disabled-plugin should not be in plugins")
	}
	if len(got) != 1 {
		t.Errorf("expected 1 plugin, got %d: %v", len(got), got)
	}
}

func TestDokkuReaderLogsAndScheduler(t *testing.T) {
	runner := &FakeRunner{
		Commands: map[string]FakeResult{
			"[apps:list]": {Output: "=====> Apps\nmyapp\n"},
			// Minimal app setup
			"[git:report myapp --git-source-image]": {Output: "nginx"},
			"[git:report myapp]":                    {Output: ""},
			"[domains:report myapp --domains-app-vhosts]": {Output: ""},
			"[ports:list myapp]":                           {Output: "", Err: fmt.Errorf("not found")},
			"[config:export myapp]":                        {Output: ""},
			"[storage:report myapp]":                       {Output: ""},
			"[docker-options:report myapp]":                {Output: ""},
			"[ps:scale myapp]":                             {Output: ""},
			"[network:report myapp]":                       {Output: ""},
			"[nginx:report myapp]":                         {Output: ""},
			"[proxy:report myapp]":                         {Output: ""},
			"[certs:report myapp]":                         {Output: ""},
			"[resource:report myapp]":                      {Output: ""},
			"[checks:report myapp]":                        {Output: ""},
			"[builder:report myapp]":                       {Output: ""},
			"[registry:report myapp]":                      {Output: ""},
			"[maintenance:report myapp]":                   {Output: ""},
			"[ps:report myapp]":                            {Output: ""},
			"[apps:locked myapp]":                          {Output: "", Err: fmt.Errorf("not locked")},
			"[letsencrypt:active myapp]":                   {Output: "", Err: fmt.Errorf("not active")},
			// Logs
			"[logs:report myapp]": {Output: `       Logs max size:           50m
       Logs vector image:       timberio/vector:latest
       Logs vector sink:        console://
       Logs app label alias:    my-alias
`},
			// Scheduler
			"[scheduler:report myapp]": {Output: `       Scheduler selected:      docker-local
`},
			"[scheduler-docker-local:report myapp]": {Output: `       Scheduler docker local init process:  true
       Scheduler docker local parallel schedule count:  3
`},
			// Buildpacks
			"[buildpacks:list myapp]": {Output: `=====> myapp buildpack urls
  1. https://github.com/heroku/heroku-buildpack-nodejs
`},
			// Builder sub-plugins
			"[builder-dockerfile:report myapp]": {Output: `       Builder dockerfile dockerfile path:  Dockerfile.prod
`},
			"[builder-pack:report myapp]":       {Output: ""},
			"[builder-nixpacks:report myapp]":   {Output: ""},
			"[builder-herokuish:report myapp]":  {Output: ""},
		},
	}

	// Add service stubs
	for _, svcType := range serviceTypes {
		key := fmt.Sprintf("[%s:list]", svcType)
		runner.Commands[key] = FakeResult{Err: fmt.Errorf("not installed")}
	}
	runner.Commands["[mail:list]"] = FakeResult{Err: fmt.Errorf("not installed")}
	runner.Commands["[auth:list]"] = FakeResult{Err: fmt.Errorf("not installed")}
	runner.Commands["[auth:frontend:list]"] = FakeResult{Err: fmt.Errorf("not installed")}
	runner.Commands["[plugin:list]"] = FakeResult{Err: fmt.Errorf("not installed")}

	reader := &DokkuReader{Runner: runner}
	df, err := reader.Read()
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}

	app := df.Apps["myapp"]

	// Logs
	if app.Logs == nil {
		t.Fatal("Logs should not be nil")
	}
	if app.Logs.MaxSize != "50m" {
		t.Errorf("Logs.MaxSize = %q, want 50m", app.Logs.MaxSize)
	}
	if app.Logs.VectorSink != "console://" {
		t.Errorf("Logs.VectorSink = %q, want console://", app.Logs.VectorSink)
	}
	if app.Logs.VectorImage != "timberio/vector:latest" {
		t.Errorf("Logs.VectorImage = %q", app.Logs.VectorImage)
	}
	if app.Logs.AppLabelAlias != "my-alias" {
		t.Errorf("Logs.AppLabelAlias = %q", app.Logs.AppLabelAlias)
	}

	// Scheduler
	if app.Scheduler == nil {
		t.Fatal("Scheduler should not be nil")
	}
	if app.Scheduler.Selected != "docker-local" {
		t.Errorf("Scheduler.Selected = %q, want docker-local", app.Scheduler.Selected)
	}
	// DockerLocalInitProcess "true" is filtered as a default
	if app.Scheduler.DockerLocalInitProcess != "" {
		t.Errorf("Scheduler.DockerLocalInitProcess = %q, want empty (filtered as default)", app.Scheduler.DockerLocalInitProcess)
	}
	if app.Scheduler.DockerLocalParallelScheduleCount != "3" {
		t.Errorf("Scheduler.DockerLocalParallelScheduleCount = %q", app.Scheduler.DockerLocalParallelScheduleCount)
	}

	// Buildpacks
	if len(app.Buildpacks) != 1 {
		t.Fatalf("expected 1 buildpack, got %d", len(app.Buildpacks))
	}
	if app.Buildpacks[0] != "https://github.com/heroku/heroku-buildpack-nodejs" {
		t.Errorf("Buildpacks[0] = %q", app.Buildpacks[0])
	}

	// Builder sub-plugin
	if app.Builder == nil {
		t.Fatal("Builder should not be nil")
	}
	if app.Builder.DockerfilePath != "Dockerfile.prod" {
		t.Errorf("Builder.DockerfilePath = %q, want Dockerfile.prod", app.Builder.DockerfilePath)
	}
}

func TestParseProxyProperties(t *testing.T) {
	input := `       Caddy tls internal:         true
       Caddy image:                caddy:2
       Caddy log level:            INFO
`
	got := parseProxyProperties(input, "Caddy", caddyPropertyNames)
	if got["tls-internal"] != "true" {
		t.Errorf("tls-internal = %q, want true", got["tls-internal"])
	}
	if got["image"] != "caddy:2" {
		t.Errorf("image = %q, want caddy:2", got["image"])
	}
	if got["log-level"] != "INFO" {
		t.Errorf("log-level = %q, want INFO", got["log-level"])
	}
}

func TestParseTraefikProperties(t *testing.T) {
	input := `       Traefik api enabled:        true
       Traefik letsencrypt email:  admin@example.com
       Traefik log level:          DEBUG
`
	got := parseProxyProperties(input, "Traefik", traefikPropertyNames)
	if got["api-enabled"] != "true" {
		t.Errorf("api-enabled = %q, want true", got["api-enabled"])
	}
	if got["letsencrypt-email"] != "admin@example.com" {
		t.Errorf("letsencrypt-email = %q", got["letsencrypt-email"])
	}
	if got["log-level"] != "DEBUG" {
		t.Errorf("log-level = %q, want DEBUG", got["log-level"])
	}
}

func TestDokkuReaderMailLink(t *testing.T) {
	fake := &FakeRunner{
		Commands: map[string]FakeResult{
			"[apps:list]":                              {Output: "=====> My Apps\nmyapp"},
			"[git:report myapp --git-source-image]":    {Output: ""},
			"[mail:list]":                              {Output: "NAME  VERSION  STATUS\nmymail  1.0  running"},
			"[mail:linked mymail myapp]":               {Output: ""},
			"[auth:list]":                              {Err: fmt.Errorf("not installed")},
		},
	}

	reader := &DokkuReader{Runner: fake}
	df, err := reader.Read()
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}

	app := df.Apps["myapp"]
	if app.Mail != "mymail" {
		t.Errorf("expected Mail=mymail, got %q", app.Mail)
	}
}

func TestDokkuReaderAuthLink(t *testing.T) {
	fake := &FakeRunner{
		Commands: map[string]FakeResult{
			"[apps:list]":                              {Output: "=====> My Apps\nmyapp"},
			"[git:report myapp --git-source-image]":    {Output: ""},
			"[mail:list]":                              {Err: fmt.Errorf("not installed")},
			"[auth:list]":                              {Output: "NAME  VERSION  STATUS\nmydir  1.0  running"},
			"[auth:linked mydir myapp]":                {Output: ""},
		},
	}

	reader := &DokkuReader{Runner: fake}
	df, err := reader.Read()
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}

	app := df.Apps["myapp"]
	if app.Auth == nil || app.Auth.Directory != "mydir" {
		t.Errorf("expected Auth.Directory=mydir, got %v", app.Auth)
	}
}

func TestCleanAppFiltersInternalEnv(t *testing.T) {
	app := &schema.App{
		Env: map[string]string{
			"DATABASE_URL":        "postgres://localhost/db",
			"DOKKU_APP_RESTORE":   "1",
			"DOKKU_APP_TYPE":      "dockerfile",
			"DOKKU_PROXY_PORT":    "80",
			"DOKKU_PROXY_SSL_PORT": "443",
			"GIT_REV":             "abc123",
			"SECRET_KEY":          "s3cret",
		},
	}
	cleanApp(app)
	if len(app.Env) != 2 {
		t.Errorf("expected 2 env vars, got %d: %v", len(app.Env), app.Env)
	}
	if app.Env["DATABASE_URL"] != "postgres://localhost/db" {
		t.Error("DATABASE_URL should be kept")
	}
	if app.Env["SECRET_KEY"] != "s3cret" {
		t.Error("SECRET_KEY should be kept")
	}
}

func TestCleanAppFiltersDefaults(t *testing.T) {
	app := &schema.App{
		Git: &schema.GitConfig{Branch: "master"},
		Checks: &schema.ChecksConfig{
			Disabled: []string{"none"},
			Skipped:  []string{"none"},
		},
		Process:   &schema.ProcessConfig{RestartPolicy: "on-failure:10"},
		Scheduler: &schema.SchedulerConfig{DockerLocalInitProcess: "true"},
	}
	cleanApp(app)
	if app.Git != nil {
		t.Error("Git should be nil (branch master is default)")
	}
	if app.Checks != nil {
		t.Error("Checks should be nil (disabled/skipped none are defaults)")
	}
	if app.Process != nil {
		t.Error("Process should be nil (on-failure:10 is default)")
	}
	if app.Scheduler != nil {
		t.Error("Scheduler should be nil (init process true is default)")
	}
}

func TestCleanAppPreservesNonDefaults(t *testing.T) {
	app := &schema.App{
		Git: &schema.GitConfig{Branch: "main"},
		Process: &schema.ProcessConfig{RestartPolicy: "always"},
		Scheduler: &schema.SchedulerConfig{
			Selected:                "docker-local",
			DockerLocalInitProcess:  "true",
		},
	}
	cleanApp(app)
	if app.Git == nil || app.Git.Branch != "main" {
		t.Error("Git branch 'main' should be preserved")
	}
	if app.Process == nil || app.Process.RestartPolicy != "always" {
		t.Error("Process restart_policy 'always' should be preserved")
	}
	if app.Scheduler == nil || app.Scheduler.Selected != "docker-local" {
		t.Error("Scheduler selected should be preserved")
	}
	// DockerLocalInitProcess should be cleared even with other fields present
	if app.Scheduler.DockerLocalInitProcess != "" {
		t.Error("Scheduler DockerLocalInitProcess 'true' should be cleared")
	}
}

func TestCleanAppCleansStorage(t *testing.T) {
	app := &schema.App{
		Storage: []string{"-v /host/path:/container/path", "-v /other:/data"},
	}
	cleanApp(app)
	if len(app.Storage) != 2 {
		t.Fatalf("expected 2 storage entries, got %d", len(app.Storage))
	}
	if app.Storage[0] != "/host/path:/container/path" {
		t.Errorf("Storage[0] = %q, want /host/path:/container/path", app.Storage[0])
	}
	if app.Storage[1] != "/other:/data" {
		t.Errorf("Storage[1] = %q, want /other:/data", app.Storage[1])
	}
}

func TestCleanAppSplitsCompoundStorage(t *testing.T) {
	app := &schema.App{
		Storage: []string{"/mnt/data/config:/etc/app -v /mnt/data/uploads:/app/uploads -v /mnt/data/logs:/var/log/app"},
	}
	cleanApp(app)
	if len(app.Storage) != 3 {
		t.Fatalf("expected 3 storage entries, got %d: %v", len(app.Storage), app.Storage)
	}
	expected := []string{
		"/mnt/data/config:/etc/app",
		"/mnt/data/uploads:/app/uploads",
		"/mnt/data/logs:/var/log/app",
	}
	for i, want := range expected {
		if app.Storage[i] != want {
			t.Errorf("Storage[%d] = %q, want %q", i, app.Storage[i], want)
		}
	}
}

func TestCleanAppCleansDockerOptions(t *testing.T) {
	app := &schema.App{
		Links: map[string]string{"postgres": "mydb"},
		Storage: []string{"-v /data:/app/data"},
		DockerOptions: schema.DockerOptions{
			Deploy: []string{
				"--link dokku.postgres.mydb:dokku-postgres-mydb --restart=on-failure:10 -v /data:/app/data --cap-add=SYS_ADMIN",
			},
			Run: []string{
				"--link dokku.postgres.mydb:dokku-postgres-mydb -v /data:/app/data",
			},
			Build: []string{
				"--link dokku.postgres.mydb:dokku-postgres-mydb",
			},
		},
	}
	cleanApp(app)
	// Deploy should only keep --cap-add=SYS_ADMIN
	if len(app.DockerOptions.Deploy) != 1 || app.DockerOptions.Deploy[0] != "--cap-add=SYS_ADMIN" {
		t.Errorf("Deploy = %v, want [--cap-add=SYS_ADMIN]", app.DockerOptions.Deploy)
	}
	// Run should be empty (all flags filtered)
	if len(app.DockerOptions.Run) != 0 {
		t.Errorf("Run = %v, want empty", app.DockerOptions.Run)
	}
	// Build should be empty (all flags filtered)
	if len(app.DockerOptions.Build) != 0 {
		t.Errorf("Build = %v, want empty", app.DockerOptions.Build)
	}
}

func TestCleanAppExtractsMailFromNetwork(t *testing.T) {
	app := schema.App{
		Network: &schema.NetworkConfig{
			AttachPostDeploy: "dokku.mail.default",
		},
	}
	cleanApp(&app)
	if app.Mail != "default" {
		t.Errorf("expected Mail=default, got %q", app.Mail)
	}
	if app.Network != nil {
		t.Errorf("expected Network to be nil after extracting mail, got %+v", app.Network)
	}
}

func TestCleanAppExtractsMailAndKeepsOtherNetworks(t *testing.T) {
	app := schema.App{
		Network: &schema.NetworkConfig{
			AttachPostCreate: "shared,dokku.mail.deanoftech",
			AttachPostDeploy: "dokku.mail.deanoftech",
		},
	}
	cleanApp(&app)
	if app.Mail != "deanoftech" {
		t.Errorf("expected Mail=deanoftech, got %q", app.Mail)
	}
	if app.Network == nil {
		t.Fatal("expected Network to remain for non-mail networks")
	}
	if app.Network.AttachPostCreate != "shared" {
		t.Errorf("expected AttachPostCreate=shared, got %q", app.Network.AttachPostCreate)
	}
	if app.Network.AttachPostDeploy != "" {
		t.Errorf("expected AttachPostDeploy to be empty, got %q", app.Network.AttachPostDeploy)
	}
}

func TestCleanAppExtractsAuthFrontendFromNetwork(t *testing.T) {
	app := schema.App{
		Network: &schema.NetworkConfig{
			AttachPostCreate: "dokku.auth.frontend.auth",
		},
	}
	cleanApp(&app)
	if app.Auth == nil || app.Auth.Protected != "auth" {
		t.Errorf("expected Auth.Protected=auth, got %+v", app.Auth)
	}
	if app.Network != nil {
		t.Errorf("expected Network to be nil, got %+v", app.Network)
	}
}

// TrackingRunner wraps a FakeRunner and records every command called.
type TrackingRunner struct {
	inner    *FakeRunner
	Called   []string // keys in the same format as FakeRunner
}

func (t *TrackingRunner) Run(args ...string) (string, error) {
	key := fmt.Sprintf("%v", args)
	t.Called = append(t.Called, key)
	return t.inner.Run(args...)
}

func (t *TrackingRunner) HasCalled(args ...string) bool {
	key := fmt.Sprintf("%v", args)
	for _, c := range t.Called {
		if c == key {
			return true
		}
	}
	return false
}

func (t *TrackingRunner) HasCalledPrefix(prefix string) bool {
	for _, c := range t.Called {
		if len(c) > 1 && strings.HasPrefix(c[1:], prefix) {
			return true
		}
	}
	return false
}

func TestReadScopedOnlyScopedApp(t *testing.T) {
	// Set up a server with two apps: "myapp" and "otherapp".
	// Scope only references "myapp".
	// Verify that only myapp's commands are called (not otherapp's).
	fake := &FakeRunner{
		Commands: map[string]FakeResult{
			// apps:list returns both apps
			fmt.Sprintf("%v", []string{"apps:list"}): {
				Output: "=====> My Apps\nmyapp\notherapp\n",
			},
			// myapp commands
			fmt.Sprintf("%v", []string{"git:report", "myapp", "--git-source-image"}): {Output: "nginx:latest\n"},
			fmt.Sprintf("%v", []string{"domains:report", "myapp", "--domains-app-vhosts"}): {Output: "example.com\n"},
			fmt.Sprintf("%v", []string{"ports:list", "myapp"}):            {Output: ""},
			fmt.Sprintf("%v", []string{"config:export", "myapp"}):         {Output: ""},
			fmt.Sprintf("%v", []string{"storage:report", "myapp"}):        {Output: ""},
			fmt.Sprintf("%v", []string{"docker-options:report", "myapp"}): {Output: ""},
			fmt.Sprintf("%v", []string{"ps:scale", "myapp"}):              {Output: ""},
			fmt.Sprintf("%v", []string{"letsencrypt:active", "myapp"}):    {Output: "", Err: fmt.Errorf("not active")},
		},
	}

	// Add service stubs — scope has no services so these should NOT be called
	for _, svcType := range serviceTypes {
		key := fmt.Sprintf("%v", []string{svcType + ":list"})
		fake.Commands[key] = FakeResult{Err: fmt.Errorf("not installed")}
	}

	tracker := &TrackingRunner{inner: fake}
	reader := &DokkuReader{Runner: tracker}

	scope := &schema.Dokkufile{
		Version:  "1",
		Apps:     map[string]schema.App{"myapp": {Image: "nginx:latest"}},
		Services: map[string]schema.Service{},
	}

	df, err := reader.ReadScoped(scope)
	if err != nil {
		t.Fatalf("ReadScoped() error: %v", err)
	}

	// myapp should be present
	if _, ok := df.Apps["myapp"]; !ok {
		t.Error("expected myapp in result")
	}
	if df.Apps["myapp"].Image != "nginx:latest" {
		t.Errorf("Image = %q, want nginx:latest", df.Apps["myapp"].Image)
	}

	// otherapp should NOT be present
	if _, ok := df.Apps["otherapp"]; ok {
		t.Error("otherapp should not be in result — it's not in scope")
	}

	// Verify no commands were called for otherapp
	for _, c := range tracker.Called {
		if strings.Contains(c, "otherapp") {
			t.Errorf("unexpected command for otherapp: %s", c)
		}
	}
}

func TestReadScopedOnlyScopedServiceTypes(t *testing.T) {
	// Scope references only postgres services, not redis.
	// Verify redis:list is never called.
	fake := &FakeRunner{
		Commands: map[string]FakeResult{
			fmt.Sprintf("%v", []string{"apps:list"}): {Output: "=====> My Apps\nmyapp\n"},
			// postgres:list — should be called
			fmt.Sprintf("%v", []string{"postgres:list"}): {
				Output: "NAME        VERSION  STATUS\nmydb        15       running\n",
			},
			fmt.Sprintf("%v", []string{"postgres:linked", "mydb", "myapp"}): {Output: ""},
			// myapp commands
			fmt.Sprintf("%v", []string{"git:report", "myapp", "--git-source-image"}): {Output: ""},
			fmt.Sprintf("%v", []string{"domains:report", "myapp", "--domains-app-vhosts"}): {Output: ""},
			fmt.Sprintf("%v", []string{"ports:list", "myapp"}):            {Output: ""},
			fmt.Sprintf("%v", []string{"config:export", "myapp"}):         {Output: ""},
			fmt.Sprintf("%v", []string{"storage:report", "myapp"}):        {Output: ""},
			fmt.Sprintf("%v", []string{"docker-options:report", "myapp"}): {Output: ""},
			fmt.Sprintf("%v", []string{"ps:scale", "myapp"}):              {Output: ""},
			fmt.Sprintf("%v", []string{"letsencrypt:active", "myapp"}):    {Output: "", Err: fmt.Errorf("not active")},
		},
	}

	tracker := &TrackingRunner{inner: fake}
	reader := &DokkuReader{Runner: tracker}

	scope := &schema.Dokkufile{
		Version:  "1",
		Apps:     map[string]schema.App{"myapp": {}},
		Services: map[string]schema.Service{"mydb": {Type: "postgres"}},
	}

	df, err := reader.ReadScoped(scope)
	if err != nil {
		t.Fatalf("ReadScoped() error: %v", err)
	}

	// postgres should be scanned
	if !tracker.HasCalled("postgres:list") {
		t.Error("expected postgres:list to be called")
	}
	if df.Services["mydb"].Type != "postgres" {
		t.Errorf("Service mydb type = %q, want postgres", df.Services["mydb"].Type)
	}

	// redis should NOT be scanned
	if tracker.HasCalled("redis:list") {
		t.Error("redis:list should not be called — not in scope")
	}

	// mysql should NOT be scanned
	if tracker.HasCalled("mysql:list") {
		t.Error("mysql:list should not be called — not in scope")
	}

	// Link should work
	app := df.Apps["myapp"]
	if app.Links["postgres"] != "mydb" {
		t.Errorf("Links = %v, expected postgres->mydb", app.Links)
	}
}

func TestReadScopedStillReadsPluginsAndGlobal(t *testing.T) {
	fake := &FakeRunner{
		Commands: map[string]FakeResult{
			fmt.Sprintf("%v", []string{"apps:list"}): {Output: "=====> My Apps\nmyapp\n"},
			// myapp commands
			fmt.Sprintf("%v", []string{"git:report", "myapp", "--git-source-image"}): {Output: ""},
			fmt.Sprintf("%v", []string{"domains:report", "myapp", "--domains-app-vhosts"}): {Output: ""},
			fmt.Sprintf("%v", []string{"ports:list", "myapp"}):            {Output: ""},
			fmt.Sprintf("%v", []string{"config:export", "myapp"}):         {Output: ""},
			fmt.Sprintf("%v", []string{"storage:report", "myapp"}):        {Output: ""},
			fmt.Sprintf("%v", []string{"docker-options:report", "myapp"}): {Output: ""},
			fmt.Sprintf("%v", []string{"ps:scale", "myapp"}):              {Output: ""},
			fmt.Sprintf("%v", []string{"letsencrypt:active", "myapp"}):    {Output: "", Err: fmt.Errorf("not active")},
			// plugin:list
			fmt.Sprintf("%v", []string{"plugin:list"}): {
				Output: "  letsencrypt          0.23.0    enabled    Auto-renewal of SSL certs\n",
			},
		},
	}

	tracker := &TrackingRunner{inner: fake}
	reader := &DokkuReader{Runner: tracker}

	scope := &schema.Dokkufile{
		Version:  "1",
		Apps:     map[string]schema.App{"myapp": {}},
		Services: map[string]schema.Service{},
	}

	df, err := reader.ReadScoped(scope)
	if err != nil {
		t.Fatalf("ReadScoped() error: %v", err)
	}

	// Plugins should be read
	if !tracker.HasCalled("plugin:list") {
		t.Error("expected plugin:list to be called")
	}
	if _, ok := df.Plugins["letsencrypt"]; !ok {
		t.Error("expected letsencrypt in plugins")
	}

	// Global config commands should be called
	if !tracker.HasCalled("domains:report", "--global") {
		t.Error("expected global domains:report to be called")
	}
}

func TestReadScopedNewAppNotOnServer(t *testing.T) {
	// An app in scope that doesn't exist on the server should not error,
	// it just won't appear in the result (it's a new app to be created).
	fake := &FakeRunner{
		Commands: map[string]FakeResult{
			fmt.Sprintf("%v", []string{"apps:list"}): {Output: "=====> My Apps\n"},
		},
	}

	tracker := &TrackingRunner{inner: fake}
	reader := &DokkuReader{Runner: tracker}

	scope := &schema.Dokkufile{
		Version:  "1",
		Apps:     map[string]schema.App{"newapp": {Image: "nginx"}},
		Services: map[string]schema.Service{},
	}

	df, err := reader.ReadScoped(scope)
	if err != nil {
		t.Fatalf("ReadScoped() error: %v", err)
	}

	if len(df.Apps) != 0 {
		t.Errorf("expected 0 apps (new app not on server), got %d", len(df.Apps))
	}
}

func TestReadScopedSkipsMailWhenNotReferenced(t *testing.T) {
	fake := &FakeRunner{
		Commands: map[string]FakeResult{
			fmt.Sprintf("%v", []string{"apps:list"}): {Output: "=====> My Apps\nmyapp\n"},
			fmt.Sprintf("%v", []string{"git:report", "myapp", "--git-source-image"}): {Output: ""},
			fmt.Sprintf("%v", []string{"domains:report", "myapp", "--domains-app-vhosts"}): {Output: ""},
			fmt.Sprintf("%v", []string{"ports:list", "myapp"}):            {Output: ""},
			fmt.Sprintf("%v", []string{"config:export", "myapp"}):         {Output: ""},
			fmt.Sprintf("%v", []string{"storage:report", "myapp"}):        {Output: ""},
			fmt.Sprintf("%v", []string{"docker-options:report", "myapp"}): {Output: ""},
			fmt.Sprintf("%v", []string{"ps:scale", "myapp"}):              {Output: ""},
			fmt.Sprintf("%v", []string{"letsencrypt:active", "myapp"}):    {Output: "", Err: fmt.Errorf("not active")},
		},
	}

	tracker := &TrackingRunner{inner: fake}
	reader := &DokkuReader{Runner: tracker}

	scope := &schema.Dokkufile{
		Version:  "1",
		Apps:     map[string]schema.App{"myapp": {}},
		Services: map[string]schema.Service{},
	}

	_, err := reader.ReadScoped(scope)
	if err != nil {
		t.Fatalf("ReadScoped() error: %v", err)
	}

	// mail:list should NOT be called when no mail in scope
	if tracker.HasCalled("mail:list") {
		t.Error("mail:list should not be called — no mail in scope")
	}
	// auth:list should NOT be called
	if tracker.HasCalled("auth:list") {
		t.Error("auth:list should not be called — no auth in scope")
	}
	// auth:frontend:list should NOT be called
	if tracker.HasCalled("auth:frontend:list") {
		t.Error("auth:frontend:list should not be called — no auth frontends in scope")
	}
}

// Verify that the Reader interface is satisfied.
var _ Reader = (*DokkuReader)(nil)
// Verify the unused import is used
var _ = schema.Dokkufile{}
