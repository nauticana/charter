package model

type EnterpriseSystem struct {
	Envelope
}

type SystemProfile struct {
	Envelope
	EnterpriseSystemID   Ref      `json:"enterpriseSystemId"`
	SystemVersion        string   `json:"systemVersion"`
	SupportedFeatures    []string `json:"supportedFeatures"`
	Constraints          []string `json:"constraints"`
	AuthenticationMethod string   `json:"authenticationMethod,omitempty"`
}

type CapabilityMappings struct {
	Inputs          string `json:"inputs"`
	Outputs         string `json:"outputs"`
	BusinessErrors  string `json:"businessErrors"`
	AuthorityChecks string `json:"authorityChecks"`
	Idempotency     string `json:"idempotency"`
	Evidence        string `json:"evidence"`
}

type LossyMapping struct {
	Mapping string `json:"mapping"`
	Effect  string `json:"effect"`
}

type CapabilityBinding struct {
	Envelope
	BindingVersion  string             `json:"bindingVersion"`
	SystemProfileID Ref                `json:"systemProfileId"`
	CapabilityID    Ref                `json:"capabilityId"`
	Operation       string             `json:"operation"`
	FeatureSupport  []FeatureSupport   `json:"featureSupport"`
	Mappings        CapabilityMappings `json:"mappings"`
	LossyMappings   []LossyMapping     `json:"lossyMappings,omitempty"`
}

type FieldMapping struct {
	CharterPath    string `json:"charterPath"`
	ExternalPath   string `json:"externalPath"`
	Transformation string `json:"transformation,omitempty"`
}

type LossyTransformation struct {
	Transformation     string `json:"transformation"`
	EffectOnValidation string `json:"effectOnValidation"`
	EffectOnDecisions  string `json:"effectOnDecisions"`
	EffectOnEvidence   string `json:"effectOnEvidence"`
}

type DataBinding struct {
	Envelope
	BindingVersion          string                `json:"bindingVersion"`
	FeatureSupport          []FeatureSupport      `json:"featureSupport"`
	SystemProfileID         Ref                   `json:"systemProfileId"`
	InformationDefinitionID Ref                   `json:"informationDefinitionId"`
	FieldMappings           []FieldMapping        `json:"fieldMappings"`
	LossyTransformations    []LossyTransformation `json:"lossyTransformations,omitempty"`
}

const FailClosed = "fail-closed"

type AuthorityBinding struct {
	Envelope
	BindingVersion                  string           `json:"bindingVersion"`
	FeatureSupport                  []FeatureSupport `json:"featureSupport"`
	SystemProfileID                 Ref              `json:"systemProfileId"`
	CapabilityID                    Ref              `json:"capabilityId"`
	ExternalEnforcement             string           `json:"externalEnforcement"`
	UnpreservableConstraintBehavior string           `json:"unpreservableConstraintBehavior"`
}

type EventBinding struct {
	Envelope
	BindingVersion    string           `json:"bindingVersion"`
	FeatureSupport    []FeatureSupport `json:"featureSupport"`
	SystemProfileID   Ref              `json:"systemProfileId"`
	ExternalEvent     string           `json:"externalEvent"`
	CharterTrigger    string           `json:"charterTrigger"`
	DeliverySemantics string           `json:"deliverySemantics"`
}

type BindingConformance struct {
	Envelope
	ConformanceEnvelope
	BindingID         Ref              `json:"bindingId"`
	EvaluatedFeatures []FeatureSupport `json:"evaluatedFeatures"`
}
