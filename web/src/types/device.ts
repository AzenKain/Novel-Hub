export interface UserDevice {
  id: string;
  user_id: string;
  name: string;
  device_type: "kindle" | "pocketbook" | "koreader" | string;
  target_address: string;
  created_at: string;
  updated_at: string;
}

export interface CreateUserDeviceInput {
  name: string;
  device_type: string;
  target_address: string;
}

export interface PushBookToDeviceInput {
  device_id: string;
}
