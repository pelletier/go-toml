package toml

// PubTOMLValue wrapping tomlValue in order to access all properties from outside.
type PubTOMLValue = tomlValue

// Value returns the property Value of tomlValue.
func (ptv *PubTOMLValue) Value() interface{} {
	return ptv.value
}

// Comment returns the property Comment of tomlValue.
func (ptv *PubTOMLValue) Comment() string {
	return ptv.comment
}

// Commented returns the property Commented of tomlValue.
func (ptv *PubTOMLValue) Commented() bool {
	return ptv.commented
}

// Multiline returns the property Multiline of tomlValue.
func (ptv *PubTOMLValue) Multiline() bool {
	return ptv.multiline
}

// Position returns the property Position of tomlValue.
func (ptv *PubTOMLValue) Position() Position {
	return ptv.position
}

// SetValue returns the property SetValue of tomlValue.
func (ptv *PubTOMLValue) SetValue(v interface{}) {
	ptv.value = v
}

// SetComment returns the property SetComment of tomlValue.
func (ptv *PubTOMLValue) SetComment(s string) {
	ptv.comment = s
}

// SetCommented returns the property SetCommented of tomlValue.
func (ptv *PubTOMLValue) SetCommented(c bool) {
	ptv.commented = c
}

// SetMultiline returns the property SetMultiline of tomlValue.
func (ptv *PubTOMLValue) SetMultiline(m bool) {
	ptv.multiline = m
}

// SetPosition returns the property SetPosition of tomlValue.
func (ptv *PubTOMLValue) SetPosition(p Position) {
	ptv.position = p
}

// PubTree wrapping Tree in order to access all properties from outside.
type PubTree = Tree

// Values return the property Values of Tree.
func (t *PubTree) Values() map[string]interface{} {
	return t.values
}

// Comment return the property Comment of Tree.
func (t *PubTree) Comment() string {
	return t.comment
}

// Commented return the property Commented of Tree.
func (t *PubTree) Commented() bool {
	return t.commented
}

// Inline return the property Inline of Tree.
func (t *PubTree) Inline() bool {
	return t.inline
}

// SetValues return the property SetValues of Tree.
func (t *PubTree) SetValues(v map[string]interface{}) {
	t.values = v
}

// SetComment return the property SetComment of Tree.
func (t *PubTree) SetComment(c string) {
	t.comment = c
}

// SetCommented return the property SetCommented of Tree.
func (t *PubTree) SetCommented(c bool) {
	t.commented = c
}

// SetInline return the property SetInline of Tree.
func (t *PubTree) SetInline(i bool) {
	t.inline = i
}
