package services

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

const auditValueMaxLen = 60

// SettingsAuditLabel renders what an admin changed.
func SettingsAuditLabel(values map[string]any) string {
	parts := make([]string, 0, len(values))
	for _, key := range slices.Sorted(maps.Keys(values)) {
		if secretSettingKey(key) {
			parts = append(parts, key+" = (updated)")
			continue
		}
		parts = append(parts, key+" = "+auditValue(values[key]))
	}
	return strings.Join(parts, ", ")
}

func auditValue(value any) string {
	rendered := fmt.Sprintf("%v", dereferenceSettingValue(value))
	if len(rendered) > auditValueMaxLen {
		return rendered[:auditValueMaxLen] + "…"
	}
	return rendered
}
