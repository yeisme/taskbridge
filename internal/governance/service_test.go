package governance

import (
	"strings"
	"testing"
)

func TestBuildOverdueQuestionsUseEnglish(t *testing.T) {
	questions := buildOverdueQuestions(1, true)
	joined := strings.Join(questions, " ")
	if !strings.Contains(joined, "Which tasks should be deferred?") {
		t.Fatalf("questions should be English and readable: %#v", questions)
	}
	if strings.Contains(joined, "哪些") || strings.Contains(joined, "ï") {
		t.Fatalf("questions should not contain Chinese/mojibake in CLI payload: %#v", questions)
	}
}
