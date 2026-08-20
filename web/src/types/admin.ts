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

export type SendUserEmailRequest = {
  subject: string;
  body: string;
};

export interface AdminReview {
  user_id: string;
  book_id: string;
  rating: number;
  review?: string;
  created_at?: string;
  updated_at?: string;
  user_name?: string;
  user_email?: string;
  book_title?: string;
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
  soundscape_bytes: number;
  font_bytes: number;
  rate_limit_auth: number;
  rate_limit_auth_window_seconds: number;
}

export interface RuntimeLimitBounds {
  min: RuntimeLimits;
  max: RuntimeLimits;
}

export type SmtpTlsMode = "none" | "starttls" | "implicit_tls";

export interface SmtpSettings {
  enabled: boolean;
  host: string;
  port: number;
  username: string;
  from_email: string;
  tls_mode: SmtpTlsMode;
  allow_private_networks: boolean;
  max_attachment_mb: number;
  password_configured: boolean;
  available_tls_modes: SmtpTlsMode[];
}

export type SmtpTestRequest = {
  host?: string;
  port?: number;
  username?: string;
  password?: string;
  from_email?: string;
  tls_mode?: SmtpTlsMode;
  allow_private_networks?: boolean;
};

export interface ProxyAuthSettings {
  enabled: boolean;
  header_names: string[];
  trusted_proxies: string[];
  auto_create: boolean;
}

export interface OAuthProviderPublic {
  id: string;
  display_name: string;
  enabled: boolean;
}

export interface OAuthSettingsPublic {
  providers: OAuthProviderPublic[];
}

export interface OAuthProviderAdmin {
  enabled: boolean;
  client_id: string;
  client_secret_set: boolean;
  redirect_uri: string;
  name?: string;
  issuer_url?: string;
  scopes?: string[];
}

export interface OAuthSettingsAdmin {
  google: OAuthProviderAdmin;
  github: OAuthProviderAdmin;
  discord: OAuthProviderAdmin;
  oidc: OAuthProviderAdmin;
}

export interface HardcoverSettingsAdmin {
  enabled: boolean;
  client_id: string;
  client_secret_set: boolean;
}

export interface AdminSettings extends Omit<PublicSettings, 'oauth'> {
  limits: RuntimeLimits;
  bounds: RuntimeLimitBounds;
  smtp: SmtpSettings;
  server_url?: string;
  proxy_auth: ProxyAuthSettings;
  oauth?: OAuthSettingsAdmin;
  hardcover?: HardcoverSettingsAdmin;
}

export interface PublicSettings {
  site: SiteSettings;
  sidebar_visible_items: string[];
  home_sections: HomeSectionSettings;
  oauth?: OAuthSettingsPublic;
  registration_enabled: boolean;
  guest_login_required: boolean;
  guest_access: LibraryPolicy;
  guest_permissions?: string[];
  enable_in_book_search?: boolean;
  enable_custom_font_upload?: boolean;
  enable_anilist_tracking?: boolean;
  enable_hardcover_scrobbling?: boolean;
  enable_auto_enrich?: boolean;
  enable_webp_cover?: boolean;
  require_email_verify?: boolean;
  password_reset_enabled?: boolean;
  smtp_enabled?: boolean;
  setup_completed: boolean;
  available_sidebar_items: string[];
  available_home_sections: string[];
  available_guest_modes: string[];
  proxy_auth_enabled?: boolean;
}

export interface Webhook {
  id: string;
  name: string;
  url: string;
  template_type: "generic" | "discord" | "telegram" | "slack" | "email";
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
  template_type: "generic" | "discord" | "telegram" | "slack" | "email";
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
  created_at: string;
  updated_at: string;
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
  created_at: string;
  updated_at: string;
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
  size_bytes: number;
  updated_at: string;
}

export interface LogTail {
  file: string;
  lines: string[];
}

export interface BackupInfo {
  name: string;
  size_bytes: number;
  created_at: string;
  includeBooks: boolean;
}

export interface RestoreResult {
  restartRequired: boolean;
  autoRestart: boolean;
}

export interface CalibreImportResult {
  imported_count: number;
}

export interface AuditLogEntry {
  id: string;
  actor_id?: string;
  actor_email: string;
  action: string;
  target_type: string;
  target_id?: string;
  target_label: string;
  ip: string;
  created_at: string;
}

export interface CacheStats {
  hits: number;
  misses: number;
  hit_rate: number;
  max_cost: number;
  entry_count: number;
}

