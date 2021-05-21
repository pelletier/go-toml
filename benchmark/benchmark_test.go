package benchmark_test

import (
	"io/ioutil"
	"testing"
	"time"

	"github.com/pelletier/go-toml/v2"
	"github.com/stretchr/testify/require"
)

func BenchmarkUnmarshalSimple(b *testing.B) {
	d := struct {
		A string
	}{}
	doc := []byte(`A = "hello"`)
	for i := 0; i < b.N; i++ {
		err := toml.Unmarshal(doc, &d)
		if err != nil {
			panic(err)
		}
	}
}

type benchmarkDoc struct {
	Table struct {
		Key      string
		Subtable struct {
			Key string
		}
		Inline struct {
			Name struct {
				First string
				Last  string
			}
			Point struct {
				X int64
				U int64
			}
		}
	}
	String struct {
		Basic struct {
			Basic string
		}
		Multiline struct {
			Key1      string
			Key2      string
			Key3      string
			Continued struct {
				Key1 string
				Key2 string
				Key3 string
			}
		}
		Literal struct {
			Winpath   string
			Winpath2  string
			Quoted    string
			Regex     string
			Multiline struct {
				Regex2 string
				Lines  string
			}
		}
	}
	Integer struct {
		Key1        int64
		Key2        int64
		Key3        int64
		Key4        int64
		Underscores struct {
			Key1 int64
			Key2 int64
			Key3 int64
		}
	}
	Float struct {
		Fractional struct {
			Key1 float64
			Key2 float64
			Key3 float64
		}
		Exponent struct {
			Key1 float64
			Key2 float64
			Key3 float64
		}
		Both struct {
			Key float64
		}
		Underscores struct {
			Key1 float64
			Key2 float64
		}
	}
	Boolean struct {
		True  bool
		False bool
	}
	Datetime struct {
		Key1 time.Time
		Key2 time.Time
		Key3 time.Time
	}
	Array struct {
		Key1 []int64
		Key2 []string
		Key3 [][]int64
		// TODO: Key4 not supported by go-toml's Unmarshal
		Key5 []int64
		Key6 []int64
	}
	Products []struct {
		Name  string
		Sku   int64
		Color string
	}
	Fruit []struct {
		Name     string
		Physical struct {
			Color   string
			Shape   string
			Variety []struct {
				Name string
			}
		}
	}
}

func BenchmarkReferenceFile(b *testing.B) {
	bytes, err := ioutil.ReadFile("benchmark.toml")
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(bytes)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		d := benchmarkDoc{}
		err := toml.Unmarshal(bytes, &d)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkReferenceFileMap(b *testing.B) {
	bytes, err := ioutil.ReadFile("benchmark.toml")
	if err != nil {
		b.Fatal(err)
	}
	b.SetBytes(int64(len(bytes)))
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m := map[string]interface{}{}
		err := toml.Unmarshal(bytes, &m)
		if err != nil {
			panic(err)
		}
	}
}

func TestReferenceFile(t *testing.T) {
	bytes, err := ioutil.ReadFile("benchmark.toml")
	require.NoError(t, err)
	d := benchmarkDoc{}
	err = toml.Unmarshal(bytes, &d)
	require.NoError(t, err)
}
