package assert

import (
	"errors"
	"fmt"
	"testing"
)

type Data struct {
	Label string
	Value int64
}

func TestBadMessage(t *testing.T) {
	invalidMessage := func() { True(t, false, 1234) }
	assertOk(t, "Non-fmt message value", func(tb testing.TB) {
		tb.Helper()
		Panics(tb, invalidMessage)
	})
	assertFail(t, "Non-fmt message value", func(tb testing.TB) {
		tb.Helper()
		True(tb, false, "example %s", "message")
	})
}

func TestTrue(t *testing.T) {
	assertOk(t, "Succeed", func(tb testing.TB) {
		tb.Helper()
		True(tb, 1 > 0)
	})
	assertFail(t, "Fail", func(tb testing.TB) {
		tb.Helper()
		True(tb, 1 < 0)
	})
}

func TestFalse(t *testing.T) {
	assertOk(t, "Succeed", func(tb testing.TB) {
		tb.Helper()
		False(tb, 1 < 0)
	})
	assertFail(t, "Fail", func(tb testing.TB) {
		tb.Helper()
		False(tb, 1 > 0)
	})
}

func TestEqual(t *testing.T) {
	assertOk(t, "Nil", func(tb testing.TB) {
		tb.Helper()
		Equal(tb, interface{}(nil), interface{}(nil))
	})
	assertOk(t, "Identical structs", func(tb testing.TB) {
		tb.Helper()
		Equal(tb, Data{"expected", 1234}, Data{"expected", 1234})
	})
	assertFail(t, "Different structs", func(tb testing.TB) {
		tb.Helper()
		Equal(tb, Data{"expected", 1234}, Data{"actual", 1234})
	})
	assertOk(t, "Identical numbers", func(tb testing.TB) {
		tb.Helper()
		Equal(tb, 1234, 1234)
	})
	assertFail(t, "Identical numbers", func(tb testing.TB) {
		tb.Helper()
		Equal(tb, 1234, 1324)
	})
	assertOk(t, "Zero-length byte arrays", func(tb testing.TB) {
		tb.Helper()
		Equal(tb, []byte(nil), []byte(""))
	})
	assertOk(t, "Identical byte arrays", func(tb testing.TB) {
		tb.Helper()
		Equal(tb, []byte{1, 2, 3, 4}, []byte{1, 2, 3, 4})
	})
	assertFail(t, "Different byte arrays", func(tb testing.TB) {
		tb.Helper()
		Equal(tb, []byte{1, 2, 3, 4}, []byte{1, 3, 2, 4})
	})
	assertOk(t, "Identical strings", func(tb testing.TB) {
		tb.Helper()
		Equal(tb, "example", "example")
	})
	assertFail(t, "Identical strings", func(tb testing.TB) {
		tb.Helper()
		Equal(tb, "example", "elpmaxe")
	})
}

func TestError(t *testing.T) {
	assertOk(t, "Error", func(tb testing.TB) {
		tb.Helper()
		Error(tb, errors.New("example"))
	})
	assertFail(t, "Nil", func(tb testing.TB) {
		tb.Helper()
		Error(tb, nil)
	})
}

func TestNoError(t *testing.T) {
	assertFail(t, "Error", func(tb testing.TB) {
		tb.Helper()
		NoError(tb, errors.New("example"))
	})
	assertOk(t, "Nil", func(tb testing.TB) {
		tb.Helper()
		NoError(tb, nil)
	})
}

func TestPanics(t *testing.T) {
	willPanic := func() { panic("example") }
	wontPanic := func() {}
	assertOk(t, "Will panic", func(tb testing.TB) {
		tb.Helper()
		Panics(tb, willPanic)
	})
	assertFail(t, "Won't panic", func(tb testing.TB) {
		tb.Helper()
		Panics(tb, wontPanic)
	})
}

func TestZero(t *testing.T) {
	assertOk(t, "Empty struct", func(tb testing.TB) {
		tb.Helper()
		Zero(tb, Data{})
	})
	assertFail(t, "Non-empty struct", func(tb testing.TB) {
		tb.Helper()
		Zero(tb, Data{Label: "example"})
	})
	assertOk(t, "Nil slice", func(tb testing.TB) {
		tb.Helper()
		var slice []int
		Zero(tb, slice)
	})
	assertFail(t, "Non-empty slice", func(tb testing.TB) {
		tb.Helper()
		slice := []int{1, 2, 3, 4}
		Zero(tb, slice)
	})
	assertOk(t, "Zero-length slice", func(tb testing.TB) {
		tb.Helper()
		slice := []int{}
		Zero(tb, slice)
	})
}

func TestNotZero(t *testing.T) {
	assertFail(t, "Empty struct", func(tb testing.TB) {
		tb.Helper()
		zero := Data{}
		NotZero(tb, zero)
	})
	assertOk(t, "Non-empty struct", func(tb testing.TB) {
		tb.Helper()
		notZero := Data{Label: "example"}
		NotZero(tb, notZero)
	})
	assertFail(t, "Nil slice", func(tb testing.TB) {
		tb.Helper()
		var slice []int
		NotZero(tb, slice)
	})
	assertFail(t, "Zero-length slice", func(tb testing.TB) {
		tb.Helper()
		slice := []int{}
		NotZero(tb, slice)
	})
	assertOk(t, "Non-empty slice", func(tb testing.TB) {
		tb.Helper()
		slice := []int{1, 2, 3, 4}
		NotZero(tb, slice)
	})
}

type testCase struct {
	*testing.T
	failed string
}

func (t *testCase) Fatal(args ...interface{}) {
	t.failed = fmt.Sprint(args...)
}

func (t *testCase) Fatalf(message string, args ...interface{}) {
	t.failed = fmt.Sprintf(message, args...)
}

func assertFail(t *testing.T, name string, fn func(testing.TB)) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		t.Helper()
		test := &testCase{T: t}
		fn(test)
		if test.failed == "" {
			t.Fatal("Test expected to fail but did not")
		} else {
			t.Log(test.failed)
		}
	})
}

func assertOk(t *testing.T, name string, fn func(testing.TB)) {
	t.Helper()
	t.Run(name, func(t *testing.T) {
		t.Helper()
		test := &testCase{T: t}
		fn(test)
		if test.failed != "" {
			t.Fatal("Test expected to succeed but did not:\n", test.failed)
		}
	})
}
