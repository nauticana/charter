package model

// ConformanceEnvelope holds the claim-level fields shared by ConformanceClaim and BindingConformance.
type ConformanceEnvelope struct {
	ConformanceProfile    string   `json:"conformanceProfile"`
	ImplementationVersion string   `json:"implementationVersion"`
	TestedRuleIDs         []string `json:"testedRuleIds"`
	VerificationTypes     []string `json:"verificationTypes"`
	Result                string   `json:"result"`
	ResultDate            Date     `json:"resultDate"`
}

type RuleResult struct {
	RuleID           string `json:"ruleId"`
	VerificationType string `json:"verificationType"`
	Result           string `json:"result"`
	Details          string `json:"details,omitempty"`
}

type ConformanceClaim struct {
	Envelope
	ConformanceEnvelope
	Implementation ObjectRef    `json:"implementation"`
	Results        []RuleResult `json:"results"`
}
