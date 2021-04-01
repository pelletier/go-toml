package benchmark_test

import (
	"compress/gzip"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

var bench_inputs = []string{
	// from https://gist.githubusercontent.com/feeeper/2197d6d734729625a037af1df14cf2aa/raw/2f22b120e476d897179be3c1e2483d18067aa7df/config.toml
	"config",

	// converted from https://github.com/miloyip/nativejson-benchmark
	"canada",
	"citm_catalog",
	"twitter",
	"code",

	// converted from https://raw.githubusercontent.com/mailru/easyjson/master/benchmark/example.json
	"example",
}

func BenchmarkUnmarshalDataset(b *testing.B) {
	for _, tc := range bench_inputs {
		buf := fixture(b, tc)
		b.Run(tc, func(b *testing.B) {
			bench(b, func(r runner, b *testing.B) {
				if r.name == "bs" && tc == "canada" {
					// bs can't handle the canada dataset due to mixed integer &
					// floats values in an array.
					b.Skip()
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
