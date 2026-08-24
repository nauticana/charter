package model

type Validity struct {
	From Date `json:"from"`
	To   Date `json:"to,omitempty"`
}
