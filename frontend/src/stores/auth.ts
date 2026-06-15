import { defineStore } from 'pinia';
import { authApi, type AuthPayload } from '../api/auth';
import type { User } from '../types';

interface AuthState {
  user: User | null;
  accessToken: string;
  refreshToken: string;
}

export const useAuthStore = defineStore('auth', {
  state: (): AuthState => ({
    user: null,
    accessToken: localStorage.getItem('bitapi.access_token') || '',
    refreshToken: localStorage.getItem('bitapi.refresh_token') || ''
  }),
  getters: {
    isAuthenticated: (state) => Boolean(state.accessToken),
    isAdmin: (state) => ['owner', 'admin', 'operator'].includes(state.user?.role || '')
  },
  actions: {
    persist(accessToken: string, refreshToken: string) {
      this.accessToken = accessToken;
      this.refreshToken = refreshToken;
      localStorage.setItem('bitapi.access_token', accessToken);
      localStorage.setItem('bitapi.refresh_token', refreshToken);
    },
    async login(payload: AuthPayload) {
      const pair = await authApi.login(payload);
      this.persist(pair.access_token, pair.refresh_token);
      this.user = pair.user;
    },
    async register(payload: AuthPayload) {
      const pair = await authApi.register(payload);
      this.persist(pair.access_token, pair.refresh_token);
      this.user = pair.user;
    },
    async loadMe() {
      if (!this.accessToken) return;
      this.user = await authApi.me();
    },
    logout() {
      this.user = null;
      this.accessToken = '';
      this.refreshToken = '';
      localStorage.removeItem('bitapi.access_token');
      localStorage.removeItem('bitapi.refresh_token');
    }
  }
});
