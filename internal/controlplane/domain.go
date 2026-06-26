package controlplane

import (
	"fmt"
	"strings"

	"github.com/yeisme/taskbridge/internal/model"
)

const (
	customFieldDomain   = "domain"
	customFieldTBDomain = "tb_domain"
	customFieldSync     = "sync_state"
	customFieldTBSync   = "tb_sync_state"
)

func classifyDomain(task model.Task) model.TaskDomain {
	if model.IsKnownTaskDomain(task.Domain) && task.Domain != "" {
		return model.NormalizeTaskDomain(task.Domain)
	}
	if task.Metadata != nil {
		if domain, ok := domainFromCustomFields(task.Metadata.CustomFields); ok {
			return domain
		}
	}
	if domain, ok := inferDomainFromSignals(task.ListName, task.ListID, strings.Join(task.Tags, " "), strings.Join(task.Categories, " ")); ok {
		return domain
	}
	return model.DomainUnknown
}

func domainFromCustomFields(fields map[string]interface{}) (model.TaskDomain, bool) {
	for _, key := range []string{customFieldTBDomain, customFieldDomain} {
		if value, ok := fields[key]; ok {
			domain := model.TaskDomain(strings.ToLower(strings.TrimSpace(fmt.Sprint(value))))
			if model.IsKnownTaskDomain(domain) {
				return model.NormalizeTaskDomain(domain), true
			}
		}
	}
	return model.DomainUnknown, false
}

func inferDomainFromSignals(signals ...string) (model.TaskDomain, bool) {
	joined := strings.ToLower(strings.Join(signals, " "))
	if strings.TrimSpace(joined) == "" {
		return model.DomainUnknown, false
	}
	workWords := []string{"work", "office", "job", "team", "client", "customer", "meeting", "sprint", "launch", "roadmap", "project", "review"}
	lifeWords := []string{"life", "home", "family", "health", "doctor", "fitness", "gym", "errand", "school", "travel", "personal"}
	for _, word := range workWords {
		if strings.Contains(joined, word) {
			return model.DomainWork, true
		}
	}
	for _, word := range lifeWords {
		if strings.Contains(joined, word) {
			if word == "personal" {
				return model.DomainPersonal, true
			}
			return model.DomainLife, true
		}
	}
	return model.DomainUnknown, false
}

func syncState(task model.Task) string {
	if task.Metadata == nil || len(task.Metadata.CustomFields) == 0 {
		return "ok"
	}
	for _, key := range []string{customFieldTBSync, customFieldSync} {
		if value, ok := task.Metadata.CustomFields[key]; ok {
			state := strings.ToLower(strings.TrimSpace(fmt.Sprint(value)))
			if state != "" {
				return state
			}
		}
	}
	return "ok"
}

func hasSyncRisk(task model.Task) bool {
	switch syncState(task) {
	case "conflict", "conflicted", "uncertain", "error", "failed", "stale":
		return true
	default:
		return false
	}
}
