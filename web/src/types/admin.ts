export type Role = {
  id: string;
  name: string;
  description: string;
  is_system: boolean;
  is_admin: boolean;
  is_banned: boolean;
  auto_assign: boolean;
  position?: number;
  is_deleted: boolean;
  created_at?: string;
  updated_at?: string;
  permissions?: RolePermission[];
};

export type RoleSimple = {
  id: string;
  name: string;
  is_admin?: boolean;
  is_banned?: boolean;
  position?: number;
  permissions?: RolePermission[];
};

export type Permission = {
  key: string;
  description: string;
  created_at?: string;
  updated_at?: string;
};

export type RolePermission = {
  id: string;
  role_id: string;
  permission_key: string;
  effect: "allow" | "deny";
  conditions_json: string;
  conditions?: Record<string, unknown>;
  created_at?: string;
  updated_at?: string;
};

export type CreateRoleRequest = {
  name: string;
  description?: string;
  auto_assign?: boolean;
  permissions?: {
    permission_key: string;
    effect?: "allow" | "deny";
    conditions?: Record<string, unknown>;
  }[];
};

export type UpdateRoleRequest = CreateRoleRequest;

export type UpdateRolePermissionsRequest = {
  permissions: {
    permission_key: string;
    effect?: "allow" | "deny";
    conditions?: Record<string, unknown>;
  }[];
};

export type UpdateSettingsRequest = Record<string, unknown>;

export interface AdminReview {
  userId: string;
  bookId: string;
  rating: number;
  review?: string;
  createdAt?: string;
  updatedAt?: string;
  userName?: string;
  userEmail?: string;
  bookTitle?: string;
}

export interface SiteSettings {
  title: string;
  description: string;
  favicon: string;
  logo: string;
  meta_description: string;
}

export interface HomeSectionSettings {
  random_books: boolean;
  top_books: boolean;
}

export interface LibraryPolicy {
  mode: string;
  library_ids: string[];
  visible_stats?: string[];
}

export interface RuntimeLimits {
  upload_chunk_bytes: number;
  upload_chunks: number;
  upload_sessions: number;
  upload_bytes: number;
  upload_session_ttl_seconds: number;
  cover_bytes: number;
  site_asset_bytes: number;
}

export interface RuntimeLimitBounds {
  min: RuntimeLimits;
  max: RuntimeLimits;
}

export interface AdminSettings extends PublicSettings {
  limits: RuntimeLimits;
  bounds: RuntimeLimitBounds;
}

export interface PublicSettings {
  site: SiteSettings;
  sidebar_visible_items: string[];
  home_sections: HomeSectionSettings;
  registration_enabled: boolean;
  guest_access: LibraryPolicy;
  guest_permissions?: string[];
  enable_in_book_search?: boolean;
  enable_custom_font_upload?: boolean;
  enable_anilist_tracking?: boolean;
  setup_completed: boolean;
  available_sidebar_items: string[];
  available_home_sections: string[];
  available_guest_modes: string[];
}

export interface Webhook {
  id: string;
  name: string;
  url: string;
  template_type: "generic" | "discord" | "telegram" | "slack";
  secret?: string;
  custom_headers?: string;
  events: string[];
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateWebhookInput {
  name: string;
  url: string;
  template_type: "generic" | "discord" | "telegram" | "slack";
  secret?: string;
  custom_headers?: string;
  events: string[];
  is_active?: boolean;
}

export interface BackgroundJob {
  id: string;
  type: string;
  status?: "pending" | "running" | "completed" | "failed";
  progress?: number;
  total?: number;
  errorMsg?: string;
  payloadJson?: string;
  createdAt: string;
  updatedAt: string;
}

export interface JobTask {
  type: string;
  description: string;
}

export interface JobSchedule {
  id: string;
  name: string;
  taskType: string;
  payloadJson?: string;
  intervalMinutes: number;
  enabled: boolean;
  nextRunAt: string;
  lastRunAt?: string;
  lastJobId?: string;
  createdAt: string;
  updatedAt: string;
}

export interface UpsertJobScheduleInput {
  name: string;
  task_type: string;
  payload_json?: string;
  interval_minutes: number;
  enabled: boolean;
}

export interface LogFileInfo {
  name: string;
  sizeBytes: number;
  updatedAt: string;
}

export interface LogTail {
  file: string;
  lines: string[];
}

export interface BackupInfo {
  name: string;
  sizeBytes: number;
  createdAt: string;
  includeBooks: boolean;
}

export interface RestoreResult {
  restartRequired: boolean;
  autoRestart: boolean;
}

export interface CalibreImportResult {
  imported_count: number;
}
