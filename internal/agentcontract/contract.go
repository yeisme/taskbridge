package agentcontract

type Result struct {
	Schema               string       `json:"schema"`
	Status               string       `json:"status"`
	RequestID            string       `json:"request_id"`
	DryRun               bool         `json:"dry_run"`
	RequiresConfirmation bool         `json:"requires_confirmation"`
	Result               interface{}  `json:"result"`
	Warnings             []string     `json:"warnings"`
	Errors               []AgentError `json:"errors"`
}

type AgentError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	NextAction string `json:"next_action,omitempty"`
}

func OK(requestID string, dryRun bool, result interface{}) Result {
	return Result{Schema: "taskbridge.agent-result.v1", Status: "ok", RequestID: requestID, DryRun: dryRun, Result: result, Warnings: []string{}, Errors: []AgentError{}}
}

func Confirmation(requestID string, dryRun bool, result interface{}) Result {
	r := OK(requestID, dryRun, result)
	r.RequiresConfirmation = true
	return r
}

func Error(requestID, code, message, nextAction string) Result {
	return Result{
		Schema:    "taskbridge.agent-result.v1",
		Status:    "error",
		RequestID: requestID,
		Warnings:  []string{},
		Errors:    []AgentError{{Code: code, Message: message, NextAction: nextAction}},
	}
}
