package state

import (
	"fmt"
	"os"
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

// Verify that the Reader interface is satisfied.
var _ Reader = (*DokkuReader)(nil)
// Verify the unused import is used
var _ = schema.Dokkufile{}
