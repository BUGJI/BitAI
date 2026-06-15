<script setup lang="ts">
import { computed } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { useAuthStore } from '../stores/auth';
import { roleLabel } from '../utils/display';
import bitaiLogo from '../assets/bitai.svg';

const route = useRoute();
const router = useRouter();
const auth = useAuthStore();
const selected = computed(() => [route.path]);
const currentTitle = computed(() => String(route.meta.title || '概览'));
const currentPath = computed(() => route.path);

function go(key: string | number) {
  router.push(String(key));
}
</script>

<template>
  <a-layout class="admin-shell">
    <a-layout-sider :width="282" collapsible breakpoint="lg">
      <div class="admin-brand">
        <img class="admin-logo" :src="bitaiLogo" alt="BitAPI" />
      </div>
      <a-menu :selected-keys="selected" @menu-item-click="go">
        <a-menu-item key="/admin"><template #icon><icon-dashboard /></template>概览</a-menu-item>
        <a-menu-item key="/admin/users"><template #icon><icon-user-group /></template>用户</a-menu-item>
        <a-menu-item key="/admin/groups"><template #icon><icon-layers /></template>分组</a-menu-item>
        <a-menu-item key="/admin/accounts"><template #icon><icon-cloud /></template>上游账号</a-menu-item>
        <a-menu-item key="/admin/usage"><template #icon><icon-history /></template>调用日志</a-menu-item>
        <a-menu-item key="/admin/billing"><template #icon><icon-gift /></template>充值兑换</a-menu-item>
        <a-menu-item key="/admin/settings"><template #icon><icon-settings /></template>系统设置</a-menu-item>
        <a-menu-item key="/dashboard"><template #icon><icon-left /></template>用户控制台</a-menu-item>
      </a-menu>
    </a-layout-sider>
    <a-layout>
      <a-layout-header class="topbar">
        <a-breadcrumb>
          <a-breadcrumb-item>
            <router-link to="/admin">管理后台</router-link>
          </a-breadcrumb-item>
          <a-breadcrumb-item>
            <router-link :to="currentPath">{{ currentTitle }}</router-link>
          </a-breadcrumb-item>
        </a-breadcrumb>
        <a-space>
          <a-tag color="red">{{ roleLabel(auth.user?.role) }}</a-tag>
          <span>{{ auth.user?.email }}</span>
        </a-space>
      </a-layout-header>
      <a-layout-content class="content">
        <router-view />
      </a-layout-content>
    </a-layout>
  </a-layout>
</template>

<style scoped>
.admin-shell {
  min-height: 100vh;
}

.admin-brand {
  height: 76px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 20px;
}

.admin-logo {
  width: 160px;
  height: 34px;
  display: block;
  flex: 0 0 160px;
  object-fit: contain;
}

.topbar {
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 22px;
  background: #fff;
  border-bottom: 1px solid var(--bitapi-border);
}

.content {
  padding: 22px;
}
</style>
