package response

type ValidationIssueResponse struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	File     string `json:"file,omitempty"`
	Message  string `json:"message"`
	Fixable  bool   `json:"fixable"`
	FixID    string `json:"fix_id,omitempty"`
}

type ValidationReportResponse struct {
	Valid    bool                      `json:"valid"`
	Errors   int                       `json:"errors"`
	Warnings int                       `json:"warnings"`
	Infos    int                       `json:"infos"`
	Issues   []ValidationIssueResponse `json:"issues"`
}

type BookRepairResponse struct {
	Success    bool                     `json:"success"`
	FixedCount int                      `json:"fixed_count"`
	Logs       []string                 `json:"logs"`
	Report     ValidationReportResponse `json:"report"`
}
