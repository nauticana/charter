package model

type Limit struct {
	LimitKind        string `json:"limitKind"`
	Operator         string `json:"operator"`
	Value            any    `json:"value"`
	Unit             string `json:"unit,omitempty"`
	Currency         string `json:"currency,omitempty"`
	CurrencyExponent *int   `json:"currencyExponent,omitempty"`
}
