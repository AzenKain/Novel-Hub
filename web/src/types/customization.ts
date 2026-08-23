export interface Soundscape {
  id: string;
  user_id?: string | null;
  name: string;
  category: string;
  file_path: string;
  stream_url: string;
  icon: string;
  volume: number;
  is_system: boolean;
  created_at: string;
  updated_at: string;
}

export interface CustomFont {
  id: string;
  user_id?: string | null;
  name: string;
  font_family: string;
  source_type: "file" | "url";
  file_url?: string;
  font_url?: string;
  is_system: boolean;
  created_at: string;
  updated_at: string;
}

export interface CustomTheme {
  id: string;
  user_id?: string | null;
  name: string;
  bg_color: string;
  text_color: string;
  accent_color: string;
  custom_css: string;
  is_system: boolean;
  created_at: string;
  updated_at: string;
}

export interface UploadSoundscapePayload {
  name: string;
  category?: string;
  icon?: string;
  volume?: number;
  audio_url?: string;
  is_system?: boolean;
  file?: File;
}

export interface UploadFontPayload {
  name: string;
  font_family: string;
  source_type: "file" | "url";
  font_url?: string;
  is_system?: boolean;
  file?: File;
}

export interface CreateCustomThemePayload {
  name: string;
  bg_color: string;
  text_color: string;
  accent_color: string;
  custom_css?: string;
  is_system?: boolean;
}

export interface UpdateCustomThemePayload {
  name: string;
  bg_color: string;
  text_color: string;
  accent_color: string;
  custom_css?: string;
}

export interface CustomizationListParams {
  cursor?: string;
  limit?: number;
}
