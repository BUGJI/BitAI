import { apiClient, unwrap } from '../client';
import type { Group, GroupAccount, PaymentOrder, RedeemCode, Setting, UpstreamAccount, UsageLog, User } from '../../types';

export interface AdminStats {
  users: number;
  api_keys: number;
  groups: number;
  accounts: number;
  requests: number;
  charged_micros: number;
}

export const adminApi = {
  stats() {
    return unwrap<AdminStats>(apiClient.get('/admin/stats'));
  },
  users() {
    return unwrap<User[]>(apiClient.get('/admin/users'));
  },
  updateUser(id: number, payload: Partial<User>) {
    return unwrap<User>(apiClient.patch(`/admin/users/${id}`, payload));
  },
  rechargeUser(id: number, amountMicros: number) {
    return unwrap<User>(apiClient.post(`/admin/users/${id}/recharge`, { amount_micros: amountMicros }));
  },
  groups() {
    return unwrap<Group[]>(apiClient.get('/admin/groups'));
  },
  createGroup(payload: Partial<Group>) {
    return unwrap<Group>(apiClient.post('/admin/groups', payload));
  },
  updateGroup(id: number, payload: Partial<Group>) {
    return unwrap<Group>(apiClient.patch(`/admin/groups/${id}`, payload));
  },
  deleteGroup(id: number) {
    return unwrap<{ deleted: boolean }>(apiClient.delete(`/admin/groups/${id}`));
  },
  accounts() {
    return unwrap<UpstreamAccount[]>(apiClient.get('/admin/upstream-accounts'));
  },
  createAccount(payload: Partial<UpstreamAccount>) {
    return unwrap<UpstreamAccount>(apiClient.post('/admin/upstream-accounts', payload));
  },
  updateAccount(id: number, payload: Partial<UpstreamAccount>) {
    return unwrap<UpstreamAccount>(apiClient.patch(`/admin/upstream-accounts/${id}`, payload));
  },
  deleteAccount(id: number) {
    return unwrap<{ deleted: boolean }>(apiClient.delete(`/admin/upstream-accounts/${id}`));
  },
  checkAccount(id: number) {
    return unwrap<{ account_id: number; ok: boolean; status: number; error?: string; latency_ms: number }>(
      apiClient.post(`/admin/upstream-accounts/${id}/check`)
    );
  },
  groupAccounts() {
    return unwrap<GroupAccount[]>(apiClient.get('/admin/group-accounts'));
  },
  linkGroupAccount(payload: { group_id: number; upstream_account_id: number; weight?: number; priority?: number }) {
    return unwrap(apiClient.post('/admin/group-accounts', payload));
  },
  updateGroupAccount(id: number, payload: Partial<GroupAccount>) {
    return unwrap(apiClient.patch(`/admin/group-accounts/${id}`, payload));
  },
  deleteGroupAccount(id: number) {
    return unwrap<{ deleted: boolean }>(apiClient.delete(`/admin/group-accounts/${id}`));
  },
  usage(limit = 200) {
    return unwrap<UsageLog[]>(apiClient.get('/admin/usage', { params: { limit } }));
  },
  settings() {
    return unwrap<Setting[]>(apiClient.get('/admin/settings'));
  },
  upsertSetting(payload: Pick<Setting, 'key' | 'value' | 'is_public'>) {
    return unwrap<Setting>(apiClient.post('/admin/settings', payload));
  },
  uploadBrandingAsset(file: File) {
    const form = new FormData();
    form.append('file', file);
    return unwrap<{ url: string }>(
      apiClient.post('/admin/branding/upload', form, {
        headers: { 'Content-Type': 'multipart/form-data' }
      })
    );
  },
  orders() {
    return unwrap<PaymentOrder[]>(apiClient.get('/admin/orders'));
  },
  markOrderPaid(id: number) {
    return unwrap<PaymentOrder>(apiClient.post(`/admin/orders/${id}/mark-paid`));
  },
  rejectOrder(id: number) {
    return unwrap<PaymentOrder>(apiClient.post(`/admin/orders/${id}/reject`));
  },
  redeemCodes() {
    return unwrap<RedeemCode[]>(apiClient.get('/admin/redeem-codes'));
  },
  createRedeemCode(payload: { amount_micros: number; max_uses?: number; expires_at?: string }) {
    return unwrap<{ code: string; item: RedeemCode }>(apiClient.post('/admin/redeem-codes', payload));
  },
  disableRedeemCode(id: number) {
    return unwrap<RedeemCode>(apiClient.post(`/admin/redeem-codes/${id}/disable`));
  },
  enableRedeemCode(id: number) {
    return unwrap<RedeemCode>(apiClient.post(`/admin/redeem-codes/${id}/enable`));
  },
  deleteRedeemCode(id: number) {
    return unwrap<{ deleted: boolean }>(apiClient.delete(`/admin/redeem-codes/${id}`));
  }
};
