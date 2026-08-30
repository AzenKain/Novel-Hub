package request

type RepairBookRequest struct {
	NormalizeMimetype *bool `json:"normalize_mimetype"`
	FixContainer      *bool `json:"fix_container"`
	FixXHTML          *bool `json:"fix_xhtml"`
	ReconcileManifest *bool `json:"reconcile_manifest"`
	ReconcileSpine    *bool `json:"reconcile_spine"`
	FixTOC            *bool `json:"fix_toc"`
	CleanBrokenLinks  *bool `json:"clean_broken_links"`
	FixMetadata       *bool `json:"fix_metadata"`
}
