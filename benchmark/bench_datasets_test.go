package benchmark_test

import (
	"compress/gzip"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

var bench_inputs = []struct {
	name    string
	jsonLen int
}{
	// from https://gist.githubusercontent.com/feeeper/2197d6d734729625a037af1df14cf2aa/raw/2f22b120e476d897179be3c1e2483d18067aa7df/config.toml
	{"config", 806507},

	// converted from https://github.com/miloyip/nativejson-benchmark
	{"canada", 2090234},
	{"citm_catalog", 479897},
	{"twitter", 428778},
	{"code", 1940472},

	// converted from https://raw.githubusercontent.com/mailru/easyjson/master/benchmark/example.json
	{"example", 7779},
}

func TestUnmarshalDatasetCode(t *testing.T) {
	for _, tc := range bench_inputs {
		buf := fixture(t, tc.name)
		t.Run(tc.name, func(t *testing.T) {
			for _, r := range runners {
				if r.name == "bs" && tc.name == "canada" {
					t.Skip("skipping: burntsushi can't handle mixed arrays")
				}

				t.Run(r.name, func(t *testing.T) {
					var v interface{}
					check(t, r.unmarshal(buf, &v))

					b, err := json.Marshal(v)
					check(t, err)
					require.Equal(t, len(b), tc.jsonLen)
				})
			}
		})
	}
}

func BenchmarkUnmarshalDataset(b *testing.B) {
	for _, tc := range bench_inputs {
		buf := fixture(b, tc.name)
		b.Run(tc.name, func(b *testing.B) {
			bench(b, func(r runner, b *testing.B) {
				if r.name == "bs" && tc.name == "canada" {
					b.Skip("skipping: burntsushi can't handle mixed arrays")
				}

				b.SetBytes(int64(len(buf)))
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					var v interface{}
					check(b, r.unmarshal(buf, &v))
				}
			})
		})
	}
}

// fixture returns the uncompressed contents of path.
func fixture(tb testing.TB, path string) []byte {
	f, err := os.Open(filepath.Join("testdata", path+".toml.gz"))
	check(tb, err)
	defer f.Close()

	gz, err := gzip.NewReader(f)
	check(tb, err)

	buf, err := ioutil.ReadAll(gz)
	check(tb, err)

	return buf
}

func check(tb testing.TB, err error) {
	if err != nil {
		tb.Helper()
		tb.Fatal(err)
	}
}
