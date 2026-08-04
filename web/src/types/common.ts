export type ProtectedRouteProps = {
  requiredRoles?: string[];
  requiredPermission?: string;
  requiredAnyPermissions?: string[];
  redirectPath?: string;
};

export type CommonResponse<T> = {
  status: boolean;
  data?: T;
  errors?: unknown;
  message?: string;
};

export type PaginatedResponse<T> = {
  status: boolean;
  message?: string;
  data?: T[];
  pagination?: {
    current_page: number;
    page_size: number;
    total_records: number;
    total_pages: number;
    next_cursor?: string;
  };
};

export type CursorPaginatedResponse<T> = {
  status: boolean;
  message?: string;
  data?: T[];
  next_cursor?: string | null;
};
