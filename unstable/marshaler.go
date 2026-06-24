package unstable

// Marshaler is implemented by types that can marshal themselves into a raw
// TOML description. The returned bytes are spliced verbatim into the encoded
// document, so they must be valid TOML for the position they end up in:
//
//   - A single value (string, integer, array, inline table, …) is emitted
//     inline, as in `key = <raw>`.
//   - One or more key-value lines (optionally with relative sub-table headers)
//     is emitted as the body of a `[key]` table.
//
// The encoder decides between those two forms by parsing the returned bytes;
// see Encoder.EnableMarshalerInterface in the root toml package.
//
// Marshaler is the encoding counterpart of Unmarshaler.
type Marshaler interface {
	MarshalTOML() ([]byte, error)
}

// MarshalTOML implements Marshaler. It returns the raw TOML bytes verbatim,
// mirroring json.RawMessage.MarshalJSON. The value receiver means both
// RawMessage and *RawMessage satisfy Marshaler.
func (m RawMessage) MarshalTOML() ([]byte, error) {
	return []byte(m), nil
}
