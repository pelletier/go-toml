package benchmark_test

import (
	"bytes"
	"testing"

	toml "github.com/pelletier/go-toml/v2"
)

// This file contains benchmarks modeled after the most common open-source
// usages of go-toml v2:
//
//   - containerd: daemon configuration with deeply nested plugin tables and
//     quoted keys, decoded into a struct holding generic sections.
//   - Viper (and tools built on it, like golangci-lint): configuration
//     decoded into map[string]interface{}, and written back with Marshal.
//   - Hugo: many small TOML front-matter documents decoded into maps.
//   - gitleaks: large rule files made of arrays of tables with nested
//     arrays of tables.
//   - Python tooling configs (pyproject.toml) read by Go developer tools:
//     dotted keys and long string arrays.

// ---------------------------------------------------------------------------
// containerd-style daemon configuration

var containerdDoc = []byte(`
version = 2
root = "/var/lib/containerd"
state = "/run/containerd"
oom_score = -999

[grpc]
  address = "/run/containerd/containerd.sock"
  uid = 0
  gid = 0
  max_recv_message_size = 16777216
  max_send_message_size = 16777216

[ttrpc]
  address = ""
  uid = 0
  gid = 0

[debug]
  address = ""
  level = "info"

[metrics]
  address = "127.0.0.1:1338"
  grpc_histogram = false

[timeouts]
  "io.containerd.timeout.shim.cleanup" = "5s"
  "io.containerd.timeout.shim.load" = "5s"
  "io.containerd.timeout.shim.shutdown" = "3s"
  "io.containerd.timeout.task.state" = "2s"

[plugins]
  [plugins."io.containerd.gc.v1.scheduler"]
    pause_threshold = 0.02
    deletion_threshold = 0
    mutation_threshold = 100
    schedule_delay = "0s"
    startup_delay = "100ms"
  [plugins."io.containerd.grpc.v1.cri"]
    disable_tcp_service = true
    stream_server_address = "127.0.0.1"
    stream_server_port = "0"
    stream_idle_timeout = "4h0m0s"
    enable_selinux = false
    sandbox_image = "registry.k8s.io/pause:3.9"
    stats_collect_period = 10
    enable_tls_streaming = false
    max_container_log_line_size = 16384
    [plugins."io.containerd.grpc.v1.cri".containerd]
      snapshotter = "overlayfs"
      default_runtime_name = "runc"
      no_pivot = false
      [plugins."io.containerd.grpc.v1.cri".containerd.runtimes]
        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
          runtime_type = "io.containerd.runc.v2"
          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
            SystemdCgroup = true
            BinaryName = "/usr/bin/runc"
        [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.kata]
          runtime_type = "io.containerd.kata.v2"
          privileged_without_host_devices = true
    [plugins."io.containerd.grpc.v1.cri".registry]
      config_path = ""
      [plugins."io.containerd.grpc.v1.cri".registry.mirrors]
        [plugins."io.containerd.grpc.v1.cri".registry.mirrors."docker.io"]
          endpoint = ["https://registry-1.docker.io"]
        [plugins."io.containerd.grpc.v1.cri".registry.mirrors."k8s.gcr.io"]
          endpoint = ["https://registry.k8s.io", "https://k8s.gcr.io"]
    [plugins."io.containerd.grpc.v1.cri".cni]
      bin_dir = "/opt/cni/bin"
      conf_dir = "/etc/cni/net.d"
      max_conf_num = 1
  [plugins."io.containerd.internal.v1.opt"]
    path = "/opt/containerd"
  [plugins."io.containerd.runtime.v2.task"]
    platforms = ["linux/amd64"]
    sched_core = false
  [plugins."io.containerd.service.v1.diff-service"]
    default = ["walking"]

[cgroup]
  path = ""

[proxy_plugins]
  [proxy_plugins.stargz]
    type = "snapshot"
    address = "/run/containerd-stargz-grpc/containerd-stargz-grpc.sock"
`)

