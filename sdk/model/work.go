package model

// WorkClassification is one version of an external classification of work, such as an occupation or skill taxonomy,
// whose concepts responsibilities, tasks, and capabilities cite (CHR-ENT-014).
type WorkClassification struct {
	Envelope
	Publisher string `json:"publisher"`
	Version   string `json:"version"`
	URL       string `json:"url,omitempty"`
	License   string `json:"license,omitempty"`
}

// WorkConcept is one term of a classification: its identifier there, its labels, and its place among the
// classification's other concepts.
type WorkConcept struct {
	Envelope
	ClassificationID  Ref     `json:"classificationId"`
	ExternalID        string  `json:"externalId"`
	PreferredLabel    string  `json:"preferredLabel"`
	Labels            []Label `json:"labels,omitempty"`
	BroaderConceptIDs []Ref   `json:"broaderConceptIds,omitempty"`
	RelatedConceptIDs []Ref   `json:"relatedConceptIds,omitempty"`
}

// Label is a name in one BCP 47 language.
type Label struct {
	Language string `json:"language"`
	Label    string `json:"label"`
}
