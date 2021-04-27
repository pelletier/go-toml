package toml

import (
	"encoding"
	"fmt"
	"io"
	"io/ioutil"
	"reflect"
	"time"

	"github.com/pelletier/go-toml/v2/internal/ast"
	"github.com/pelletier/go-toml/v2/internal/tracker"
	"github.com/pelletier/go-toml/v2/internal/unsafe"
)

func Unmarshal(data []byte, v interface{}) error {
	p := parser{}
	p.Reset(data)
	d := decoder{}

	return d.FromParser(&p, v)
}

// Decoder reads and decode a TOML document from an input stream.
type Decoder struct {
	// input
	r io.Reader

	// global settings
	strict bool
}

// NewDecoder creates a new Decoder that will read from r.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: r}
}

// SetStrict toggles decoding in stict mode.
//
// When the decoder is in strict mode, it will record fields from the document
// that could not be set on the target value. In that case, the decoder returns
// a StrictMissingError that can be used to retrieve the individual errors as
// well as generate a human readable description of the missing fields.
func (d *Decoder) SetStrict(strict bool) {
	d.strict = strict
}

// Decode the whole content of r into v.
//
// When a TOML local date is decoded into a time.Time, its value is represented
// in time.Local timezone.
//
// Empty tables decoded in an interface{} create an empty initialized
// map[string]interface{}.
func (d *Decoder) Decode(v interface{}) error {
	b, err := ioutil.ReadAll(d.r)
	if err != nil {
		return err
	}

	p := parser{}
	p.Reset(b)
	dec := decoder{
		strict: strict{
			Enabled: d.strict,
		},
	}

	return dec.FromParser(&p, v)
}

type decoder struct {
	// Tracks position in Go arrays.
	arrayIndexes map[reflect.Value]int

	// Tracks keys that have been seen, with which type.
	seen tracker.SeenTracker

	// Strict mode
	strict strict
}

func (d *decoder) arrayIndex(append bool, v reflect.Value) int {
	if d.arrayIndexes == nil {
		d.arrayIndexes = make(map[reflect.Value]int, 1)
	}

	idx, ok := d.arrayIndexes[v]

	if !ok {
		d.arrayIndexes[v] = 0
	} else if append {
		idx++
		d.arrayIndexes[v] = idx
	}

	return idx
}

func (d *decoder) FromParser(p *parser, v interface{}) error {
	err := d.fromParser(p, v)
	if err != nil {
		de, ok := err.(*decodeError)
		if ok {
			err = wrapDecodeError(p.data, de)
		}
	}

	if err == nil {
		err = d.strict.Error(p.data)
	}

	return err
}

func keyLocation(node ast.Node) []byte {
	k := node.Key()

	hasOne := k.Next()
	if !hasOne {
		panic("should not be called with empty key")
	}

	start := k.Node().Data
	end := k.Node().Data

	for k.Next() {
		end = k.Node().Data
	}

	return unsafe.BytesRange(start, end)
}

func (d *decoder) fromParser(p *parser, v interface{}) error {
	r := reflect.ValueOf(v)
	if r.Kind() != reflect.Ptr {
		return fmt.Errorf("need to target a pointer, not %s", r.Kind())
	}

	if r.IsNil() {
		return fmt.Errorf("target pointer must be non-nil")
	}

	var (
		skipUntilTable bool
		root           target = valueTarget(r.Elem())
	)

	current := root

	for p.NextExpression() {
		node := p.Expression()

		if node.Kind == ast.KeyValue && skipUntilTable {
			continue
		}

		err := d.seen.CheckExpression(node)
		if err != nil {
			return err
		}

		var found bool

		switch node.Kind {
		case ast.KeyValue:
			err = d.unmarshalKeyValue(current, node)
			found = true
		case ast.Table:
			d.strict.EnterTable(node)
			current, found, err = d.scopeWithKey(root, node.Key())
			if err == nil && found {
				// In case this table points to an interface,
				// make sure it at least holds something that
				// looks like a table. Otherwise the information
				// of a table is lost, and marshal cannot do the
				// round trip.
				ensureMapIfInterface(current)
			}
		case ast.ArrayTable:
			d.strict.EnterArrayTable(node)
			current, found, err = d.scopeWithArrayTable(root, node.Key())
		default:
			panic(fmt.Errorf("this should not be a top level node type: %s", node.Kind))
		}

		if err != nil {
			return err
		}

		if !found {
			skipUntilTable = true

			d.strict.MissingTable(node)
		}
	}

	return p.Error()
}

