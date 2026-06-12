package toml

import (
	"encoding"
	"fmt"
	"io"
	"math"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pelletier/go-toml/v2/internal/tracker"
	"github.com/pelletier/go-toml/v2/unstable"
)

// Unmarshal deserializes a TOML document into a Go value.
//
// It is a shortcut for Decoder.Decode() with the default options.
func Unmarshal(data []byte, v interface{}) error {
	d := decoder{}
	return d.unmarshal(data, v)
}

// Decoder reads and decode a TOML document from an input stream.
type Decoder struct {
	// input
	r io.Reader

	// global settings
	strict bool

	// toggles unmarshaler interface
	unmarshalerInterface bool
}

// NewDecoder creates a new Decoder that will read from r.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r}
}

// DisallowUnknownFields causes the Decoder to return an error when the
// destination is a struct and the input contains a key that does not match a
// non-ignored field.
//
// In that case, the Decoder returns a StrictMissingError that can be used to
// retrieve the individual errors as well as generate a human readable
// description of the missing fields.
func (d *Decoder) DisallowUnknownFields() *Decoder {
	d.strict = true
	return d
}

// EnableUnmarshalerInterface allows to enable unmarshaler interface.
//
// With this feature enabled, types implementing the unstable.Unmarshaler
// interface can be decoded from any structure of the document. It allows types
// that don't have a straightforward TOML representation to provide their own
// decoding logic.
//
// The UnmarshalTOML method receives raw TOML bytes:
//   - For single values: the raw value bytes (e.g., `"hello"` for a string)
//   - For tables: all key-value lines belonging to that table
//   - For inline tables/arrays: the raw bytes of the inline structure
//
// The unstable.RawMessage type can be used to capture raw TOML bytes for
// later processing, similar to json.RawMessage.
//
// *Unstable:* This method does not follow the compatibility guarantees of
// semver. It can be changed or removed without a new major version being
// issued.
func (d *Decoder) EnableUnmarshalerInterface() *Decoder {
	d.unmarshalerInterface = true
	return d
}

// Decode the whole content of r into v.
//
// By default, values in the document that don't exist in the target Go value
// are ignored. See Decoder.DisallowUnknownFields() to change this behavior.
//
// When a TOML local date, time, or date-time is decoded into a time.Time, its
// value is represented in time.Local timezone. Otherwise the appropriate Local*
// structure is used. For time values, precision up to the nanosecond is
// supported by truncating extra digits.
//
// Empty tables decoded in an interface{} create an empty initialized
// map[string]interface{}.
//
// Types implementing the encoding.TextUnmarshaler interface are decoded from a
// TOML string.
//
// When decoding a number, go-toml will return an error if the number is out of
// bounds for the target type (which includes negative numbers when decoding
// into an unsigned int).
//
// If an error occurs while decoding the content of the document, this function
// returns a toml.DecodeError, providing context about the issue. When using
// strict mode and a field is missing, a `toml.StrictMissingError` is
// returned. In any other case, this function returns a standard Go error.
//
// # Type mapping
//
// List of supported TOML types and their associated accepted Go types:
//
//	String           -> string
//	Integer          -> uint*, int*, depending on size
//	Float            -> float*, depending on size
//	Boolean          -> bool
//	Offset Date-Time -> time.Time
//	Local Date-time  -> LocalDateTime, time.Time
//	Local Date       -> LocalDate, time.Time
//	Local Time       -> LocalTime, time.Time
//	Array            -> slice and array, depending on elements types
//	Table            -> map and struct
//	Inline Table     -> same as Table
//	Array of Tables  -> same as Array and Table
func (d *Decoder) Decode(v interface{}) error {
	b, err := io.ReadAll(d.r)
	if err != nil {
		return fmt.Errorf("toml: %w", err)
	}

	dec := decoder{
		strict: strict{
			Enabled: d.strict,
		},
		unmarshalerInterface: d.unmarshalerInterface,
	}

	return dec.unmarshal(b, v)
}

// pathPart is one part of the key path leading to a value. Parts that come
// from the current table header only carry a name; parts that come from the
// key of the current key-value expression also carry the AST node.
type pathPart struct {
	name string
	node *unstable.Node
}

// rawCapture accumulates the raw bytes fed to a type implementing
// unstable.Unmarshaler for a table target. The target is identified by the
// parts of its key and the array-table indexes in effect when the capture
// was created, so that it can be located again once the whole document has
// been processed (the address of the target may change as slices grow).
type rawCapture struct {
	names []string
	// indexes[i] is the index to use when reaching a slice or array right
	// before consuming names[i]. indexes[len(names)] is the index of the
	// element when the target is an element of an array table. -1 when not
	// relevant.
	indexes []int
	buf     []byte
}

type decoder struct {
	p unstable.Parser

	// strict mode
	strict strict

	// toggles unmarshaler interface
	unmarshalerInterface bool

	// tracks the duplicate and type consistency of the keys
	seen tracker.SeenTracker

	// path of the current table header, as copied strings
	tableKey []string

	// true when the expressions under the current table header cannot be
	// stored anywhere and should be skipped
	skipUntilTable bool

	// scratch buffer for the key path of the current expression
	path []pathPart

	// raw captures for the unmarshaler interface, in order of first
	// appearance. captureIdx is the index of the capture the current table
	// belongs to, or -1.
	captures   []rawCapture
	captureIdx int

	// segIdx[i] records the array element index used when traversing a
	// slice or array right before consuming the i-th part of the current
	// table key. Reset for each table expression.
	segIdx []int

	// arrayCounts tracks the number of elements appended to fixed-size
	// arrays used as array tables, keyed by the NUL-joined key parts.
	arrayCounts map[string]int
}

func joinPath(parts []string) string {
	return strings.Join(parts, "\x00")
}

// arrayCount returns the number of elements appended so far to the array
// table at the given path.
func (d *decoder) arrayCount(key string) int {
	if d.arrayCounts == nil {
		return 0
	}
	return d.arrayCounts[key]
}

func (d *decoder) setArrayCount(key string, n int) {
	if d.arrayCounts == nil {
		d.arrayCounts = map[string]int{}
	}
	d.arrayCounts[key] = n
}

