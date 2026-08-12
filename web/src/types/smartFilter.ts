export interface SmartFilterRuleItem {
  field: 'status' | 'format' | 'rating_gte' | 'author_id' | 'series_id' | 'tag_id';
  operator: 'eq' | 'gte' | 'lte';
  value: string;
}

export interface SmartFilter {
  id: string;
  user_id: string;
  name: string;
  rules: SmartFilterRuleItem[];
  is_pinned_sidebar: boolean;
  is_pinned_home: boolean;
  home_position: number;
  created_at: string;
  updated_at: string;
}

export interface UpsertSmartFilterPayload {
  name: string;
  rules: SmartFilterRuleItem[];
  is_pinned_sidebar: boolean;
  is_pinned_home: boolean;
}

export interface ReorderHomeShelfItem {
  id: string;
  position: number;
}