// scopeWithKey performs target scoping when unmarshaling an ast.KeyValue node.
//
// The goal is to hop from target to target recursively using the names in key.
// Parts of the key should be used to resolve field names for structs, and as
// keys when targeting maps.
//
// When encountering slices, it should always use its last element, and error
// if the slice does not have any.
func (d *decoder) scopeWithKey(x target, key ast.Iterator) (target, bool, error) {
	var err error
	found := true

	for key.Next() {
		n := key.Node()

		x, found, err = d.scopeTableTarget(false, x, string(n.Data))
		if err != nil || !found {
			return nil, found, err
		}
	}

	return x, true, nil
}

// scopeWithArrayTable performs target scoping when unmarshaling an
// ast.ArrayTable node.
//
// It is the same as scopeWithKey, but when scoping the last part of the key
// it creates a new element in the array instead of using the last one.
func (d *decoder) scopeWithArrayTable(x target, key ast.Iterator) (target, bool, error) {
	var err error

	found := true

	for key.Next() {
		n := key.Node()
		if !n.Next().Valid() { // want to stop at one before last
			break
		}

		x, found, err = d.scopeTableTarget(false, x, string(n.Data))
		if err != nil || !found {
			return nil, found, err
		}
	}

	n := key.Node()

	x, found, err = d.scopeTableTarget(false, x, string(n.Data))
	if err != nil || !found {
		return x, found, err
	}

	v := x.get()

	if v.Kind() == reflect.Ptr {
		x, err = scopePtr(x)
		if err != nil {
			return x, false, err
		}

		v = x.get()
	}

	if v.Kind() == reflect.Interface {
		x, err = scopeInterface(true, x)
		if err != nil {
			return x, found, err
		}

		v = x.get()
	}

	switch v.Kind() {
	case reflect.Slice:
		x, err = scopeSlice(true, x)
	case reflect.Array:
		x, err = d.scopeArray(true, x)
	}

	return x, found, err
}

func (d *decoder) unmarshalKeyValue(x target, node ast.Node) error {
	assertNode(ast.KeyValue, node)

	d.strict.EnterKeyValue(node)
	defer d.strict.ExitKeyValue(node)

	x, found, err := d.scopeWithKey(x, node.Key())
	if err != nil {
		return err
	}

	// A struct in the path was not found. Skip this value.
	if !found {
		d.strict.MissingField(node)
		return nil
	}

	return d.unmarshalValue(x, node.Value())
}

var textUnmarshalerType = reflect.TypeOf(new(encoding.TextUnmarshaler)).Elem()

func tryTextUnmarshaler(x target, node ast.Node) (bool, error) {
	v := x.get()

	if v.Kind() != reflect.Struct {
		return false, nil
	}

	// Special case for time, because we allow to unmarshal to it from
	// different kind of AST nodes.
	if v.Type() == timeType {
		return false, nil
	}

	if v.Type().Implements(textUnmarshalerType) {
		return true, v.Interface().(encoding.TextUnmarshaler).UnmarshalText(node.Data)
	}

	if v.CanAddr() && v.Addr().Type().Implements(textUnmarshalerType) {
		return true, v.Addr().Interface().(encoding.TextUnmarshaler).UnmarshalText(node.Data)
	}

	return false, nil
}

