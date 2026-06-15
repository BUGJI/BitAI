export interface ApiEnvelope<T> {
  code: number;
  message: string;
  data: T;
}

export interface User {
  id: number;
  email: string;
  display_name: string;
  avatar_url?: string;
  role: 'owner' | 'admin' | 'operator' | 'user' | 'auditor';
  status: string;
  balance_micros: number;
  concurrency_limit: number;
  rpm_limit: number;
  totp_enabled: boolean;
  created_at: string;
}

export interface TokenPair {
  access_token: string;
  refresh_token: string;
  expires_at: string;
  user: User;
}

export interface APIKey {
  id: number;
  name: string;
  key?: string;
  key_prefix: string;
  status: string;
  group_id?: number;
  quota_limit_micros: number;
  quota_used_micros: number;
  expires_at?: string;
  last_used_at?: string;
  created_at: string;
}

export interface Group {
  id: number;
  name: string;
  description: string;
  platform: string;
  mode: string;
  status: string;
  rate_multiplier_ppm: number;
  rpm_limit: number;
  model_mapping_json: string;
  model_list_json: string;
  features_json: string;
  sort_order: number;
}

export interface UpstreamAccount {
  id: number;
  name: string;
  platform: string;
  auth_type: string;
  credentials?: string;
  base_url: string;
  priority: number;
  weight: number;
  status: string;
  schedulable: boolean;
  last_used_at?: string;
}

export interface UsageLog {
  id: number;
  request_id: string;
  user_id: number;
  api_key_id: number;
  group_id?: number;
  upstream_account_id?: number;
  platform: string;
  model_requested: string;
  model_used: string;
  prompt_tokens: number;
  completion_tokens: number;
  total_tokens: number;
  charged_micros: number;
  status_code: number;
  latency_ms: number;
  created_at: string;
}

export interface GroupAccount {
  id: number;
  group_id: number;
  group_name: string;
  upstream_account_id: number;
  upstream_name: string;
  weight: number;
  priority: number;
  enabled: boolean;
  created_at: string;
}

export interface Setting {
  id: number;
  key: string;
  value: string;
  is_public: boolean;
  created_at: string;
  updated_at: string;
}

export interface PaymentOrder {
  id: number;
  user_id: number;
  order_no: string;
  amount_micros: number;
  status: string;
  provider: string;
  paid_at?: string;
  created_at: string;
}

export interface RedeemCode {
  id: number;
  code?: string;
  code_prefix: string;
  amount_micros: number;
  status: string;
  max_uses: number;
  used_count: number;
  expires_at?: string;
  created_by_id: number;
  created_at: string;
}
