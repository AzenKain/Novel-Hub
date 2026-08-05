package services

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

const auditValueMaxLen = 60

// SettingsAuditLabel renders what an admin changed. Values are included so the trail can answer
// "who turned registration off", but a secret key is reduced to its name: smtp.password is
// encrypted at rest precisely so it never lands somewhere every admin can read.
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
