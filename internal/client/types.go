package client

// ---------------------------------------------------------------------------
// Enrichment
// ---------------------------------------------------------------------------

// EnrichmentRequest is the request body for POST /enrichment/lookup
type EnrichmentRequest struct {
	Indicators []Indicator `json:"indicators"`
}

// Indicator represents a single indicator of compromise.
type Indicator struct {
	Type  string `json:"type"`  // ip, domain, url, hash, email
	Value string `json:"value"`
}

// EnrichmentResponse is the data field of the enrichment response.
type EnrichmentResponse struct {
	Results []EnrichmentResult `json:"results"`
	Summary EnrichmentSummary  `json:"summary"`
}

// EnrichmentResult holds the enrichment outcome for a single indicator.
type EnrichmentResult struct {
	Indicator  Indicator                 `json:"indicator"`
	Verdict    string                    `json:"verdict"` // malicious, suspicious, clean, unknown
	Score      int                       `json:"score"`
	Providers  map[string]ProviderResult `json:"providers"`
	EnrichedAt string                    `json:"enriched_at"`
}

// ProviderResult contains a single provider's enrichment data.
type ProviderResult struct {
	Verdict string                 `json:"verdict"`
	Score   int                    `json:"score"`
	Details map[string]interface{} `json:"details"`
}

// EnrichmentSummary provides aggregate counts by verdict.
type EnrichmentSummary struct {
	Total      int `json:"total"`
	Malicious  int `json:"malicious"`
	Suspicious int `json:"suspicious"`
	Clean      int `json:"clean"`
	Unknown    int `json:"unknown"`
}

// ---------------------------------------------------------------------------
// Phishing Analysis
// ---------------------------------------------------------------------------

// PhishingAnalyzeRequest is the request body for POST /phishing/analyze.
type PhishingAnalyzeRequest struct {
	InputType string                  `json:"input_type"` // headers, eml, raw
	Content   string                  `json:"content"`
	Options   *PhishingAnalyzeOptions `json:"options,omitempty"`
}

// PhishingAnalyzeOptions controls optional analysis modules.
type PhishingAnalyzeOptions struct {
	IncludeIOCs               bool `json:"include_iocs"`
	IncludeBodyAnalysis       bool `json:"include_body_analysis"`
	IncludeHomoglyphCheck     bool `json:"include_homoglyph_check"`
	IncludeLinkAnalysis       bool `json:"include_link_analysis"`
	IncludeAttachmentAnalysis bool `json:"include_attachment_analysis"`
}

// PhishingAnalyzeResponse is the data field of the phishing analysis response.
type PhishingAnalyzeResponse struct {
	Verdict              PhishingVerdict       `json:"verdict"`
	KeyFindings          []string              `json:"key_findings"`
	RecommendedActions   []string              `json:"recommended_actions"`
	AuthenticationResults *AuthResults          `json:"authentication_results,omitempty"`
	SuspiciousIndicators []SuspiciousIndicator `json:"suspicious_indicators,omitempty"`
	EmailMetadata        *EmailMetadata        `json:"email_metadata,omitempty"`
	RoutingHops          []RoutingHop          `json:"routing_hops,omitempty"`
	ExtractedIOCs        []ExtractedIOC        `json:"extracted_iocs,omitempty"`
	BodyAnalysis         *BodyAnalysis         `json:"body_analysis,omitempty"`
	HomoglyphFindings    []HomoglyphFinding    `json:"homoglyph_findings,omitempty"`
	LinkMismatches       []LinkMismatch        `json:"link_mismatches,omitempty"`
	AttachmentAnalysis   *AttachmentAnalysis   `json:"attachment_analysis,omitempty"`
}

// PhishingVerdict describes the overall phishing assessment.
type PhishingVerdict struct {
	Level   string `json:"level"`   // safe, suspicious, malicious, highly_malicious
	Score   int    `json:"score"`
	Summary string `json:"summary"`
}

// AuthResults holds email authentication check results.
type AuthResults struct {
	SPF           string `json:"spf"`
	DKIM          string `json:"dkim"`
	DMARC         string `json:"dmarc"`
	ARC           string `json:"arc"`
	DKIMSignature string `json:"dkim_signature"`
}

// SuspiciousIndicator is a single suspicious signal found during analysis.
type SuspiciousIndicator struct {
	Category    string `json:"category"`
	Description string `json:"description"`
	Severity    string `json:"severity"` // low, medium, high
}

// EmailMetadata contains parsed header fields from the email.
type EmailMetadata struct {
	From           string `json:"from"`
	To             string `json:"to"`
	Subject        string `json:"subject"`
	Date           string `json:"date"`
	MessageID      string `json:"message_id"`
	ReturnPath     string `json:"return_path"`
	ReplyTo        string `json:"reply_to"`
	XMailer        string `json:"x_mailer"`
	XOriginatingIP string `json:"x_originating_ip"`
	ContentType    string `json:"content_type"`
}

