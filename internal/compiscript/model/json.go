package model

import "encoding/json"

func (v Types) MarshalJSON() ([]byte, error) {
	if v == nil {
		v = Types{}
	}
	return json.Marshal([]Type(v))
}
func (v Symbols) MarshalJSON() ([]byte, error) {
	if v == nil {
		v = Symbols{}
	}
	return json.Marshal([]Symbol(v))
}
func (v ScopeSnapshots) MarshalJSON() ([]byte, error) {
	if v == nil {
		v = ScopeSnapshots{}
	}
	return json.Marshal([]ScopeSnapshot(v))
}
func (v Diagnostics) MarshalJSON() ([]byte, error) {
	if v == nil {
		v = Diagnostics{}
	}
	return json.Marshal([]Diagnostic(v))
}
func (v ASTViews) MarshalJSON() ([]byte, error) {
	if v == nil {
		v = ASTViews{}
	}
	return json.Marshal([]ASTView(v))
}