// resetChildArrayCounts forgets the counts of all the array tables under
// the given path, so that a new element starts fresh.
func (d *decoder) resetChildArrayCounts(key string) {
	prefix := key + "\x00"
	for k := range d.arrayCounts {
		if strings.HasPrefix(k, prefix) {
			delete(d.arrayCounts, k)
		}
	}
}

func (d *decoder) typeMismatchError(toml string, target reflect.Type, highlight []byte) error {
	return &typeMismatchError{
		toml:      toml,
		target:    target,
		highlight: highlight,
	}
}

type typeMismatchError struct {
	toml      string
	target    reflect.Type
	highlight []byte
}

func (e *typeMismatchError) Error() string {
	return fmt.Sprintf("cannot decode TOML %s into %s", e.toml, e.target)
}

func (d *decoder) unmarshal(data []byte, v interface{}) error {
	r := reflect.ValueOf(v)
	if r.Kind() != reflect.Ptr {
		return fmt.Errorf("toml: decoding can only be performed into a pointer, not %s", r.Kind())
	}
	if r.IsNil() {
		return fmt.Errorf("toml: decoding pointer target cannot be nil")
	}

	root := r.Elem()

	d.captureIdx = -1
	d.p.Reset(data)
	for d.p.NextExpression() {
		err := d.handleRootExpression(d.p.Expression(), root)
		if err != nil {
			return d.wrapError(data, err)
		}
	}
	if err := d.p.Error(); err != nil {
		if perr, ok := err.(*unstable.ParserError); ok {
			return wrapDecodeError(data, perr)
		}
		return err
	}

	// Deliver the accumulated raw documents to the unmarshaler-interface
	// targets.
	for i := range d.captures {
		nv, err := d.resolveCapture(root, &d.captures[i], 0, false)
		if err != nil {
			return err
		}
		if nv.IsValid() {
			root.Set(nv)
		}
	}

	// An empty document into a generic target still initializes it.
	switch root.Kind() {
	case reflect.Map:
		if root.IsNil() {
			root.Set(reflect.MakeMap(root.Type()))
		}
	case reflect.Interface:
		if root.IsNil() {
			root.Set(reflect.ValueOf(map[string]interface{}{}))
		}
	}

	return d.strict.Error(data)
}

// wrapError gives document context to errors generated while processing an
// expression.
func (d *decoder) wrapError(data []byte, err error) error {
	switch e := err.(type) {
	case *unstable.ParserError:
		return wrapDecodeError(data, e)
	case *typeMismatchError:
		return wrapDecodeError(data, &unstable.ParserError{
			Highlight: e.highlight,
			Message:   e.Error(),
		})
	default:
		return err
	}
}

func (d *decoder) handleRootExpression(expr *unstable.Node, root reflect.Value) error {
	first, err := d.seen.CheckExpression(expr)
	if err != nil {
		return err
	}

	switch expr.Kind {
	case unstable.KeyValue:
		if d.skipUntilTable {
			return nil
		}
		if d.captureIdx >= 0 {
			d.captureKeyValue(expr)
			return nil
		}
		return d.handleKeyValueExpression(expr, root)
	case unstable.Table:
		d.skipUntilTable = false
		d.captureIdx = -1
		d.strict.EnterTable(expr)
		return d.handleTableExpression(expr, root, false, first)
	case unstable.ArrayTable:
		d.skipUntilTable = false
		d.captureIdx = -1
		d.strict.EnterTable(expr)
		return d.handleTableExpression(expr, root, true, first)
	default:
		return unstable.NewParserError(expr.Data, "unsupported expression kind %s", expr.Kind)
	}
}

// updateTableKey copies the parts of the key of a table expression into
// tableKey.
func (d *decoder) updateTableKey(expr *unstable.Node) {
	d.tableKey = d.tableKey[:0]
	it := expr.Key()
	for it.Next() {
		d.tableKey = append(d.tableKey, string(it.Node().Data))
	}
}

// handleTableExpression processes a [table] or [[array table]] expression:
// it creates the intermediate containers, applies the strict policy, hooks
// the unmarshaler interface captures, and saves the table key for the
// key-values that follow.
func (d *decoder) handleTableExpression(expr *unstable.Node, root reflect.Value, isArrayTable bool, first bool) error {
	d.updateTableKey(expr)

	// Check whether this table belongs to an exisiting raw capture (split
	// tables, or children of a table assigned to an Unmarshaler).
	if d.unmarshalerInterface {
		if d.resumeCapture(expr) {
			return nil
		}
	}

	// Reset the per-segment array indexes.
	d.segIdx = d.segIdx[:0]
	for i := 0; i <= len(d.tableKey); i++ {
		d.segIdx = append(d.segIdx, -1)
	}

	nv, err := d.descendTable(root, expr, 0, isArrayTable, first)
	if err != nil {
		return err
	}
	if nv.IsValid() {
		root.Set(nv)
	}
	return nil
}

// resumeCapture looks for an existing capture this table expression belongs
// to. It returns true if the expression was consumed.
func (d *decoder) resumeCapture(expr *unstable.Node) bool {
	// Iterate in reverse, so that tables attach to the latest element of
	// array tables.
	for i := len(d.captures) - 1; i >= 0; i-- {
		c := &d.captures[i]
		if len(d.tableKey) < len(c.names) {
			continue
		}
		if expr.Kind == unstable.ArrayTable && len(d.tableKey) == len(c.names) {
			// A new element of an array table is not part of the capture of
			// the previous element.
			continue
		}
		match := true
		for j, p := range c.names {
			if d.tableKey[j] != p {
				match = false
				break
			}
		}
		if !match {
			continue
		}
		d.captureIdx = i
		if len(d.tableKey) > len(c.names) {
			d.appendCaptureHeader(c, expr, len(c.names))
		}
		return true
	}
	return false
}

// appendCaptureHeader writes the table header of expr in the capture buffer,
// adjusted to be relative to the capture root.
func (d *decoder) appendCaptureHeader(c *rawCapture, expr *unstable.Node, skip int) {
	c.buf = append(c.buf, '[')
	if expr.Kind == unstable.ArrayTable {
		c.buf = append(c.buf, '[')
	}
	c.buf = append(c.buf, d.rawKeySuffix(expr, skip)...)
	c.buf = append(c.buf, ']')
	if expr.Kind == unstable.ArrayTable {
		c.buf = append(c.buf, ']')
	}
	c.buf = append(c.buf, '\n')
}

