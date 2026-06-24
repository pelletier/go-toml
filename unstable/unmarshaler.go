package unstable

// Unmarshaler is implemented by types that can unmarshal a TOML description
// of themselves. The input is a valid TOML document containing the relevant
// portion of the parsed document.
//
// For tables (including split tables defined in multiple places), the data
// contains the raw key-value bytes from the original document with adjusted
// table headers to be relative to the unmarshaling target.
//
// When the decoding target itself implements this interface, it receives the
// whole document — every top-level key-value as well as every table and array
// table — assembled into a single valid TOML document and delivered once.
type Unmarshaler interface {
	UnmarshalTOML(data []byte) error
}

// RawMessage is a raw encoded TOML value. It implements Unmarshaler and can
// be used to delay TOML decoding or capture raw content.
//
// Example usage:
//
//	type Config struct {
//	    Plugin RawMessage `toml:"plugin"`
//	}
//
//	var cfg Config
//	toml.NewDecoder(r).EnableUnmarshalerInterface().Decode(&cfg)
//	// cfg.Plugin now contains the raw TOML bytes for [plugin]
type RawMessage []byte

// UnmarshalTOML implements Unmarshaler.
func (m *RawMessage) UnmarshalTOML(data []byte) error {
	*m = append((*m)[:0], data...)
	return nil
}
