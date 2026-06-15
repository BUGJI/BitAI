import axios from 'axios';
import type { ApiEnvelope } from '../types';

export const apiClient = axios.create({
  baseURL: '/api/v1',
  timeout: 30000
});

apiClient.interceptors.request.use((config) => {
  const token = localStorage.getItem('bitapi.access_token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

export async function unwrap<T>(request: Promise<{ data: ApiEnvelope<T> }>): Promise<T> {
  const response = await request;
  return response.data.data;
}