// rawKeySuffix returns the raw bytes of the key of the expression, skipping
// the first n parts.
func (d *decoder) rawKeySuffix(expr *unstable.Node, n int) []byte {
	it := expr.Key()
	idx := 0
	var start, end unstable.Range
	for it.Next() {
		if idx >= n {
			r := it.Node().Raw
			if start.Length == 0 && start.Offset == 0 && idx == n {
				start = r
			}
			end = r
		}
		idx++
	}
	return d.p.Data()[start.Offset : end.Offset+end.Length]
}

// startCapture registers a new capture for the table at the given path
// (prefix of tableKey).
func (d *decoder) startCapture(pathLen int, expr *unstable.Node) {
	names := make([]string, pathLen)
	copy(names, d.tableKey[:pathLen])
	indexes := make([]int, pathLen+1)
	copy(indexes, d.segIdx[:pathLen+1])
	d.captures = append(d.captures, rawCapture{
		names:   names,
		indexes: indexes,
	})
	d.captureIdx = len(d.captures) - 1
	if pathLen < len(d.tableKey) {
		d.appendCaptureHeader(&d.captures[d.captureIdx], expr, pathLen)
	}
}

// resolveCapture walks back to the target of a capture and delivers the
// accumulated raw bytes to its UnmarshalTOML implementation.
func (d *decoder) resolveCapture(v reflect.Value, c *rawCapture, idx int, indexed bool) (reflect.Value, error) {
	if v.Kind() == reflect.Ptr {
		if v.Type().Implements(unmarshalerType) && idx == len(c.names) {
			u, _ := unmarshalerOf(v)
			return v, u.UnmarshalTOML(c.buf)
		}
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		nv, err := d.resolveCapture(v.Elem(), c, idx, indexed)
		if err != nil {
			return reflect.Value{}, err
		}
		if nv.IsValid() {
			v.Elem().Set(nv)
		}
		return v, nil
	}

	if !indexed && (v.Kind() == reflect.Slice || v.Kind() == reflect.Array) && c.indexes[idx] >= 0 {
		i := c.indexes[idx]
		if i >= v.Len() {
			return reflect.Value{}, fmt.Errorf("toml: internal error: capture index out of range")
		}
		elem := v.Index(i)
		nv, err := d.resolveCapture(elem, c, idx, true)
		if err != nil {
			return reflect.Value{}, err
		}
		if nv.IsValid() {
			elem.Set(nv)
		}
		return v, nil
	}

	if idx == len(c.names) {
		u, ok := unmarshalerOf(v)
		if !ok {
			return reflect.Value{}, fmt.Errorf("toml: internal error: capture target does not implement UnmarshalTOML")
		}
		return v, u.UnmarshalTOML(c.buf)
	}

	name := c.names[idx]

	switch v.Kind() {
	case reflect.Struct:
		plan := planForType(v.Type())
		f, found := plan.lookup(name)
		if !found {
			return v, nil
		}
		fv := fieldByIndexAlloc(v, f.index)
		nv, err := d.resolveCapture(fv, c, idx+1, false)
		if err != nil {
			return reflect.Value{}, err
		}
		if nv.IsValid() && fv.CanSet() {
			fv.Set(nv)
		}
		return v, nil
	case reflect.Map:
		key, err := makeMapKey(v.Type().Key(), name)
		if err != nil {
			return reflect.Value{}, err
		}
		if v.IsNil() {
			v = reflect.MakeMap(v.Type())
		}
		elem := reflect.New(v.Type().Elem()).Elem()
		if existing := v.MapIndex(key); existing.IsValid() {
			elem.Set(existing)
		}
		nv, err := d.resolveCapture(elem, c, idx+1, false)
		if err != nil {
			return reflect.Value{}, err
		}
		if nv.IsValid() {
			v.SetMapIndex(key, nv)
		}
		return v, nil
	case reflect.Interface:
		elem, err := elemOrNewMap(v)
		if err != nil {
			return reflect.Value{}, err
		}
		nv, err := d.resolveCapture(elem, c, idx, indexed)
		if err != nil || !nv.IsValid() {
			return reflect.Value{}, err
		}
		boxed := reflect.New(v.Type()).Elem()
		boxed.Set(nv)
		return boxed, nil
	default:
		return reflect.Value{}, fmt.Errorf("toml: internal error: cannot resolve capture target through %s", v.Kind())
	}
}

// captureKeyValue appends the raw bytes of a key-value expression to the
// current capture.
func (d *decoder) captureKeyValue(expr *unstable.Node) {
	c := &d.captures[d.captureIdx]
	c.buf = append(c.buf, d.p.Raw(expr.Raw)...)
	c.buf = append(c.buf, '\n')
}

