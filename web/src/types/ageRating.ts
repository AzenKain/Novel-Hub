export interface ContentWarning {
  id: string;
  name: string;
  description: string;
  created_at?: string;
}

export interface KidsModeInfo {
  id: string;
  is_kids_mode: boolean;
  has_pin: boolean;
  max_allowed_age_rating: string;
}
