/**
 * Determines which quick-filter chips are logically valid for the active navigation view.
 * Prevents contradictory filters (e.g. Unread filter on Read tab) and redundant chips.
 */
export function getAvailableChips(
  nav?: string,
  activeSmartFilterId?: string | null,
): string[] {
  // On Read books: Unread is contradictory, Reading is contradictory
  if (nav === "read") {
    return ["All", "No cover"];
  }
  // On Unread books: Unread is redundant, Reading is contradictory
  if (nav === "unread") {
    return ["All", "No cover"];
  }
  // On Smart Filter: predefined rules govern criteria
  if (activeSmartFilterId) {
    return ["All", "No cover"];
  }
  return ["All", "Reading", "Unread", "No cover"];
}

/**
 * Sanitizes the selected chip against currently available chips.
 * Falls back to "All" if an incompatible chip was active prior to tab change.
 */
export function sanitizeChip(
  rawChip: string | null | undefined,
  availableChips: string[],
): string {
  if (!rawChip) return "All";
  return availableChips.includes(rawChip) ? rawChip : "All";
}

/**
 * Checks whether the current view is a dedicated catalog page rather than the Home Dashboard.
 */
export function isCatalogView(params: {
  hasFacetSection?: boolean;
  activeSmartFilterId?: string | null;
  effectiveCollection?: string;
  debouncedSearch?: string;
  effectiveNav?: string;
}): boolean {
  return (
    !!params.hasFacetSection ||
    !!params.activeSmartFilterId ||
    !!params.effectiveCollection ||
    !!params.debouncedSearch ||
    (!!params.effectiveNav && params.effectiveNav !== "books")
  );
}
