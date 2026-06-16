<script setup lang="ts">
import { computed, reactive, ref } from 'vue';
import { Message } from '@arco-design/web-vue';
import { useRoute, useRouter } from 'vue-router';
import { userApi } from '../api/user';
import { useAuthStore } from '../stores/auth';
import { usePublicConfigStore } from '../stores/publicConfig';
import { roleLabel } from '../utils/display';
import bitaiLogo from '../assets/bitai.svg';
import yunLogo from '../assets/yun.svg';

const route = useRoute();
const router = useRouter();
const auth = useAuthStore();
const config = usePublicConfigStore();

const selected = computed(() => [route.path]);
const collapsed = ref(false);
const profileVisible = ref(false);
const savingProfile = ref(false);
const uploadingAvatar = ref(false);
const profileForm = reactive({
  display_name: '',
  avatar_url: ''
});
const userName = computed(() => auth.user?.display_name || auth.user?.email || '用户');
const avatarText = computed(() => userName.value.slice(0, 1).toUpperCase());
const siteName = computed(() => config.text('site.name', 'BitAPI'));
const sidebarLogo = computed(() => config.text('site.logo_url', '') || bitaiLogo);
const collapsedLogo = computed(() => config.text('site.logo_collapsed_url', '') || yunLogo);

function go(key: string | number) {
  router.push(String(key));
}

function logout() {
  auth.logout();
  router.push('/auth/login');
}

function openProfile() {
  profileForm.display_name = auth.user?.display_name || '';
  profileForm.avatar_url = auth.user?.avatar_url || '';
  profileVisible.value = true;
}

async function saveProfile() {
  savingProfile.value = true;
  try {
    await auth.updateProfile({
      display_name: profileForm.display_name,
      avatar_url: profileForm.avatar_url
    });
    Message.success('个人资料已保存');
    profileVisible.value = false;
  } catch (error: any) {
    Message.error(error?.response?.data?.message || '保存个人资料失败');
  } finally {
    savingProfile.value = false;
  }
}

function uploadAvatar(option: any) {
  uploadingAvatar.value = true;
  void (async () => {
    try {
      const result = await userApi.uploadAvatar(option.fileItem?.file || option.file);
      profileForm.avatar_url = result.avatar_url;
      Message.success('头像已上传');
      option.onSuccess?.(result);
    } catch (error: any) {
      Message.error(error?.response?.data?.message || '上传头像失败');
      option.onError?.(error);
    } finally {
      uploadingAvatar.value = false;
    }
  })();
  return { abort() {} };
}
</script>

<template>
  <a-layout class="app-shell">
    <a-layout-sider v-model:collapsed="collapsed" :width="282" collapsible breakpoint="lg">
      <div class="side-brand">
        <button class="side-logo-button" type="button" @click="router.push('/')">
          <img :class="collapsed ? 'side-logo side-logo-collapsed' : 'side-logo'" :src="collapsed ? collapsedLogo : sidebarLogo" :alt="siteName" />
        </button>
      </div>
      <a-menu :selected-keys="selected" @menu-item-click="go">
        <a-menu-item v-if="config.enabled('module.user.dashboard.enabled', true)" key="/dashboard"><template #icon><icon-dashboard /></template>控制台</a-menu-item>
        <a-menu-item v-if="config.enabled('module.user.api_keys.enabled', true)" key="/api-keys"><template #icon><icon-safe /></template>调用密钥</a-menu-item>
        <a-menu-item v-if="config.enabled('module.user.billing.enabled', true)" key="/billing"><template #icon><icon-gift /></template>费用中心</a-menu-item>
        <a-menu-item v-if="config.enabled('module.user.usage.enabled', true)" key="/usage"><template #icon><icon-bar-chart /></template>使用明细</a-menu-item>
        <a-menu-item v-if="auth.isAdmin" key="/admin"><template #icon><icon-settings /></template>管理后台</a-menu-item>
      </a-menu>
    </a-layout-sider>
    <a-layout>
      <a-layout-header class="topbar">
        <button class="profile-chip" type="button" @click="openProfile">
          <a-avatar :size="32" :image-url="auth.user?.avatar_url">{{ avatarText }}</a-avatar>
          <span>{{ userName }}</span>
          <a-tag color="arcoblue">{{ roleLabel(auth.user?.role) }}</a-tag>
        </button>
        <a-space>
          <a-button type="text" @click="openProfile"><template #icon><icon-user /></template>个人资料</a-button>
          <a-button type="text" @click="logout"><template #icon><icon-export /></template>退出登录</a-button>
        </a-space>
      </a-layout-header>
      <a-layout-content class="content">
        <router-view />
      </a-layout-content>
    </a-layout>

    <a-modal v-model:visible="profileVisible" title="个人资料" :confirm-loading="savingProfile" @ok="saveProfile">
      <a-form layout="vertical" :model="profileForm">
        <a-form-item label="用户名" required>
          <a-input v-model="profileForm.display_name" placeholder="请输入用户名" :max-length="60" show-word-limit />
        </a-form-item>
        <a-form-item label="头像地址">
          <a-input v-model="profileForm.avatar_url" placeholder="https://example.com/avatar.png" />
        </a-form-item>
        <div class="avatar-preview">
          <a-avatar :size="56" :image-url="profileForm.avatar_url">{{ (profileForm.display_name || userName).slice(0, 1).toUpperCase() }}</a-avatar>
          <div class="avatar-actions">
            <a-upload :show-file-list="false" accept="image/png,image/jpeg,image/gif,image/webp" :custom-request="uploadAvatar">
              <template #upload-button>
                <a-button :loading="uploadingAvatar">
                  <template #icon><icon-upload /></template>
                  上传头像
                </a-button>
              </template>
            </a-upload>
            <span>头像和用户名会同步显示在主页、控制台和后台顶部。</span>
          </div>
        </div>
      </a-form>
    </a-modal>
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

.side-logo-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  border: 0;
  background: transparent;
  cursor: pointer;
}

.side-logo-collapsed {
  width: 34px;
  height: 34px;
  flex-basis: 34px;
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

.profile-chip {
  height: 40px;
  display: inline-flex;
  align-items: center;
  gap: 10px;
  padding: 0 10px 0 4px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  cursor: pointer;
  color: var(--bitapi-text);
  font: inherit;
}

.profile-chip:hover {
  background: #f2f5fa;
}

.avatar-preview {
  display: flex;
  align-items: center;
  gap: 12px;
  color: var(--bitapi-muted);
}

.avatar-actions {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.content {
  padding: 22px;
}
</style>
