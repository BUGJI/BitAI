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

function go(key: string | number) {
  router.push(String(key));
}

function logout() {
  auth.logout();
  router.push('/auth/login');
}
</script>

<template>
  <a-layout class="app-shell">
    <a-layout-sider :width="282" collapsible breakpoint="lg">
      <div class="side-brand">
        <img class="side-logo" :src="bitaiLogo" alt="BitAPI" />
      </div>
      <a-menu :selected-keys="selected" @menu-item-click="go">
        <a-menu-item key="/dashboard"><template #icon><icon-dashboard /></template>控制台</a-menu-item>
        <a-menu-item key="/api-keys"><template #icon><icon-safe /></template>调用密钥</a-menu-item>
        <a-menu-item key="/billing"><template #icon><icon-gift /></template>费用中心</a-menu-item>
        <a-menu-item key="/usage"><template #icon><icon-bar-chart /></template>使用明细</a-menu-item>
        <a-menu-item v-if="auth.isAdmin" key="/admin"><template #icon><icon-settings /></template>管理后台</a-menu-item>
      </a-menu>
    </a-layout-sider>
    <a-layout>
      <a-layout-header class="topbar">
        <a-space>
          <a-tag color="arcoblue">{{ roleLabel(auth.user?.role) }}</a-tag>
          <span>{{ auth.user?.display_name || auth.user?.email }}</span>
        </a-space>
        <a-button type="text" @click="logout"><template #icon><icon-export /></template>退出登录</a-button>
      </a-layout-header>
      <a-layout-content class="content">
        <router-view />
      </a-layout-content>
    </a-layout>
  </a-layout>
</template>

<style scoped>
.app-shell {
  min-height: 100vh;
}

.side-brand {
  height: 76px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 20px;
}

.side-logo {
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
