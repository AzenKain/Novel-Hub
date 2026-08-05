package services

import (
	"encoding/json"
	"os"
	"testing"
)

// ListTasks is what the admin UI renders; a task absent from it is unreachable even though
// Trigger would accept it. scan_library_inbox was in the map and had a handler for months.
func TestListTasksExposesEveryTask(t *testing.T) {
	service := NewJobService(nil, nil)

	listed := make(map[string]string)
	for _, task := range service.ListTasks() {
		if _, dup := listed[task.Type]; dup {
			t.Errorf("%s appears twice in ListTasks", task.Type)
		}
		listed[task.Type] = task.Description
	}

	for taskType := range service.tasks {
		description, ok := listed[taskType]
		if !ok {
			t.Errorf("%s is in the tasks map but not in ListTasks, so the admin UI cannot trigger it", taskType)
			continue
		}
		if description == "" {
			t.Errorf("%s has no description; the admin UI would show a blank row", taskType)
		}
	}
	if len(listed) != len(service.tasks) {
		t.Errorf("ListTasks returned %d tasks, tasks map holds %d", len(listed), len(service.tasks))
	}
}

// Each task type is rendered with t(`admin.operations.tasks.${type}`), so a missing key shows
// the raw i18n path to the operator.
func TestEveryTaskHasALocaleLabel(t *testing.T) {
	service := NewJobService(nil, nil)

	for _, lang := range []string{"en", "vi", "ja", "ko", "zh"} {
		raw, err := os.ReadFile("../../web/public/locales/" + lang + ".json")
		if err != nil {
			t.Fatalf("read %s locale: %v", lang, err)
		}
		var bundle struct {
			Admin struct {
				Operations struct {
					Tasks map[string]string `json:"tasks"`
				} `json:"operations"`
			} `json:"admin"`
		}
		if err := json.Unmarshal(raw, &bundle); err != nil {
			t.Fatalf("parse %s locale: %v", lang, err)
		}
		for _, task := range service.ListTasks() {
			if label := bundle.Admin.Operations.Tasks[task.Type]; label == "" {
				t.Errorf("%s.json has no admin.operations.tasks.%s label", lang, task.Type)
			}
		}
	}
}
