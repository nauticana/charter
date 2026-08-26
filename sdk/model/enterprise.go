package model

type Enterprise struct {
	Envelope
}

type OrganizationUnit struct {
	Envelope
	ParentUnitID *Ref `json:"parentUnitId,omitempty"`
}

type PositionType struct {
	Envelope
	Purpose string `json:"purpose"`
}

type Position struct {
	Envelope
	OrganizationUnitID Ref  `json:"organizationUnitId"`
	PositionTypeID     *Ref `json:"positionTypeId,omitempty"`
}

type Role struct {
	Envelope
	ResponsibilityIDs []Ref `json:"responsibilityIds,omitempty"`
}

type Responsibility struct {
	Envelope
	ExpectedOutcome string    `json:"expectedOutcome"`
	Accountable     ObjectRef `json:"accountable"`
}

type Assignment struct {
	Envelope
	Subject       ObjectRef `json:"subject"`
	Target        ObjectRef `json:"target"`
	Participation string    `json:"participation"`
	Condition     string    `json:"condition,omitempty"`
	Reason        string    `json:"reason,omitempty"`
}

type OrganizationRelationship struct {
	Envelope
	RelationshipType string    `json:"relationshipType"`
	Subject          ObjectRef `json:"subject"`
	Object           ObjectRef `json:"object"`
}
