import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router';
import { useAuthStore } from '../stores/auth';

const routes: RouteRecordRaw[] = [
  {
    path: '/auth',
    component: () => import('../layouts/AuthLayout.vue'),
    children: [
      { path: 'login', name: 'login', meta: { title: '登录' }, component: () => import('../pages/auth/LoginPage.vue') },
      { path: 'register', name: 'register', meta: { title: '注册' }, component: () => import('../pages/auth/RegisterPage.vue') }
    ]
  },
  {
    path: '/',
    component: () => import('../layouts/UserLayout.vue'),
    meta: { requiresAuth: true },
    children: [
      { path: '', redirect: '/dashboard' },
      { path: 'dashboard', name: 'dashboard', meta: { title: '控制台' }, component: () => import('../pages/user/DashboardPage.vue') },
      { path: 'api-keys', name: 'api-keys', meta: { title: '调用密钥' }, component: () => import('../pages/user/ApiKeysPage.vue') },
      { path: 'billing', name: 'billing', meta: { title: '费用中心' }, component: () => import('../pages/user/BillingPage.vue') },
      { path: 'usage', name: 'usage', meta: { title: '使用明细' }, component: () => import('../pages/user/UsagePage.vue') }
    ]
  },
  {
    path: '/admin',
    component: () => import('../layouts/AdminLayout.vue'),
    meta: { requiresAuth: true, requiresAdmin: true },
    children: [
      { path: '', name: 'admin-dashboard', meta: { title: '概览' }, component: () => import('../pages/admin/AdminDashboardPage.vue') },
      { path: 'users', name: 'admin-users', meta: { title: '用户' }, component: () => import('../pages/admin/AdminUsersPage.vue') },
      { path: 'groups', name: 'admin-groups', meta: { title: '分组' }, component: () => import('../pages/admin/AdminGroupsPage.vue') },
      { path: 'accounts', name: 'admin-accounts', meta: { title: '上游账号' }, component: () => import('../pages/admin/AdminAccountsPage.vue') },
      { path: 'usage', name: 'admin-usage', meta: { title: '调用日志' }, component: () => import('../pages/admin/AdminUsagePage.vue') },
      { path: 'billing', name: 'admin-billing', meta: { title: '充值兑换' }, component: () => import('../pages/admin/AdminBillingPage.vue') },
      { path: 'settings', name: 'admin-settings', meta: { title: '系统设置' }, component: () => import('../pages/admin/AdminSettingsPage.vue') }
    ]
  },
  { path: '/:pathMatch(.*)*', redirect: '/dashboard' }
];

export const router = createRouter({
  history: createWebHistory(),
  routes
});

router.beforeEach(async (to) => {
  const auth = useAuthStore();
  if (auth.accessToken && !auth.user) {
    try {
      await auth.loadMe();
    } catch {
      auth.logout();
    }
  }
  if (to.meta.requiresAuth && !auth.isAuthenticated) {
    return { name: 'login' };
  }
  if (to.meta.requiresAdmin && !auth.isAdmin) {
    return { name: 'dashboard' };
  }
  if ((to.name === 'login' || to.name === 'register') && auth.isAuthenticated) {
    return { name: 'dashboard' };
  }
});