// descendTable walks the key of a table expression, creating the
// intermediate containers on the way. It returns the value to write back at
// this level, or an invalid value if the table turned out not to be
// storable (strict mode bookkeeping happens inside).
func (d *decoder) descendTable(v reflect.Value, expr *unstable.Node, idx int, isArrayTable bool, first bool) (reflect.Value, error) {
	// pointers are allocated and traversed
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		nv, err := d.descendTable(v.Elem(), expr, idx, isArrayTable, first)
		if err != nil || !nv.IsValid() {
			return reflect.Value{}, err
		}
		v.Elem().Set(nv)
		return v, nil
	}

	// Tables assigned to a type implementing the unmarshaler interface are
	// captured as raw bytes, delivered once the document is fully read.
	if d.unmarshalerInterface {
		if hasUnmarshaler(v) {
			d.startCapture(idx, expr)
			return reflect.Value{}, nil
		}
	}

	if idx >= len(d.tableKey) {
		return d.finalizeTable(v, expr, isArrayTable, first)
	}

	name := d.tableKey[idx]

	switch v.Kind() {
	case reflect.Map:
		key, err := makeMapKey(v.Type().Key(), name)
		if err != nil {
			return reflect.Value{}, err
		}
		if v.IsNil() {
			v = reflect.MakeMap(v.Type())
		}
		elem := reflect.New(v.Type().Elem()).Elem()
		if existing := v.MapIndex(key); existing.IsValid() {
			elem.Set(existing)
		}
		nv, err := d.descendTable(elem, expr, idx+1, isArrayTable, first)
		if err != nil {
			return reflect.Value{}, err
		}
		if nv.IsValid() {
			v.SetMapIndex(key, nv)
		}
		return v, nil
	case reflect.Struct:
		plan := planForType(v.Type())
		f, found := plan.lookup(name)
		if !found {
			d.strict.MissingTable(expr)
			d.skipUntilTable = true
			return reflect.Value{}, nil
		}
		fv := fieldByIndexAlloc(v, f.index)
		nv, err := d.descendTable(fv, expr, idx+1, isArrayTable, first)
		if err != nil {
			return reflect.Value{}, err
		}
		if nv.IsValid() && fv.CanSet() {
			fv.Set(nv)
		}
		return v, nil
	case reflect.Interface:
		elem, err := elemOrNewMap(v)
		if err != nil {
			return reflect.Value{}, err
		}
		nv, err := d.descendTable(elem, expr, idx, isArrayTable, first)
		if err != nil || !nv.IsValid() {
			return reflect.Value{}, err
		}
		boxed := reflect.New(v.Type()).Elem()
		boxed.Set(nv)
		return boxed, nil
	case reflect.Slice:
		if v.Len() == 0 {
			// Implicit creation of the first element: the array table that
			// would create it has not been seen yet (issue 995).
			if v.IsNil() {
				v = reflect.MakeSlice(v.Type(), 0, 4)
			}
			v = reflect.Append(v, reflect.New(v.Type().Elem()).Elem())
		}
		elemIdx := v.Len() - 1
		d.segIdx[idx] = elemIdx
		elem := v.Index(elemIdx)
		nv, err := d.descendTable(elem, expr, idx, isArrayTable, first)
		if err != nil {
			return reflect.Value{}, err
		}
		if nv.IsValid() {
			elem.Set(nv)
		}
		return v, nil
	case reflect.Array:
		key := joinPath(d.tableKey[:idx])
		cnt := d.arrayCount(key)
		if cnt == 0 {
			// Implicit creation of the first element.
			cnt = 1
			d.setArrayCount(key, 1)
		}
		elemIdx := cnt - 1
		if elemIdx >= v.Len() {
			return reflect.Value{}, unstable.NewParserError(d.p.Raw(expr.Raw), "cannot reach element %d of array of size %d", elemIdx, v.Len())
		}
		d.segIdx[idx] = elemIdx
		elem := v.Index(elemIdx)
		nv, err := d.descendTable(elem, expr, idx, isArrayTable, first)
		if err != nil {
			return reflect.Value{}, err
		}
		if nv.IsValid() {
			elem.Set(nv)
		}
		return v, nil
	default:
		return reflect.Value{}, unstable.NewParserError(d.p.Raw(expr.Raw), "cannot store a table in a %s", v.Kind())
	}
}

// finalizeTable applies the effect of a table expression once its key is
// fully traversed: making sure containers exist, and appending an element
// for array tables.
func (d *decoder) finalizeTable(v reflect.Value, expr *unstable.Node, isArrayTable bool, first bool) (reflect.Value, error) {
	if !isArrayTable {
		switch v.Kind() {
		case reflect.Map:
			// Concrete maps are created lazily, when their first key is
			// set: an empty table header leaves a nil map untouched.
			return v, nil
		case reflect.Struct:
			return v, nil
		case reflect.Interface:
			if v.IsNil() {
				return reflect.ValueOf(map[string]interface{}{}), nil
			}
			return v, nil
		default:
			return reflect.Value{}, fmt.Errorf("toml: cannot store a table in a %s", v.Kind())
		}
	}

	// Array table: append an element and reset the state of all the nested
	// array tables.
	key := joinPath(d.tableKey)
	d.resetChildArrayCounts(key)

	switch v.Kind() {
	case reflect.Slice:
		if v.IsNil() {
			v = reflect.MakeSlice(v.Type(), 0, 4)
		} else if first {
			v = v.Slice(0, 0)
		}
		elem := reflect.New(v.Type().Elem()).Elem()
		v = reflect.Append(v, elem)
		elemIdx := v.Len() - 1
		d.setArrayCount(key, elemIdx+1)
		d.segIdx[len(d.tableKey)] = elemIdx
		if d.unmarshalerInterface && hasUnmarshaler(v.Index(elemIdx)) {
			d.startCapture(len(d.tableKey), expr)
		}
		return v, nil
	case reflect.Array:
		cnt := d.arrayCount(key)
		if first {
			cnt = 0
		}
		if cnt >= v.Len() {
			return reflect.Value{}, unstable.NewParserError(d.p.Raw(expr.Raw), "array of size %d is too small to store this array table", v.Len())
		}
		v.Index(cnt).Set(reflect.Zero(v.Type().Elem()))
		d.setArrayCount(key, cnt+1)
		d.segIdx[len(d.tableKey)] = cnt
		if d.unmarshalerInterface && hasUnmarshaler(v.Index(cnt)) {
			d.startCapture(len(d.tableKey), expr)
		}
		return v, nil
	case reflect.Interface:
		var slice []interface{}
		if !v.IsNil() {
			if s, ok := v.Interface().([]interface{}); ok {
				slice = s
			}
		}
		if first {
			slice = slice[:0]
		}
		slice = append(slice, map[string]interface{}{})
		d.setArrayCount(key, len(slice))
		d.segIdx[len(d.tableKey)] = len(slice) - 1
		return reflect.ValueOf(slice), nil
	default:
		return reflect.Value{}, fmt.Errorf("toml: cannot store an array table in a %s", v.Kind())
	}
}

// hasUnmarshaler reports whether v can provide an unstable.Unmarshaler,
// without allocating anything.
func hasUnmarshaler(v reflect.Value) bool {
	t := v.Type()
	return t.Implements(unmarshalerType) || (v.CanAddr() && reflect.PtrTo(t).Implements(unmarshalerType))
}

