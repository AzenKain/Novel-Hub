export type Role = {
  id: number;
  name: string;
  description: string;
  is_system: boolean;
  is_admin: boolean;
  auto_assign: boolean;
  is_deleted: boolean;
  created_at?: string;
  updated_at?: string;
  permissions?: RolePermission[];
};

export type RoleSimple = {
  id: number;
  name: string;
  is_admin?: boolean;
};

export type Permission = {
  key: string;
  description: string;
  created_at?: string;
  updated_at?: string;
};

export type RolePermission = {
  id: number;
  role_id: number;
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
  userId: number;
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

export interface PublicSettings {
  site: SiteSettings;
  sidebar_visible_items: string[];
  home_sections: HomeSectionSettings;
  registration_enabled: boolean;
  guest_access: LibraryPolicy;
  download: LibraryPolicy;
  bookmark: LibraryPolicy;
  collection: LibraryPolicy;
  review: LibraryPolicy;
  share: LibraryPolicy;
  read: LibraryPolicy;
  stats: LibraryPolicy;
  enable_in_book_search?: boolean;
  enable_custom_font_upload?: boolean;
  setup_completed: boolean;
  available_sidebar_items: string[];
  available_home_sections: string[];
  available_policy_modes: string[];
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
