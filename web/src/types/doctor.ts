export interface ValidationIssue {
  severity: "error" | "warning" | "info";
  code: string;
  file?: string;
  message: string;
  fixable: boolean;
  fix_id?: string;
}

export interface ValidationReport {
  valid: boolean;
  errors: number;
  warnings: number;
  infos: number;
  issues: ValidationIssue[];
}

export interface RepairOptions {
  normalize_mimetype?: boolean;
  fix_container?: boolean;
  fix_xhtml?: boolean;
  reconcile_manifest?: boolean;
  reconcile_spine?: boolean;
  fix_toc?: boolean;
  clean_broken_links?: boolean;
  fix_metadata?: boolean;
}

export interface BookRepairResult {
  success: boolean;
  fixed_count: number;
  logs: string[];
  report: ValidationReport;
}
