package services

import "testing"

func TestConditionsMatchFailsClosed(t *testing.T) {
	tests := []struct {
		name       string
		conditions map[string]any
		attrs      map[string]any
		want       bool
	}{
		{name: "global", conditions: map[string]any{}, want: true},
		{name: "matching library", conditions: map[string]any{"library_ids": []any{"lib-1"}}, attrs: map[string]any{"library_id": "lib-1"}, want: true},
		{name: "other library", conditions: map[string]any{"library_ids": []any{"lib-1"}}, attrs: map[string]any{"library_id": "lib-2"}},
		{name: "missing attribute", conditions: map[string]any{"library_ids": []any{"lib-1"}}},
		{name: "unknown key", conditions: map[string]any{"library_id": "lib-1"}, attrs: map[string]any{"library_id": "lib-1"}},
		{name: "extra key", conditions: map[string]any{"library_ids": []any{"lib-1"}, "owner": true}, attrs: map[string]any{"library_id": "lib-1"}},
		{name: "scalar", conditions: map[string]any{"library_ids": "lib-1"}, attrs: map[string]any{"library_id": "lib-1"}},
		{name: "empty list", conditions: map[string]any{"library_ids": []any{}}, attrs: map[string]any{"library_id": "lib-1"}},
		{name: "non string item", conditions: map[string]any{"library_ids": []any{1}}, attrs: map[string]any{"library_id": "1"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := conditionsMatch(tt.conditions, tt.attrs); got != tt.want {
				t.Fatalf("conditionsMatch() = %v, want %v", got, tt.want)
			}
		})
	}
}
