// Copyright (c) 2017 Tamás Gulácsi
//
// The MIT License (MIT)
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package toml

import (
	"bufio"
	"fmt"
	"io"
	"sort"
	"time"
)

type tomlString string

func (s tomlString) WriteTo(w io.Writer) (int64, error) {
	ew := newErrWriter(w)
	bw := bufio.NewWriterSize(ew, 6*len(s))
	for _, rr := range s {
		switch rr {
		case '\b', '\t', '\n', '\f', '\r', '"', '\\':
			bw.WriteByte('\\')
			switch rr {
			case '\b':
				bw.WriteByte('b')
			case '\t':
				bw.WriteByte('t')
			case '\n':
				bw.WriteByte('n')
			case '\f':
				bw.WriteByte('f')
			case '\r':
				bw.WriteByte('r')
			default:
				bw.WriteByte(byte(rr))
			}
		default:
			if rr >= 0x1F {
				bw.WriteRune(rr)
			} else {
				fmt.Fprintf(bw, "\\u%0.4X", rr)
			}
		}
	}
	err := bw.Flush()
	if err == nil {
		err = ew.Err()
	}
	return ew.Count(), err
}

// WriteIndent w the value of Item with indent as indentation.
func (item tomlValue) WriteIndent(w io.Writer, indent string) (int64, error) {
	ew := newErrWriter(w)
	switch value := item.value.(type) {
	case nil:
		return 0, nil

	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64:
		fmt.Fprintf(ew, "%s%d", indent, value)
	case float32, float64:
		fmt.Fprintf(ew, "%s%f", indent, value)
	case time.Time:
		fmt.Fprintf(ew, "%s%s", indent, value.Format(time.RFC3339))

	case string:
		io.WriteString(ew, indent)
		ew.Write([]byte{'"'})
		tomlString(value).WriteTo(ew)
		ew.Write([]byte{'"'})
	case bool:
		t := []byte("false")
		if value {
			t = []byte("true")
		}
		ew.Write(t)
	case []interface{}:
		ew.Write([]byte{'['})
		for i, item := range value {
			if i != 0 {
				ew.Write([]byte{','})
			}
			asTomlValue(item).WriteIndent(ew, "")
		}
		ew.Write([]byte{']'})
	default:
		return 0, fmt.Errorf("unsupported value type %T: %v", value, value)
	}
	return ew.Count(), ew.Err()
}

func asTomlValue(i interface{}) tomlValue {
	switch v := i.(type) {
	case tomlValue:
		return v
	case *tomlValue:
		return *v
	default:
		return tomlValue{value: i}
	}
}

// WriteToToml w the text representation of the tree, in TOML format.
// For the root tree, use "","" as indent and keyspace.
func (t *TomlTree) WriteToToml(w io.Writer, indent, keyspace string) (int64, error) {
	ew := newErrWriter(w)

	keys := make([]string, 0, len(t.values))
	for k := range t.values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Simple values comes first
	for _, k := range keys {
		v := t.values[k]
		switch v.(type) {
		case *TomlTree, []*TomlTree,
			map[string]interface{},
			map[string]string,
			map[interface{}]interface{}:
			continue
		default:
			fmt.Fprintf(ew, "%s%s = ", indent, k)
			asTomlValue(v).WriteIndent(ew, "")
			ew.Write([]byte{'\n'})
		}
	}

	// Now the maps and trees
	for _, k := range keys {
		v := t.values[k]
		// convert maps to TomlTree
		switch node := v.(type) {
		case map[string]interface{}:
			v = TreeFromMap(node)
		case map[string]string:
			v = TreeFromMap(convertMapStringString(node))
		case map[interface{}]interface{}:
			v = TreeFromMap(convertMapInterfaceInterface(node))
		case *TomlTree, []*TomlTree:
			// later
		default:
			// simple values are already printed
			continue
		}

		// figure out the keyspace
		combinedKey := k
		if keyspace != "" {
			combinedKey = keyspace + "." + combinedKey
		}

		// output based on type
		switch node := v.(type) {
		case *TomlTree:
			if len(node.Keys()) > 0 {
				fmt.Fprintf(ew, "\n%s[%s]\n", indent, combinedKey)
			}
			node.WriteToToml(ew, indent+"  ", combinedKey)
		case []*TomlTree:
			for _, item := range node {
				if len(item.Keys()) > 0 {
					fmt.Fprintf(ew, "\n%s[[%s]]\n", indent, combinedKey)
				}
				item.WriteToToml(ew, indent+"  ", combinedKey)
			}
		default:
			panic(fmt.Errorf("Should not meet not *TomlTree/[]*TomlTree here, got %T", v))
		}

	}
	return ew.Count(), ew.Err()
}

type errWriter struct {
	w   io.Writer
	n   int64
	err error
}

func newErrWriter(w io.Writer) *errWriter {
	if ew, ok := w.(*errWriter); ok {
		return ew
	}
	return &errWriter{w: w}
}

func (ew *errWriter) Write(p []byte) (int, error) {
	if ew.err != nil {
		return 0, ew.err
	}
	n, err := ew.w.Write(p)
	ew.n += int64(n)
	ew.err = err
	return n, err
}
func (ew *errWriter) Count() int64 { return ew.n }
func (ew *errWriter) Err() error   { return ew.err }
