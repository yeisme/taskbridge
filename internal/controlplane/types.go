package controlplane

import "time"

const (
	SchemaToday   = "taskbridge.today.v1"
	SchemaNext    = "taskbridge.next.v1"
	SchemaInbox   = "taskbridge.inbox.v1"
	SchemaReview  = "taskbridge.review.v1"
	SchemaDoctor  = "taskbridge.doctor.v1"
	SchemaActions = "taskbridge.actions.v1"
)

type Options struct {
	Now    time.Time
	Limit  int
	Source string
}

type TaskRef struct {
	ID               string     `json:"id"`
	Title            string     `json:"title"`
	Status           string     `json:"status"`
	Source           string     `json:"source"`
	ListID           string     `json:"list_id,omitempty"`
	ListName         string     `json:"list_name,omitempty"`
	Priority         int        `json:"priority,omitempty"`
	Quadrant         int        `json:"quadrant,omitempty"`
	DueDate          *time.Time `json:"due_date,omitempty"`
	EstimatedMinutes int        `json:"estimated_minutes,omitempty"`
	ProjectID        string     `json:"project_id,omitempty"`
	Reason           string     `json:"reason,omitempty"`
}

type Section struct {
	ID    string    `json:"id"`
	Title string    `json:"title"`
	Tasks []TaskRef `json:"tasks"`
}

type SuggestedAction struct {
	ActionID             string `json:"action_id"`
	Type                 string `json:"type"`
	TaskID               string `json:"task_id,omitempty"`
	ProjectID            string `json:"project_id,omitempty"`
	DueDate              string `json:"due_date,omitempty"`
	Reason               string `json:"reason"`
	RequiresConfirmation bool   `json:"requires_confirmation"`
}

type TodayResult struct {
	Schema           string            `json:"schema"`
	Date             string            `json:"date"`
	Status           string            `json:"status"`
	Summary          map[string]int    `json:"summary"`
	Sections         []Section         `json:"sections"`
	ProjectNext      []ProjectNextItem `json:"project_next,omitempty"`
	SuggestedActions []SuggestedAction `json:"suggested_actions,omitempty"`
	Warnings         []string          `json:"warnings"`
}

type ProjectNextItem struct {
	ProjectID   string `json:"project_id"`
	ProjectName string `json:"project_name"`
	NextTaskID  string `json:"next_task_id,omitempty"`
	RiskLevel   string `json:"risk_level"`
}

type ListResult struct {
	Schema   string    `json:"schema"`
	Status   string    `json:"status"`
	Count    int       `json:"count"`
	Tasks    []TaskRef `json:"tasks"`
	Warnings []string  `json:"warnings,omitempty"`
}

type ReviewResult struct {
	Schema           string            `json:"schema"`
	Status           string            `json:"status"`
	Summary          map[string]int    `json:"summary"`
	SuggestedActions []SuggestedAction `json:"suggested_actions"`
	Warnings         []string          `json:"warnings,omitempty"`
}

type DoctorResult struct {
	Schema     string        `json:"schema"`
	Status     string        `json:"status"`
	Checks     []DoctorCheck `json:"checks"`
	NextAction string        `json:"next_action,omitempty"`
}

type DoctorCheck struct {
	ID         string `json:"id"`
	Status     string `json:"status"`
	Message    string `json:"message"`
	NextAction string `json:"next_action,omitempty"`
}
