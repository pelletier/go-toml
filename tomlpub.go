package toml

// PubTOMLValue wrapping tomlValue in order to access all properties from outside.
type PubTOMLValue = tomlValue

func (ptv *PubTOMLValue) Value() interface{} {
	return ptv.value
}
func (ptv *PubTOMLValue) Comment() string {
	return ptv.comment
}
func (ptv *PubTOMLValue) Commented() bool {
	return ptv.commented
}
func (ptv *PubTOMLValue) Multiline() bool {
	return ptv.multiline
}
func (ptv *PubTOMLValue) Position() Position {
	return ptv.position
}

// PubTree wrapping Tree in order to access all properties from outside.
type PubTree = Tree

func (pt *PubTree) Values() map[string]interface{} {
	return pt.values
}

func (pt *PubTree) Comment() string {
	return pt.comment
}

func (pt *PubTree) Commented() bool {
	return pt.commented
}

func (pt *PubTree) Inline() bool {
	return pt.inline
}
