package testsuite

import (
	"testing"

	tomltest "github.com/BurntSushi/toml-test"
)

func TestTomlTestSuite(t *testing.T) {
	run := func(t *testing.T, enc bool) {
		r := tomltest.Runner{
			Files:     tomltest.EmbeddedTests(),
			Encoder:   enc,
			Parser:    parser{},
			SkipTests: []string{},
		}

		tests, err := r.Run()
		if err != nil {
			t.Fatal(err)
		}

		for _, test := range tests.Tests {
			t.Run(test.Path, func(t *testing.T) {
				if test.Failed() {
					t.Fatalf("\nError:\n%s\n\nInput:\n%s\nOutput:\n%s\nWant:\n%s\n",
						test.Failure, test.Input, test.Output, test.Want)
					return
				}
			})
		}
		t.Logf("passed: %d; failed: %d; skipped: %d", tests.Passed, tests.Failed, tests.Skipped)
	}

	t.Run("decode", func(t *testing.T) { run(t, false) })
	t.Run("encode", func(t *testing.T) { run(t, true) })
}
