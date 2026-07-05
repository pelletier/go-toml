// Package edit provides comment- and layout-preserving editing of TOML
// documents.
//
// A Document is created from TOML source with Parse. Bytes returns the
// current source, byte-for-byte identical to the input except for the parts
// modified through Set and Delete: an edit rewrites only the bytes that
// express it, leaving the comments, whitespace, and ordering of everything
// else untouched.
//
// Keys are addressed by their path, one element per key part:
// []string{"servers", "alpha", "ip"} addresses ip in [servers.alpha]. Path
// elements are plain strings, never quoted or dotted: quoting is applied as
// needed when writing.
//
// Set and Delete address the structures that make up the document: tables
// and key-values, whether defined by [table] headers or dotted keys. Arrays
// and inline tables are atomic values: Set can replace one wholesale, but
// Set and Delete paths cannot reach inside them. Elements of arrays of
// tables ([[table]]) are not addressable either; Delete removes such an
// array as a whole. Get operates on decoded values instead, so it can
// descend into inline tables.
//
// Values passed to Set are rendered with the same encoder as toml.Marshal,
// in inline (single-line) form. New tables created by Set get their own
// [header] section, appended after the section of their closest existing
// parent, unless that parent was defined with dotted keys, in which case
// dotted keys are used for the new values as well.
//
// Every mutation is validated: if an edit would produce an invalid TOML
// document, the document is left unchanged and an error is returned.
//
// Like the rest of the unstable API, this package does not follow the
// backward compatibility guarantees of go-toml. It also favors correctness
// and fidelity over speed: the document is re-validated and re-indexed after
// every mutation, so it is meant for editing configuration files, not for
// hot paths.
//
// A Document is not safe for concurrent use.
package edit

import (
	"bytes"

	toml "github.com/pelletier/go-toml/v2"
)

// Document is a TOML document whose layout (comments, whitespace, order) is
// preserved across edits.
type Document struct {
	data []byte
	root *table
}

// Parse reads a TOML document and returns a Document ready to be inspected
// and edited. The input must be a valid TOML document: syntactic or semantic
// errors (for example duplicate keys) are returned, as editing an invalid
// document would not produce meaningful results. The input slice is copied
// and can be reused by the caller.
func Parse(b []byte) (*Document, error) {
	var v interface{}
	if err := toml.Unmarshal(b, &v); err != nil {
		return nil, err
	}
	d := &Document{data: bytes.Clone(b)}
	if err := d.reindex(); err != nil {
		return nil, err
	}
	return d, nil
}

// Bytes returns the current TOML source of the document. If the document has
// not been modified, it is identical to the input of Parse.
func (d *Document) Bytes() []byte {
	return bytes.Clone(d.data)
}

// String returns the current TOML source of the document, like Bytes.
func (d *Document) String() string {
	return string(d.data)
}

// Unmarshal decodes the current state of the document into v, like
// toml.Unmarshal.
func (d *Document) Unmarshal(v interface{}) error {
	return toml.Unmarshal(d.data, v)
}

// Get returns the decoded value at the given key path, and whether it
// exists. Values are decoded like toml.Unmarshal into an interface{}: tables
// (inline or not) become map[string]interface{}, arrays and arrays of tables
// become []interface{}, and scalars follow the usual decoding rules. An
// empty path returns the whole document. Unlike Set and Delete, Get descends
// into inline tables.
func (d *Document) Get(key []string) (interface{}, bool) {
	var v interface{}
	if err := toml.Unmarshal(d.data, &v); err != nil {
		// The document is valid by construction.
		return nil, false
	}
	for _, k := range key {
		m, ok := v.(map[string]interface{})
		if !ok {
			return nil, false
		}
		v, ok = m[k]
		if !ok {
			return nil, false
		}
	}
	return v, true
}

// Has reports whether a value exists at the given key path.
func (d *Document) Has(key []string) bool {
	_, ok := d.Get(key)
	return ok
}