// containerdConfig mirrors the shape of containerd's config struct: typed
// top-level sections, with plugin configuration kept generic.
type containerdConfig struct {
	Version  int    `toml:"version"`
	Root     string `toml:"root"`
	State    string `toml:"state"`
	OOMScore int    `toml:"oom_score"`
	GRPC     struct {
		Address        string `toml:"address"`
		UID            int    `toml:"uid"`
		GID            int    `toml:"gid"`
		MaxRecvMsgSize int    `toml:"max_recv_message_size"`
		MaxSendMsgSize int    `toml:"max_send_message_size"`
	} `toml:"grpc"`
	TTRPC struct {
		Address string `toml:"address"`
		UID     int    `toml:"uid"`
		GID     int    `toml:"gid"`
	} `toml:"ttrpc"`
	Debug struct {
		Address string `toml:"address"`
		Level   string `toml:"level"`
	} `toml:"debug"`
	Metrics struct {
		Address       string `toml:"address"`
		GRPCHistogram bool   `toml:"grpc_histogram"`
	} `toml:"metrics"`
	Timeouts     map[string]string                 `toml:"timeouts"`
	Plugins      map[string]interface{}            `toml:"plugins"`
	Cgroup       struct{ Path string }             `toml:"cgroup"`
	ProxyPlugins map[string]map[string]interface{} `toml:"proxy_plugins"`
}

func BenchmarkRealWorldContainerdConfig(b *testing.B) {
	b.SetBytes(int64(len(containerdDoc)))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var cfg containerdConfig
		if err := toml.Unmarshal(containerdDoc, &cfg); err != nil {
			b.Fatal(err)
		}
	}
}

// ---------------------------------------------------------------------------
// Viper-style generic configuration handling: decode into a generic map,
// and write it back (WriteConfig).

var viperDoc = []byte(`
app_name = "service"
environment = "production"
debug = false
port = 8080
timeout = 30.5
hosts = ["alpha.example.com", "beta.example.com", "gamma.example.com"]

[database]
host = "db.internal"
port = 5432
user = "svc"
password = "hunter2"
max_connections = 100
sslmode = "verify-full"

[database.replica]
host = "db-replica.internal"
port = 5432

[redis]
addr = "redis.internal:6379"
db = 3
pool_size = 50

[logging]
level = "info"
format = "json"
outputs = ["stdout", "/var/log/service.log"]

[features]
new_billing = true
dark_mode = false
beta_users = ["alice", "bob"]

[limits]
requests_per_second = 1000
burst = 2000
max_body_bytes = 1048576
`)

func BenchmarkRealWorldViperRead(b *testing.B) {
	b.SetBytes(int64(len(viperDoc)))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m := map[string]interface{}{}
		if err := toml.Unmarshal(viperDoc, &m); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRealWorldViperWrite(b *testing.B) {
	m := map[string]interface{}{}
	if err := toml.Unmarshal(viperDoc, &m); err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	var buf bytes.Buffer
	for i := 0; i < b.N; i++ {
		buf.Reset()
		if err := toml.NewEncoder(&buf).Encode(m); err != nil {
			b.Fatal(err)
		}
	}
	b.SetBytes(int64(buf.Len()))
}

// ---------------------------------------------------------------------------
// Hugo-style front matter: many small documents decoded into generic maps.

var hugoFrontMatters = func() [][]byte {
	docs := [][]byte{
		[]byte(`title = "Getting started with TOML"
date = 2023-01-15T10:30:00Z
draft = false
tags = ["toml", "go", "tutorial"]
categories = ["development"]
slug = "getting-started-toml"
description = "A short introduction to the TOML configuration language."
`),
		[]byte(`title = "Release notes"
date = 2024-06-02T08:00:00Z
draft = true
tags = ["release"]
aliases = ["/blog/release", "/news/release"]
[params]
author = "team"
toc = true
weight = 12
`),
		[]byte(`title = "About us"
layout = "single"
menu = "main"
weight = 1
[params.seo]
noindex = false
canonical = "https://example.com/about"
`),
		[]byte(`title = "Benchmarking Go programs"
date = 2025-11-23T17:45:12+01:00
lastmod = 2025-11-25T09:00:00+01:00
tags = ["go", "performance", "benchmarks"]
series = ["performance"]
math = false
[cover]
image = "covers/bench.png"
alt = "histogram"
`),
	}
	// A site build parses hundreds of these; cycle through a batch of 40.
	out := make([][]byte, 0, 40)
	for i := 0; len(out) < 40; i++ {
		out = append(out, docs[i%len(docs)])
	}
	return out
}()

func BenchmarkRealWorldHugoFrontMatterBatch(b *testing.B) {
	total := 0
	for _, d := range hugoFrontMatters {
		total += len(d)
	}
	b.SetBytes(int64(total))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, d := range hugoFrontMatters {
			m := map[string]interface{}{}
			if err := toml.Unmarshal(d, &m); err != nil {
				b.Fatal(err)
			}
		}
	}
}

