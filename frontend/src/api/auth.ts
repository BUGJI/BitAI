import { apiClient, unwrap } from './client';
import type { TokenPair, User } from '../types';

export interface AuthPayload {
  email: string;
  password: string;
  display_name?: string;
  captcha_token: string;
  captcha_code: string;
  email_token?: string;
  email_code?: string;
}

export const authApi = {
  captcha() {
    return unwrap<{ captcha_token: string; captcha_image: string; expires_at: string }>(apiClient.get('/auth/captcha'));
  },
  sendEmailCode(payload: Pick<AuthPayload, 'email' | 'captcha_token' | 'captcha_code'>) {
    return unwrap<{ sent: boolean; email_token: string }>(apiClient.post('/auth/email-code', payload));
  },
  login(payload: AuthPayload) {
    return unwrap<TokenPair>(apiClient.post('/auth/login', payload));
  },
  register(payload: AuthPayload) {
    return unwrap<TokenPair>(apiClient.post('/auth/register', payload));
  },
  me() {
    return unwrap<User>(apiClient.get('/auth/me'));
  }
};
