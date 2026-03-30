package client

// UsageRequest is the request for usage statistics.
type UsageRequest struct {
	Period  string `json:"period,omitempty"`  // current, previous, YYYY-MM
	Service string `json:"service,omitempty"` // phishing, exposure, enrichment
}

// UsageResponse is the response from the usage endpoint.
type UsageResponse struct {
	Period        string                  `json:"period"`
	TotalRequests int                     `json:"total_requests"`
	TotalCredits  int                     `json:"total_credits"`
	ByService     map[string]ServiceUsage `json:"by_service"`
	TopOperations []OperationUsage        `json:"top_operations,omitempty"`
}

// ServiceUsage holds request and credit counts for a single service.
type ServiceUsage struct {
	Requests int `json:"requests"`
	Credits  int `json:"credits"`
}

// OperationUsage holds request and credit counts for a single operation.
type OperationUsage struct {
	Operation string `json:"operation"`
	Service   string `json:"service"`
	Requests  int    `json:"requests"`
	Credits   int    `json:"credits"`
}