// RoutingHop represents a single hop in the email delivery chain.
type RoutingHop struct {
	Hostname  string `json:"hostname"`
	IP        string `json:"ip"`
	Timestamp string `json:"timestamp"`
	Delay     string `json:"delay"`
}

// ExtractedIOC is an indicator of compromise extracted from the email.
type ExtractedIOC struct {
	Type              string `json:"type"`
	Value             string `json:"value"`
	Location          string `json:"location"` // body, header, attachment
	Defanged          string `json:"defanged"`
	EnrichmentVerdict string `json:"enrichmentVerdict,omitempty"`
	EnrichmentScore   int    `json:"enrichmentScore,omitempty"`
}

// BodyAnalysis contains NLP-derived signals from the email body.
type BodyAnalysis struct {
	UrgencyIndicators   []string `json:"urgency_indicators"`
	FinancialReferences []string `json:"financial_references"`
	TrustSignals        []string `json:"trust_signals"`
	AuthorityAppeals    []string `json:"authority_appeals"`
	BrandMentions       []string `json:"brand_mentions"`
}

// HomoglyphFinding describes a detected homoglyph/lookalike domain.
type HomoglyphFinding struct {
	DetectedDomain  string `json:"detected_domain"`
	PotentialTarget string `json:"potential_target"`
	SimilarityScore int    `json:"similarity_score"`
}

// LinkMismatch flags a mismatch between displayed text and the actual URL.
type LinkMismatch struct {
	DisplayText      string `json:"display_text"`
	ActualURL        string `json:"actual_url"`
	MismatchSeverity string `json:"mismatch_severity"`
}

// AttachmentAnalysis holds analysis results for email attachments.
type AttachmentAnalysis struct {
	Attachments []Attachment `json:"attachments"`
}

// Attachment describes a single analysed email attachment.
type Attachment struct {
	Filename    string   `json:"filename"`
	Extension   string   `json:"extension"`
	SizeBytes   int64    `json:"size_bytes"`
	ContentType string   `json:"content_type"`
	Hash        string   `json:"hash"`
	Warnings    []string `json:"warnings"`
}

// ---------------------------------------------------------------------------
// Phishing AI Analysis
// ---------------------------------------------------------------------------

// PhishingAIResponse is the data field of the AI analysis response.
type PhishingAIResponse struct {
	Analysis  PhishingAnalyzeResponse `json:"analysis"`
	AIVerdict *AIVerdict              `json:"ai_verdict"`
}

// AIVerdict contains the AI model's assessment of the email.
type AIVerdict struct {
	RiskLevel          string   `json:"risk_level"` // safe, suspicious, malicious, highly_malicious
	ConfidenceScore    int      `json:"confidence_score"`
	ExecutiveSummary   string   `json:"executive_summary"`
	KeyFindings        []string `json:"key_findings"`
	RecommendedActions []string `json:"recommended_actions"`
	Model              string   `json:"model"`
}

// ---------------------------------------------------------------------------
// Exposure Scan
// ---------------------------------------------------------------------------

// ExposureScanRequest is the request body for POST /exposure/scan.
type ExposureScanRequest struct {
	Target     string `json:"target"`
	TargetType string `json:"target_type,omitempty"` // domain, ip, auto
}

// ExposureScanResponse is the data field of the exposure scan response.
type ExposureScanResponse struct {
	Cached     bool                   `json:"cached"`
	ScanID     string                 `json:"scan_id"`
	Target     string                 `json:"target"`
	TargetType string                 `json:"target_type"`
	Status     string                 `json:"status"` // READY, IN_PROGRESS, CACHED
	RiskScore  int                    `json:"riskScore"`
	RiskLevel  string                 `json:"riskLevel"` // low, medium, high, critical
	Results    map[string]interface{} `json:"results"`
	Providers  []string               `json:"providers"`
	Stats      *ScanStats             `json:"stats,omitempty"`
}

// ScanStats contains timing metadata for a scan.
type ScanStats struct {
	DurationMs  int    `json:"duration_ms"`
	LastScanned string `json:"last_scanned"`
}

// ---------------------------------------------------------------------------
// Auth / Validate
// ---------------------------------------------------------------------------

// AuthValidateResponse is returned by GET /auth/validate on dfir-platform.
type AuthValidateResponse struct {
	Plan             string   `json:"plan"` // free, starter, professional, enterprise
	Credits          int      `json:"credits"`
	OrganizationName string   `json:"organization_name"`
	OrganizationID   string   `json:"organization_id"`
	Permissions      []string `json:"permissions"`
	MaxAPIKeys       int      `json:"max_api_keys"`
	MaxMembers       int      `json:"max_members"`
	RateLimitTier    string   `json:"rate_limit_tier"`
}

// ---------------------------------------------------------------------------
// Health Check
// ---------------------------------------------------------------------------

// HealthResponse is returned by the health endpoint.
type HealthResponse struct {
	Status    string `json:"status"` // operational, degraded, outage
	Version   string `json:"version"`
	Timestamp string `json:"timestamp"`
}