// makeMapKey converts a TOML key into a value usable as the given map key
// type.
func makeMapKey(kt reflect.Type, name string) (reflect.Value, error) {
	switch kt.Kind() {
	case reflect.String:
		return reflect.ValueOf(name).Convert(kt), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(name, 10, 64)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("toml: cannot parse map key %q as %s: %w", name, kt, err)
		}
		k := reflect.New(kt).Elem()
		if k.OverflowInt(i) {
			return reflect.Value{}, fmt.Errorf("toml: map key %q overflows %s", name, kt)
		}
		k.SetInt(i)
		return k, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		u, err := strconv.ParseUint(name, 10, 64)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("toml: cannot parse map key %q as %s: %w", name, kt, err)
		}
		k := reflect.New(kt).Elem()
		if k.OverflowUint(u) {
			return reflect.Value{}, fmt.Errorf("toml: map key %q overflows %s", name, kt)
		}
		k.SetUint(u)
		return k, nil
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(name, 64)
		if err != nil {
			return reflect.Value{}, fmt.Errorf("toml: cannot parse map key %q as %s: %w", name, kt, err)
		}
		k := reflect.New(kt).Elem()
		k.SetFloat(f)
		return k, nil
	case reflect.Ptr:
		if kt.Implements(textUnmarshalerType) {
			k := reflect.New(kt.Elem())
			err := k.Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(name))
			if err != nil {
				return reflect.Value{}, fmt.Errorf("toml: error unmarshaling map key %q: %w", name, err)
			}
			return k, nil
		}
	default:
		if reflect.PtrTo(kt).Implements(textUnmarshalerType) {
			k := reflect.New(kt)
			err := k.Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(name))
			if err != nil {
				return reflect.Value{}, fmt.Errorf("toml: error unmarshaling map key %q: %w", name, err)
			}
			return k.Elem(), nil
		}
	}
	return reflect.Value{}, fmt.Errorf("toml: cannot decode a key into a map with key type %s", kt)
}

// elemOrNewMap unwraps an interface value to descend into it. Contents that
// can hold a table (generic maps and slices) are kept; anything else is
// replaced by a fresh map[string]interface{}. The result is an addressable
// copy of the content.
func elemOrNewMap(v reflect.Value) (reflect.Value, error) {
	if !v.IsNil() {
		concrete := v.Elem()
		t := concrete.Type()
		if t == mapStringInterfaceType || t == sliceInterfaceType {
			tmp := reflect.New(t).Elem()
			tmp.Set(concrete)
			return tmp, nil
		}
	}
	return reflect.ValueOf(map[string]interface{}{}), nil
}

// handleKeyValueExpression stores the value of a top-level key-value
// expression, relative to the current table.
func (d *decoder) handleKeyValueExpression(expr *unstable.Node, root reflect.Value) error {
	d.path = d.path[:0]
	for _, name := range d.tableKey {
		d.path = append(d.path, pathPart{name: name})
	}
	it := expr.Key()
	for it.Next() {
		n := it.Node()
		d.path = append(d.path, pathPart{name: string(n.Data), node: n})
	}

	nv, err := d.descend(root, d.path, 0, expr, expr.Value())
	if err != nil {
		return err
	}
	if nv.IsValid() {
		root.Set(nv)
	}
	return nil
}

// descend walks the given key path into v, and assigns the value at the
// end. It returns the value to store back at this level. An invalid value
// means nothing should be stored (e.g. unknown field).
func (d *decoder) descend(v reflect.Value, path []pathPart, idx int, expr *unstable.Node, value *unstable.Node) (reflect.Value, error) {
	if idx == len(path) {
		return d.assignValue(v, expr, value)
	}

	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		nv, err := d.descend(v.Elem(), path, idx, expr, value)
		if err != nil || !nv.IsValid() {
			return reflect.Value{}, err
		}
		v.Elem().Set(nv)
		return v, nil
	}

	// A target implementing the unmarshaler interface consumes the value,
	// whatever the remaining parts of the key are.
	if d.unmarshalerInterface {
		if u, ok := unmarshalerOf(v); ok {
			return v, u.UnmarshalTOML(d.rawValue(expr, value))
		}
	}

	part := path[idx]

	switch v.Kind() {
	case reflect.Map:
		key, err := makeMapKey(v.Type().Key(), part.name)
		if err != nil {
			return reflect.Value{}, err
		}
		if v.IsNil() {
			v = reflect.MakeMap(v.Type())
		}
		elem := reflect.New(v.Type().Elem()).Elem()
		if existing := v.MapIndex(key); existing.IsValid() {
			elem.Set(existing)
		}
		nv, err := d.descend(elem, path, idx+1, expr, value)
		if err != nil {
			return reflect.Value{}, err
		}
		if nv.IsValid() {
			v.SetMapIndex(key, nv)
		}
		return v, nil
	case reflect.Struct:
		plan := planForType(v.Type())
		f, found := plan.lookup(part.name)
		if !found {
			if part.node != nil {
				d.strict.MissingField(expr)
			}
			return v, nil
		}
		fv := fieldByIndexAlloc(v, f.index)
		nv, err := d.descend(fv, path, idx+1, expr, value)
		if err != nil {
			if mm, ok := err.(*typeMismatchError); ok {
				err = &unstable.ParserError{
					Highlight: mm.highlight,
					Message: fmt.Sprintf("cannot decode TOML %s into struct field %s.%s of type %s",
						mm.toml, v.Type(), f.fieldName, mm.target),
				}
			}
			return reflect.Value{}, err
		}
		if nv.IsValid() && fv.CanSet() {
			fv.Set(nv)
		}
		return v, nil
	case reflect.Interface:
		elem, err := elemOrNewMap(v)
		if err != nil {
			return reflect.Value{}, err
		}
		nv, err := d.descend(elem, path, idx, expr, value)
		if err != nil || !nv.IsValid() {
			return reflect.Value{}, err
		}
		boxed := reflect.New(v.Type()).Elem()
		boxed.Set(nv)
		return boxed, nil
	case reflect.Slice:
		if v.Len() == 0 {
			if v.IsNil() {
				v = reflect.MakeSlice(v.Type(), 0, 4)
			}
			v = reflect.Append(v, reflect.New(v.Type().Elem()).Elem())
		}
		elem := v.Index(v.Len() - 1)
		nv, err := d.descend(elem, path, idx, expr, value)
		if err != nil {
			return reflect.Value{}, err
		}
		if nv.IsValid() {
			elem.Set(nv)
		}
		return v, nil
	case reflect.Array:
		names := make([]string, idx)
		for i := range names {
			names[i] = path[i].name
		}
		cnt := d.arrayCount(joinPath(names))
		if cnt == 0 {
			cnt = 1
		}
		elemIdx := cnt - 1
		if elemIdx >= v.Len() {
			return reflect.Value{}, unstable.NewParserError(keyHighlight(d.p.Data(), part.node), "cannot reach element %d of array of size %d", elemIdx, v.Len())
		}
		elem := v.Index(elemIdx)
		nv, err := d.descend(elem, path, idx, expr, value)
		if err != nil {
			return reflect.Value{}, err
		}
		if nv.IsValid() {
			elem.Set(nv)
		}
		return v, nil
	default:
		return reflect.Value{}, d.typeMismatchError("table", v.Type(), keyHighlight(d.p.Data(), part.node))
	}
}

