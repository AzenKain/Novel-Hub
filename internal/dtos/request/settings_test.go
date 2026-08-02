package request

import "testing"

// An absent field must not reach the service at all. It used to arrive as a typed
// nil pointer boxed in a non-nil interface, which the settings service then
// dereferenced — panicking the whole process on any admin settings save.
func TestValuesOmitsAbsentFields(t *testing.T) {
	empty := &UpdateSettingsDto{}
	if values := empty.Values(); len(values) != 0 {
		t.Fatalf("expected no values from an all-absent DTO, got %d: %#v", len(values), values)
	}
}

func TestValuesUnwrapsPresentFields(t *testing.T) {
	title := "NovelHub"
	registration := false
	apiMax := 42
	window := int64(90)
	libraries := []string{"lib-1"}
	randomBooks := true

	dto := &UpdateSettingsDto{
		SiteTitle:                 &title,
		RegistrationEnabled:       &registration,
		RateLimitAPI:              &apiMax,
		RateLimitAPIWindowSeconds: &window,
		GuestAccessLibraryIDs:     &libraries,
		HomeSections:              &HomeSectionSettingsDto{RandomBooks: &randomBooks},
	}

	values := dto.Values()
	for key, want := range map[string]any{
		"site.title":                           title,
		"auth.registration_enabled":            registration,
		"limits.rate_limit_api":                apiMax,
		"limits.rate_limit_api_window_seconds": window,
	} {
		if got := values[key]; got != want {
			t.Errorf("%s = %#v, want %#v", key, got, want)
		}
	}

	if got, ok := values["guest_access.library_ids"].([]string); !ok || len(got) != 1 || got[0] != "lib-1" {
		t.Errorf("guest_access.library_ids = %#v, want []string{\"lib-1\"}", values["guest_access.library_ids"])
	}

	sections, ok := values["home.sections"].(map[string]any)
	if !ok {
		t.Fatalf("home.sections = %#v, want map", values["home.sections"])
	}
	if sections["random_books"] != true {
		t.Errorf("random_books = %#v, want true", sections["random_books"])
	}
	if _, present := sections["top_books"]; present {
		t.Error("top_books was absent from the request and must not be set")
	}
}