// ---------------------------------------------------------------------------
// gitleaks-style rule files: arrays of tables with nested arrays of tables
// and long literal strings (regexes).

var gitleaksDoc = func() []byte {
	var buf bytes.Buffer
	buf.WriteString(`title = "gitleaks config"

[extend]
useDefault = true

[allowlist]
description = "global allow list"
paths = ['''gitleaks\.toml''', '''(.*?)(jpg|gif|doc)$''', '''vendor/''']
regexes = ['''219-09-9999''', '''078-05-1120''']
stopwords = ["example", "test"]
`)
	rules := []struct{ id, desc, regex string }{
		{"aws-access-key", "AWS Access Key", `(A3T[A-Z0-9]|AKIA|AGPA|AIDA|AROA|AIPA|ANPA|ANVA|ASIA)[A-Z0-9]{16}`},
		{"github-pat", "GitHub Personal Access Token", `ghp_[0-9a-zA-Z]{36}`},
		{"slack-token", "Slack token", `xox[baprs]-([0-9a-zA-Z]{10,48})?`},
		{"private-key", "Asymmetric private key", `-----BEGIN ((EC|PGP|DSA|RSA|OPENSSH) )?PRIVATE KEY( BLOCK)?-----`},
		{"stripe-key", "Stripe key", `(sk|pk)_(test|live)_[0-9a-zA-Z]{10,32}`},
		{"gcp-api-key", "GCP API key", `AIza[0-9A-Za-z\\-_]{35}`},
		{"generic-password", "Generic password assignment", `(?i)(password|passwd|pwd)\s*[:=]\s*['\"][^'\"]{8,64}['\"]`},
		{"jwt", "JSON Web Token", `ey[A-Za-z0-9-_=]+\.[A-Za-z0-9-_=]+\.?[A-Za-z0-9-_.+/=]*`},
	}
	for _, r := range rules {
		buf.WriteString("\n[[rules]]\nid = \"" + r.id + "\"\ndescription = \"" + r.desc + "\"\n")
		buf.WriteString("regex = '''" + r.regex + "'''\nentropy = 3.5\nsecretGroup = 1\n")
		buf.WriteString("keywords = [\"" + r.id + "\"]\ntags = [\"key\", \"" + r.id + "\"]\n")
		buf.WriteString("\n[[rules.allowlists]]\ndescription = \"test files\"\n")
		buf.WriteString("paths = ['''(.*?)_test\\.go''', '''testdata/''']\nstopwords = [\"example\"]\n")
		buf.WriteString("\n[[rules.allowlists]]\ndescription = \"docs\"\nregexes = ['''docs?/''']\n")
	}
	return buf.Bytes()
}()

type gitleaksConfig struct {
	Title  string `toml:"title"`
	Extend struct {
		UseDefault bool `toml:"useDefault"`
	} `toml:"extend"`
	Allowlist struct {
		Description string   `toml:"description"`
		Paths       []string `toml:"paths"`
		Regexes     []string `toml:"regexes"`
		Stopwords   []string `toml:"stopwords"`
	} `toml:"allowlist"`
	Rules []struct {
		ID          string   `toml:"id"`
		Description string   `toml:"description"`
		Regex       string   `toml:"regex"`
		Entropy     float64  `toml:"entropy"`
		SecretGroup int      `toml:"secretGroup"`
		Keywords    []string `toml:"keywords"`
		Tags        []string `toml:"tags"`
		Allowlists  []struct {
			Description string   `toml:"description"`
			Paths       []string `toml:"paths"`
			Regexes     []string `toml:"regexes"`
			Stopwords   []string `toml:"stopwords"`
		} `toml:"allowlists"`
	} `toml:"rules"`
}

func BenchmarkRealWorldGitleaksRules(b *testing.B) {
	b.SetBytes(int64(len(gitleaksDoc)))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var cfg gitleaksConfig
		if err := toml.Unmarshal(gitleaksDoc, &cfg); err != nil {
			b.Fatal(err)
		}
	}
}

// ---------------------------------------------------------------------------
// pyproject.toml-style documents, as read by Go developer tooling: dotted
// keys and long arrays of strings.