// keyHighlight returns a highlight for the given key part node, falling back
// to the start of the document.
func keyHighlight(doc []byte, node *unstable.Node) []byte {
	if node == nil {
		return doc[0:0]
	}
	return doc[node.Raw.Offset : node.Raw.Offset+node.Raw.Length]
}

// rawValue returns the raw bytes of the value of a key-value expression.
func (d *decoder) rawValue(expr *unstable.Node, value *unstable.Node) []byte {
	if value.Kind != unstable.InlineTable && value.Kind != unstable.Array {
		return d.p.Raw(value.Raw)
	}
	if expr == nil || expr.Kind != unstable.KeyValue {
		// Inline container nested in another container: best effort.
		return d.p.Raw(value.Raw)
	}
	// Reconstruct the span of the value: it starts after the equal sign
	// following the last part of the key, and stops at the end of the
	// expression.
	var last unstable.Range
	it := expr.Key()
	for it.Next() {
		last = it.Node().Raw
	}
	doc := d.p.Data()
	i := int(last.Offset + last.Length)
	for i < len(doc) && (doc[i] == ' ' || doc[i] == '\t') {
		i++
	}
	i++ // equal sign
	for i < len(doc) && (doc[i] == ' ' || doc[i] == '\t') {
		i++
	}
	end := int(expr.Raw.Offset + expr.Raw.Length)
	return doc[i:end]
}

// unmarshalerOf returns the unstable.Unmarshaler implementation of v, if
// any. It allocates intermediate pointers as needed.
func unmarshalerOf(v reflect.Value) (unstable.Unmarshaler, bool) {
	t := v.Type()
	if t.Implements(unmarshalerType) {
		if v.Kind() == reflect.Ptr && v.IsNil() {
			v.Set(reflect.New(t.Elem()))
		}
		return v.Interface().(unstable.Unmarshaler), true
	}
	if v.CanAddr() && reflect.PtrTo(t).Implements(unmarshalerType) {
		return v.Addr().Interface().(unstable.Unmarshaler), true
	}
	return nil, false
}

var unmarshalerType = reflect.TypeOf(new(unstable.Unmarshaler)).Elem()

// assignValue stores the TOML value carried by the node into v.
func (d *decoder) assignValue(v reflect.Value, expr *unstable.Node, value *unstable.Node) (reflect.Value, error) {
	for v.Kind() == reflect.Ptr {
		if d.unmarshalerInterface {
			if u, ok := unmarshalerOf(v); ok {
				return v, u.UnmarshalTOML(d.rawValue(expr, value))
			}
		}
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		nv, err := d.assignValue(v.Elem(), expr, value)
		if err != nil || !nv.IsValid() {
			return reflect.Value{}, err
		}
		v.Elem().Set(nv)
		return v, nil
	}

	if d.unmarshalerInterface {
		if u, ok := unmarshalerOf(v); ok {
			return v, u.UnmarshalTOML(d.rawValue(expr, value))
		}
	}

	switch value.Kind {
	case unstable.String:
		return d.assignString(v, value)
	case unstable.Integer:
		return d.assignInteger(v, value)
	case unstable.Float:
		return d.assignFloat(v, value)
	case unstable.Bool:
		return d.assignBool(v, value)
	case unstable.DateTime:
		return d.assignDateTime(v, value)
	case unstable.LocalDateTime:
		return d.assignLocalDateTime(v, value)
	case unstable.LocalDate:
		return d.assignLocalDate(v, value)
	case unstable.LocalTime:
		return d.assignLocalTime(v, value)
	case unstable.Array:
		return d.assignArray(v, expr, value)
	case unstable.InlineTable:
		return d.assignInlineTable(v, expr, value)
	default:
		return reflect.Value{}, unstable.NewParserError(value.Data, "unsupported value kind %s", value.Kind)
	}
}

func (d *decoder) assignString(v reflect.Value, value *unstable.Node) (reflect.Value, error) {
	switch v.Kind() {
	case reflect.String:
		v.SetString(string(value.Data))
		return v, nil
	case reflect.Interface:
		return boxInto(v, reflect.ValueOf(string(value.Data)))
	}
	if v.CanAddr() && v.Addr().Type().Implements(textUnmarshalerType) {
		err := v.Addr().Interface().(encoding.TextUnmarshaler).UnmarshalText(value.Data)
		if err != nil {
			return reflect.Value{}, unstable.NewParserError(d.p.Raw(value.Raw), "%s", err)
		}
		return v, nil
	}
	return reflect.Value{}, d.typeMismatchError("string", v.Type(), d.p.Raw(value.Raw))
}

