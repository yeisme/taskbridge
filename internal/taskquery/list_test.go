package taskquery

import (
	"testing"

	"github.com/yeisme/taskbridge/internal/model"
)

func TestBuildListQueryDefaultsToActiveStatuses(t *testing.T) {
	query, err := BuildListQuery(ListOptions{})
	if err != nil {
		t.Fatalf("BuildListQuery returned error: %v", err)
	}

	want := []model.TaskStatus{model.StatusTodo, model.StatusInProgress}
	if len(query.Query.Statuses) != len(want) {
		t.Fatalf("len(Statuses)=%d, want %d", len(query.Query.Statuses), len(want))
	}
	for i := range want {
		if query.Query.Statuses[i] != want[i] {
			t.Fatalf("Statuses[%d]=%q, want %q", i, query.Query.Statuses[i], want[i])
		}
	}
}

func TestBuildListQueryExplicitStatusBypassesDefault(t *testing.T) {
	query, err := BuildListQuery(ListOptions{Status: "completed,deferred", StatusChanged: true})
	if err != nil {
		t.Fatalf("BuildListQuery returned error: %v", err)
	}

	want := []model.TaskStatus{model.StatusCompleted, model.StatusDeferred}
	if len(query.Query.Statuses) != len(want) {
		t.Fatalf("len(Statuses)=%d, want %d", len(query.Query.Statuses), len(want))
	}
	for i := range want {
		if query.Query.Statuses[i] != want[i] {
			t.Fatalf("Statuses[%d]=%q, want %q", i, query.Query.Statuses[i], want[i])
		}
	}
}

func TestBuildListQueryNormalizesCanceledAlias(t *testing.T) {
	query, err := BuildListQuery(ListOptions{Status: "canceled", StatusChanged: true})
	if err != nil {
		t.Fatalf("BuildListQuery returned error: %v", err)
	}

	if len(query.Query.Statuses) != 1 || query.Query.Statuses[0] != model.StatusCancelled {
		t.Fatalf("Statuses=%v, want [%q]", query.Query.Statuses, model.StatusCancelled)
	}
}

func TestBuildListQueryResolvesProviderAlias(t *testing.T) {
	query, err := BuildListQuery(ListOptions{Source: "ms"})
	if err != nil {
		t.Fatalf("BuildListQuery returned error: %v", err)
	}
	if query.Source != "microsoft" {
		t.Fatalf("Source=%q, want microsoft", query.Source)
	}
	if got := query.Query.Sources[0]; got != model.TaskSource("microsoft") {
		t.Fatalf("Query.Sources[0]=%q, want microsoft", got)
	}
}

func TestBuildListQueryCleansListFilters(t *testing.T) {
	query, err := BuildListQuery(ListOptions{ListNames: []string{" Inbox ", "", "Inbox"}, TaskIDs: []string{"t1", " t1 ", "t2"}})
	if err != nil {
		t.Fatalf("BuildListQuery returned error: %v", err)
	}
	if got := query.Query.ListNames; len(got) != 1 || got[0] != "Inbox" {
		t.Fatalf("ListNames=%v, want [Inbox]", got)
	}
	if got := query.Query.TaskIDs; len(got) != 2 || got[0] != "t1" || got[1] != "t2" {
		t.Fatalf("TaskIDs=%v, want [t1 t2]", got)
	}
}