var pyprojectDoc = []byte(`
[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"

[project]
name = "example-package"
version = "1.24.3"
description = "An example project configuration"
readme = "README.md"
requires-python = ">=3.9"
license = { text = "MIT" }
keywords = ["sample", "configuration", "packaging"]
authors = [{ name = "Jane Doe", email = "jane@example.com" }]
dependencies = [
  "requests>=2.31.0",
  "click>=8.1",
  "pydantic>=2.0,<3",
  "rich>=13.0",
  "httpx[http2]>=0.27",
  "sqlalchemy>=2.0",
  "alembic>=1.13",
  "uvicorn[standard]>=0.29",
]

[project.optional-dependencies]
dev = ["pytest>=8.0", "pytest-cov", "mypy>=1.10", "ruff>=0.4"]
docs = ["sphinx>=7.0", "furo"]

[project.scripts]
example = "example.cli:main"

[tool.ruff]
line-length = 100
target-version = "py39"
src = ["src", "tests"]

[tool.ruff.lint]
select = ["E", "F", "I", "UP", "B", "SIM"]
ignore = ["E501"]

[tool.ruff.lint.per-file-ignores]
"tests/*" = ["S101"]

[tool.pytest.ini_options]
addopts = "-ra --strict-markers"
testpaths = ["tests"]

[tool.mypy]
python_version = "3.9"
strict = true
warn_unused_ignores = true

[tool.coverage.run]
branch = true
source = ["src"]
`)

func BenchmarkRealWorldPyproject(b *testing.B) {
	b.SetBytes(int64(len(pyprojectDoc)))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m := map[string]interface{}{}
		if err := toml.Unmarshal(pyprojectDoc, &m); err != nil {
			b.Fatal(err)
		}
	}
}

// ---------------------------------------------------------------------------
// golangci-lint-style tool configuration decoded strictly into structs.

var golangciDoc = []byte(`
[run]
timeout = "5m"
tests = true
build-tags = ["integration"]

[output]
formats = ["colored-line-number"]
print-issued-lines = true
sort-results = true

[linters]
disable-all = true
enable = ["errcheck", "govet", "ineffassign", "staticcheck", "unused", "misspell", "revive", "gocritic"]

[linters-settings.errcheck]
check-type-assertions = true
check-blank = false

[linters-settings.govet]
enable-all = true
disable = ["fieldalignment"]

[linters-settings.misspell]
locale = "US"
ignore-words = ["colour"]

[linters-settings.revive]
severity = "warning"
confidence = 0.8

[issues]
max-issues-per-linter = 50
max-same-issues = 3
exclude-use-default = false
exclude-dirs = ["vendor", "third_party"]
`)

type golangciConfig struct {
	Run struct {
		Timeout   string   `toml:"timeout"`
		Tests     bool     `toml:"tests"`
		BuildTags []string `toml:"build-tags"`
	} `toml:"run"`
	Output struct {
		Formats          []string `toml:"formats"`
		PrintIssuedLines bool     `toml:"print-issued-lines"`
		SortResults      bool     `toml:"sort-results"`
	} `toml:"output"`
	Linters struct {
		DisableAll bool     `toml:"disable-all"`
		Enable     []string `toml:"enable"`
	} `toml:"linters"`
	LintersSettings struct {
		Errcheck struct {
			CheckTypeAssertions bool `toml:"check-type-assertions"`
			CheckBlank          bool `toml:"check-blank"`
		} `toml:"errcheck"`
		Govet struct {
			EnableAll bool     `toml:"enable-all"`
			Disable   []string `toml:"disable"`
		} `toml:"govet"`
		Misspell struct {
			Locale      string   `toml:"locale"`
			IgnoreWords []string `toml:"ignore-words"`
		} `toml:"misspell"`
		Revive struct {
			Severity   string  `toml:"severity"`
			Confidence float64 `toml:"confidence"`
		} `toml:"revive"`
	} `toml:"linters-settings"`
	Issues struct {
		MaxIssuesPerLinter int      `toml:"max-issues-per-linter"`
		MaxSameIssues      int      `toml:"max-same-issues"`
		ExcludeUseDefault  bool     `toml:"exclude-use-default"`
		ExcludeDirs        []string `toml:"exclude-dirs"`
	} `toml:"issues"`
}

func BenchmarkRealWorldGolangciStrict(b *testing.B) {
	b.SetBytes(int64(len(golangciDoc)))
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		var cfg golangciConfig
		d := toml.NewDecoder(bytes.NewReader(golangciDoc))
		d.DisallowUnknownFields()
		if err := d.Decode(&cfg); err != nil {
			b.Fatal(err)
		}
	}
}