func (d *decoder) assignInteger(v reflect.Value, value *unstable.Node) (reflect.Value, error) {
	// Integer values targeting a float field are parsed as floats: they can
	// represent (approximately) numbers beyond the int64 range.
	if k := v.Kind(); k == reflect.Float32 || k == reflect.Float64 {
		return d.assignFloat(v, value)
	}

	i, err := parseInteger(value.Data)
	if err != nil {
		return reflect.Value{}, err
	}

	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if v.OverflowInt(i) {
			return reflect.Value{}, unstable.NewParserError(value.Data, "integer value %d cannot be stored in %s", i, v.Type())
		}
		v.SetInt(i)
		return v, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		if i < 0 {
			return reflect.Value{}, unstable.NewParserError(value.Data, "negative integer value %d cannot be stored in %s", i, v.Type())
		}
		if v.OverflowUint(uint64(i)) {
			return reflect.Value{}, unstable.NewParserError(value.Data, "integer value %d cannot be stored in %s", i, v.Type())
		}
		v.SetUint(uint64(i))
		return v, nil
	case reflect.Float32, reflect.Float64:
		v.SetFloat(float64(i))
		return v, nil
	case reflect.Interface:
		return boxInto(v, reflect.ValueOf(i))
	}
	if ok, err := tryTextUnmarshaler(v, value.Data); ok {
		return v, err
	}
	return reflect.Value{}, d.typeMismatchError("integer", v.Type(), d.p.Raw(value.Raw))
}

// tryTextUnmarshaler attempts to deliver the raw text of a value to a target
// implementing encoding.TextUnmarshaler.
func tryTextUnmarshaler(v reflect.Value, text []byte) (bool, error) {
	if v.CanAddr() && v.Addr().Type().Implements(textUnmarshalerType) {
		return true, v.Addr().Interface().(encoding.TextUnmarshaler).UnmarshalText(text)
	}
	return false, nil
}

func (d *decoder) assignFloat(v reflect.Value, value *unstable.Node) (reflect.Value, error) {
	f, err := parseFloat(value.Data)
	if err != nil {
		return reflect.Value{}, err
	}

	switch v.Kind() {
	case reflect.Float64:
		v.SetFloat(f)
		return v, nil
	case reflect.Float32:
		if !math.IsInf(f, 0) && math.Abs(f) > math.MaxFloat32 {
			return reflect.Value{}, unstable.NewParserError(value.Data, "float value %f cannot be stored in float32", f)
		}
		v.SetFloat(f)
		return v, nil
	case reflect.Interface:
		return boxInto(v, reflect.ValueOf(f))
	}
	if ok, err := tryTextUnmarshaler(v, value.Data); ok {
		return v, err
	}
	return reflect.Value{}, d.typeMismatchError("float", v.Type(), d.p.Raw(value.Raw))
}

func (d *decoder) assignBool(v reflect.Value, value *unstable.Node) (reflect.Value, error) {
	b := value.Data[0] == 't'

	switch v.Kind() {
	case reflect.Bool:
		v.SetBool(b)
		return v, nil
	case reflect.Interface:
		return boxInto(v, reflect.ValueOf(b))
	}
	if ok, err := tryTextUnmarshaler(v, value.Data); ok {
		return v, err
	}
	return reflect.Value{}, d.typeMismatchError("boolean", v.Type(), d.p.Raw(value.Raw))
}

func (d *decoder) assignDateTime(v reflect.Value, value *unstable.Node) (reflect.Value, error) {
	t, err := parseDateTime(value.Data)
	if err != nil {
		return reflect.Value{}, err
	}

	if v.Type() == timeType {
		v.Set(reflect.ValueOf(t))
		return v, nil
	}
	if v.Kind() == reflect.Interface {
		return boxInto(v, reflect.ValueOf(t))
	}
	return reflect.Value{}, d.typeMismatchError("datetime", v.Type(), d.p.Raw(value.Raw))
}

func (d *decoder) assignLocalDateTime(v reflect.Value, value *unstable.Node) (reflect.Value, error) {
	dt, rest, err := parseLocalDateTime(value.Data)
	if err != nil {
		return reflect.Value{}, err
	}
	if len(rest) > 0 {
		return reflect.Value{}, unstable.NewParserError(rest, "extra characters at the end of a local date time")
	}

	switch v.Type() {
	case localDateTimeType:
		v.Set(reflect.ValueOf(dt))
		return v, nil
	case timeType:
		v.Set(reflect.ValueOf(dt.AsTime(time.Local)))
		return v, nil
	}
	if v.Kind() == reflect.Interface {
		return boxInto(v, reflect.ValueOf(dt))
	}
	return reflect.Value{}, d.typeMismatchError("local datetime", v.Type(), d.p.Raw(value.Raw))
}

func (d *decoder) assignLocalDate(v reflect.Value, value *unstable.Node) (reflect.Value, error) {
	date, err := parseLocalDate(value.Data)
	if err != nil {
		return reflect.Value{}, err
	}

	switch v.Type() {
	case localDateType:
		v.Set(reflect.ValueOf(date))
		return v, nil
	case timeType:
		v.Set(reflect.ValueOf(date.AsTime(time.Local)))
		return v, nil
	}
	if v.Kind() == reflect.Interface {
		return boxInto(v, reflect.ValueOf(date))
	}
	return reflect.Value{}, d.typeMismatchError("local date", v.Type(), d.p.Raw(value.Raw))
}

func (d *decoder) assignLocalTime(v reflect.Value, value *unstable.Node) (reflect.Value, error) {
	t, rest, err := parseLocalTime(value.Data)
	if err != nil {
		return reflect.Value{}, err
	}
	if len(rest) > 0 {
		return reflect.Value{}, unstable.NewParserError(rest, "extra characters at the end of a local time")
	}

	switch v.Type() {
	case localTimeType:
		v.Set(reflect.ValueOf(t))
		return v, nil
	case timeType:
		v.Set(reflect.ValueOf(time.Date(0, 1, 1, t.Hour, t.Minute, t.Second, t.Nanosecond, time.Local)))
		return v, nil
	}
	if v.Kind() == reflect.Interface {
		return boxInto(v, reflect.ValueOf(t))
	}
	return reflect.Value{}, d.typeMismatchError("local time", v.Type(), d.p.Raw(value.Raw))
}

