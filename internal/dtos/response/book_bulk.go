package response

type BulkOperationResponse struct {
	SuccessCount int               `json:"success_count"`
	FailedCount  int               `json:"failed_count"`
	Errors       map[string]string `json:"errors,omitempty"`
}
