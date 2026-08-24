package model

import "encoding/json"

// Ref is an idRef: a same-namespace identifier string or an explicitly namespaced identifier.
// Its JSON codec is the one behavior in this package, because encoding/json requires it on the type.
type Ref struct {
	Namespace string
	ID        string
}

func (r *Ref) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		*r = Ref{ID: s}
		return nil
	}
	var o struct {
		Namespace string `json:"namespace"`
		ID        string `json:"id"`
	}
	if err := json.Unmarshal(b, &o); err != nil {
		return err
	}
	*r = Ref{Namespace: o.Namespace, ID: o.ID}
	return nil
}

func (r Ref) MarshalJSON() ([]byte, error) {
	if r.Namespace == "" {
		return json.Marshal(r.ID)
	}
	return json.Marshal(map[string]string{"namespace": r.Namespace, "id": r.ID})
}
