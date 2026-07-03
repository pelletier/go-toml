package toml

import (
	"errors"
	"reflect"

	"github.com/pelletier/go-toml/v2/internal/parserbridge"
	"github.com/pelletier/go-toml/v2/unstable"
)

// fusedStructDocument drives the top-level expression loop for reflection
// targets (structs, typed maps, ...) without building an AST for scalar
// key-values, which are the bulk of a document: their key and value are
// scanned directly and dispatched through the same descend/assign machinery
// as the AST path. Table headers and container values still go through the
// expression parser, so array-table bookkeeping, strict-mode reporting and
// error rendering stay identical.
//
// It is used whenever the unmarshaler interface is disabled (captures need
// the raw expression ranges of the AST); fully generic targets take the
// unmarshalFused path instead.
func (d *decoder) fusedStructDocument(root reflect.Value, data []byte) error {
	b := data
	for {
		b = fusedSkipWS(b)
		if len(b) == 0 {
			return nil
		}
		switch b[0] {
		case '\n':
			b = b[1:]
		case '\r':
			if len(b) > 1 && b[1] == '\n' {
				b = b[2:]
				continue
			}
			return d.wrapFusedError(data, unstable.NewParserError(b[:1], "expected newline but got %#U", b[0]))
		case '#':
			_, rest, err := parserbridge.ScanComment(b)
			if err == nil {
				rest, err = fusedConsumeEOL(rest)
			}
			if err != nil {
				return d.wrapFusedError(data, err)
			}
			b = rest
		case '[':
			// Table headers run through the expression parser and the
			// regular handling (seen-tracking, strict bookkeeping, walk).
			parserbridge.SetCursor(&d.p, b)
			if !d.p.NextExpression() {
				if err := d.p.Error(); err != nil {
					var perr *unstable.ParserError
					if errors.As(err, &perr) {
						return wrapDecodeError(data, perr)
					}
					return err
				}
				return nil
			}
			if err := d.handleRootExpression(d.p.Expression(), root); err != nil {
				return d.wrapError(data, err)
			}
			b = parserbridge.Cursor(&d.p)
		default:
			rest, err := d.fusedStructKeyVal(b, root)
			if err != nil {
				return d.wrapFusedError(data, err)
			}
			b = rest
		}
	}
}

// fusedStructKeyVal handles one `key = value` expression for a reflection
// target. b starts at the first character of the key. Scalar values are
// scanned without any AST node; container values are parsed into the arena
// as usual, so typed arrays and inline tables decode identically to the AST
// path.
func (d *decoder) fusedStructKeyVal(b []byte, root reflect.Value) ([]byte, error) {
	var err error
	var rawKey []byte
	d.keyParts, d.keyRaws, rawKey, b, err = parserbridge.ScanKeyRaws(&d.p, b, d.keyParts[:0], d.keyRaws[:0])
	if err != nil {
		return nil, err
	}
	if len(b) == 0 || b[0] != '=' {
		return nil, unstable.NewParserError(fusedHL1(b), "expected '=' after key")
	}
	b = fusedSkipWS(b[1:])
	if len(b) == 0 {
		return nil, unstable.NewParserError(b, "expected value, not end of input")
	}

	if c := b[0]; c == '[' || c == '{' {
		valStart := b
		nodeAny, rest, err := parserbridge.ParseValue(&d.p, b)
		if err != nil {
			return nil, err
		}
		node := nodeAny.(*unstable.Node)
		valSpan := valStart[:len(valStart)-len(rest)]
		rest, err = d.fusedFinishLine(rest)
		if err != nil {
			return nil, err
		}
		leafID, err := d.seen.CheckKeyValue(d.keyParts)
		if err != nil {
			return nil, d.fusedSeenError(rawKey, d.keyParts, err)
		}
		if err := d.seen.CheckValueUnder(leafID, node); err != nil {
			return nil, d.fusedSeenError(rawKey, d.keyParts, err)
		}
		if d.skipUntilTable {
			return rest, nil
		}
		d.fusedValueSpan = valSpan
		err = d.fusedHandleKeyValue(root, rawKey, node)
		d.fusedValueSpan = nil
		return rest, err
	}

	// Scalar value: no AST at all. The stack node carries the same kind,
	// data, and raw range an arena node would; nothing retains it.
	k, rawVal, value, rest, err := parserbridge.ScanScalar(&d.p, b)
	if err != nil {
		return nil, err
	}
	rest, err = d.fusedFinishLine(rest)
	if err != nil {
		return nil, err
	}
	if _, err := d.seen.CheckKeyValue(d.keyParts); err != nil {
		return nil, d.fusedSeenError(rawKey, d.keyParts, err)
	}
	if d.skipUntilTable {
		return rest, nil
	}
	node := unstable.Node{
		Kind: unstable.Kind(k),
		Raw:  d.p.Range(rawVal),
		Data: value,
	}
	return rest, d.fusedHandleKeyValue(root, rawKey, &node)
}

// fusedHandleKeyValue mirrors handleKeyValueExpression, driven by the scanned
// key parts (d.keyParts/d.keyRaws) instead of an expression node.
func (d *decoder) fusedHandleKeyValue(root reflect.Value, rawKey []byte, value *unstable.Node) error {
	d.path = d.path[:0]

	target := root
	useCache := d.tableTargetValid && len(d.tableKey) > 0
	if useCache {
		target = d.tableTarget
	} else {
		for _, name := range d.tableKey {
			d.path = append(d.path, pathPart{name: name})
		}
	}
	for i := range d.keyParts {
		d.path = append(d.path, pathPart{data: d.keyParts[i], rng: d.p.Range(d.keyRaws[i])})
	}
	d.fusedKVParts = d.keyParts
	d.fusedKVKeyRange = d.p.Range(rawKey)

	nv, err := d.descend(target, d.path, 0, nil, value)
	if err != nil {
		return d.contextualizeError(err, useCache)
	}
	if !nv.IsValid() {
		return nil
	}
	if useCache {
		// The target may have been replaced (e.g. a nil map allocated):
		// re-link it into its parent.
		if nv.Kind() == reflect.Map && nv.Pointer() != d.tableTarget.Pointer() {
			d.storeSlot(&d.tableParentSlot, nv)
			d.tableTarget = nv
		}
	} else {
		if root.CanSet() {
			root.Set(nv)
		}
	}
	return nil
}