func (d *decoder) assignArray(v reflect.Value, expr *unstable.Node, value *unstable.Node) (reflect.Value, error) {
	switch v.Kind() {
	case reflect.Slice:
		elemType := v.Type().Elem()
		slice := reflect.MakeSlice(v.Type(), 0, 4)
		it := value.Children()
		for it.Next() {
			n := it.Node()
			if n.Kind == unstable.Comment {
				continue
			}
			elem := reflect.New(elemType).Elem()
			nv, err := d.assignValue(elem, nil, n)
			if err != nil {
				return reflect.Value{}, err
			}
			slice = reflect.Append(slice, nv)
		}
		return slice, nil
	case reflect.Array:
		it := value.Children()
		i := 0
		for it.Next() {
			n := it.Node()
			if n.Kind == unstable.Comment {
				continue
			}
			if i >= v.Len() {
				// Extra elements are dropped when the target array is too
				// small.
				break
			}
			elem := v.Index(i)
			nv, err := d.assignValue(elem, nil, n)
			if err != nil {
				return reflect.Value{}, err
			}
			elem.Set(nv)
			i++
		}
		return v, nil
	case reflect.Interface:
		slice := []interface{}{}
		it := value.Children()
		for it.Next() {
			n := it.Node()
			if n.Kind == unstable.Comment {
				continue
			}
			elem := reflect.New(interfaceType).Elem()
			nv, err := d.assignValue(elem, nil, n)
			if err != nil {
				return reflect.Value{}, err
			}
			slice = append(slice, nv.Interface())
		}
		return boxInto(v, reflect.ValueOf(slice))
	}
	return reflect.Value{}, d.typeMismatchError("array", v.Type(), d.rawValue(expr, value))
}

func (d *decoder) assignInlineTable(v reflect.Value, expr *unstable.Node, value *unstable.Node) (reflect.Value, error) {
	switch v.Kind() {
	case reflect.Map:
		// Inline tables are self-contained: they fully replace the target.
		v = reflect.MakeMap(v.Type())
	case reflect.Struct:
		// fields are set in place
	case reflect.Interface:
		elem := reflect.ValueOf(map[string]interface{}{})
		nv, err := d.assignInlineTable(elem, expr, value)
		if err != nil {
			return reflect.Value{}, err
		}
		return boxInto(v, nv)
	default:
		return reflect.Value{}, d.typeMismatchError("inline table", v.Type(), d.rawValue(expr, value))
	}

	it := value.Children()
	for it.Next() {
		kv := it.Node()
		// Build the path from the key of this key-value.
		path := make([]pathPart, 0, 2)
		kit := kv.Key()
		for kit.Next() {
			n := kit.Node()
			path = append(path, pathPart{name: string(n.Data), node: n})
		}
		nv, err := d.descend(v, path, 0, kv, kv.Value())
		if err != nil {
			return reflect.Value{}, err
		}
		if nv.IsValid() {
			v = nv
		}
	}
	return v, nil
}

// boxInto stores the concrete value c into the interface value v.
func boxInto(v reflect.Value, c reflect.Value) (reflect.Value, error) {
	boxed := reflect.New(v.Type()).Elem()
	if !c.Type().AssignableTo(v.Type()) {
		return reflect.Value{}, fmt.Errorf("toml: cannot store %s into %s", c.Type(), v.Type())
	}
	boxed.Set(c)
	return boxed, nil
}

var interfaceType = reflect.TypeOf(new(interface{})).Elem()
var localDateType = reflect.TypeOf(LocalDate{})
var localTimeType = reflect.TypeOf(LocalTime{})
var localDateTimeType = reflect.TypeOf(LocalDateTime{})

// structPlan caches the mapping between TOML keys and the fields of a struct
// type.
type structPlan struct {
	byName map[string]structField
	byFold map[string]structField
}

type structField struct {
	index     []int
	fieldName string
}

func (p *structPlan) lookup(name string) (structField, bool) {
	f, ok := p.byName[name]
	if ok {
		return f, true
	}
	f, ok = p.byFold[strings.ToLower(name)]
	return f, ok
}

var structPlans sync.Map // reflect.Type -> *structPlan

func planForType(t reflect.Type) *structPlan {
	if plan, ok := structPlans.Load(t); ok {
		return plan.(*structPlan)
	}
	plan := buildPlan(t)
	structPlans.Store(t, plan)
	return plan
}

func buildPlan(t reflect.Type) *structPlan {
	plan := &structPlan{
		byName: map[string]structField{},
		byFold: map[string]structField{},
	}
	addFields(plan, t, nil)
	return plan
}

func addFields(plan *structPlan, t reflect.Type, prefix []int) {
	var embedded []reflect.StructField
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag, tagged := f.Tag.Lookup("toml")
		name := f.Name
		if tagged {
			// A tag of exactly "-" drops the field. "-," names it "-".
			if tag == "-" {
				continue
			}
			parts := strings.SplitN(tag, ",", 2)
			if parts[0] != "" {
				name = parts[0]
			}
		}
		if f.Anonymous {
			ft := f.Type
			if ft.Kind() == reflect.Ptr {
				ft = ft.Elem()
			}
			if ft.Kind() != reflect.Struct {
				// Embedded non-struct fields are not decoded into.
				continue
			}
			if !tagged {
				// Untagged embedded structs are flattened, even when their
				// type is unexported: only their own exported fields are
				// reachable.
				embedded = append(embedded, f)
				continue
			}
			// A tagged embedded struct acts as a regular named field.
		} else if f.PkgPath != "" {
			// unexported
			continue
		}
		index := make([]int, 0, len(prefix)+1)
		index = append(index, prefix...)
		index = append(index, i)
		sf := structField{index: index, fieldName: f.Name}
		if _, ok := plan.byName[name]; !ok {
			plan.byName[name] = sf
		}
		lower := strings.ToLower(name)
		if _, ok := plan.byFold[lower]; !ok {
			plan.byFold[lower] = sf
		}
	}
	// Embedded structs are flattened after the regular fields, so that
	// shallower fields win.
	for _, f := range embedded {
		ft := f.Type
		if ft.Kind() == reflect.Ptr {
			ft = ft.Elem()
		}
		index := make([]int, 0, len(prefix)+1)
		index = append(index, prefix...)
		idx := f.Index[0]
		index = append(index, idx)
		addFields(plan, ft, index)
	}
}

// fieldByIndexAlloc returns the field of v at the given index path,
// allocating intermediate embedded pointers as needed.
func fieldByIndexAlloc(v reflect.Value, index []int) reflect.Value {
	for i, x := range index {
		if i > 0 {
			for v.Kind() == reflect.Ptr {
				if v.IsNil() {
					v.Set(reflect.New(v.Type().Elem()))
				}
				v = v.Elem()
			}
		}
		v = v.Field(x)
	}
	return v
}