func (d *decoder) unmarshalValue(x target, node ast.Node) error {
	v := x.get()

	if v.Kind() == reflect.Ptr {
		if !v.Elem().IsValid() {
			err := x.set(reflect.New(v.Type().Elem()))
			if err != nil {
				return err
			}

			v = x.get()
		}

		return d.unmarshalValue(valueTarget(v.Elem()), node)
	}

	ok, err := tryTextUnmarshaler(x, node)
	if ok {
		return err
	}

	switch node.Kind {
	case ast.String:
		return unmarshalString(x, node)
	case ast.Bool:
		return unmarshalBool(x, node)
	case ast.Integer:
		return unmarshalInteger(x, node)
	case ast.Float:
		return unmarshalFloat(x, node)
	case ast.Array:
		return d.unmarshalArray(x, node)
	case ast.InlineTable:
		return d.unmarshalInlineTable(x, node)
	case ast.LocalDateTime:
		return unmarshalLocalDateTime(x, node)
	case ast.DateTime:
		return unmarshalDateTime(x, node)
	case ast.LocalDate:
		return unmarshalLocalDate(x, node)
	default:
		panic(fmt.Errorf("unhandled unmarshalValue kind %s", node.Kind))
	}
}

func unmarshalLocalDate(x target, node ast.Node) error {
	assertNode(ast.LocalDate, node)

	v, err := parseLocalDate(node.Data)
	if err != nil {
		return err
	}

	return setDate(x, v)
}

func unmarshalLocalDateTime(x target, node ast.Node) error {
	assertNode(ast.LocalDateTime, node)

	v, rest, err := parseLocalDateTime(node.Data)
	if err != nil {
		return err
	}

	if len(rest) > 0 {
		return newDecodeError(rest, "extra characters at the end of a local date time")
	}

	return setLocalDateTime(x, v)
}

func unmarshalDateTime(x target, node ast.Node) error {
	assertNode(ast.DateTime, node)

	v, err := parseDateTime(node.Data)
	if err != nil {
		return err
	}

	return setDateTime(x, v)
}

func setLocalDateTime(x target, v LocalDateTime) error {
	if x.get().Type() == timeType {
		cast := v.In(time.Local)
		return setDateTime(x, cast)
	}

	return x.set(reflect.ValueOf(v))
}

func setDateTime(x target, v time.Time) error {
	return x.set(reflect.ValueOf(v))
}

var timeType = reflect.TypeOf(time.Time{})

func setDate(x target, v LocalDate) error {
	if x.get().Type() == timeType {
		cast := v.In(time.Local)
		return setDateTime(x, cast)
	}

	return x.set(reflect.ValueOf(v))
}

func unmarshalString(x target, node ast.Node) error {
	assertNode(ast.String, node)

	return setString(x, string(node.Data))
}

func unmarshalBool(x target, node ast.Node) error {
	assertNode(ast.Bool, node)
	v := node.Data[0] == 't'

	return setBool(x, v)
}

func unmarshalInteger(x target, node ast.Node) error {
	assertNode(ast.Integer, node)

	v, err := parseInteger(node.Data)
	if err != nil {
		return err
	}

	return setInt64(x, v)
}

func unmarshalFloat(x target, node ast.Node) error {
	assertNode(ast.Float, node)
	v, err := parseFloat(node.Data)
	if err != nil {
		return err
	}

	return setFloat64(x, v)
}

func (d *decoder) unmarshalInlineTable(x target, node ast.Node) error {
	assertNode(ast.InlineTable, node)

	ensureMapIfInterface(x)

	it := node.Children()
	for it.Next() {
		n := it.Node()

		err := d.unmarshalKeyValue(x, n)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *decoder) unmarshalArray(x target, node ast.Node) error {
	assertNode(ast.Array, node)

	err := ensureValueIndexable(x)
	if err != nil {
		return err
	}

	idx := 0

	it := node.Children()
	for it.Next() {
		n := it.Node()

		v, err := elementAt(x, idx)
		if err != nil {
			return err
		}

		if v == nil {
			// when we go out of bound for an array just stop processing it to
			// mimic encoding/json
			break
		}

		err = d.unmarshalValue(v, n)
		if err != nil {
			return err
		}

		idx++
	}

	return nil
}

func assertNode(expected ast.Kind, node ast.Node) {
	if node.Kind != expected {
		panic(fmt.Errorf("expected node of kind %s, not %s", expected, node.Kind))
	}
}
