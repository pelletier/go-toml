package toml

import "github.com/pelletier/go-toml/v2/internal/ast"

type strict struct {
	Enabled bool

	missing []decodeError
}

func (s *strict) MissingTable(node ast.Node) {
	if !s.Enabled {
		return
	}
	s.missing = append(s.missing, decodeError{
		highlight: keyLocation(node),
		message:   "missing table",
	})
}

func (s *strict) MissingField(node ast.Node) {
	if !s.Enabled {
		return
	}
	s.missing = append(s.missing, decodeError{
		highlight: keyLocation(node),
		message:   "missing field",
	})
}

func (s *strict) Error() *StrictMissingError {
	if !s.Enabled {
		return nil
	}

	err := &StrictMissingError{
		Errors: make([]DecodeError, 0, len(s.missing)),
	}
	for _, derr := range s.missing {
		err.Errors = append(err.Errors, *wrapDecodeError(p.data, &derr))
	}
	return err
}
