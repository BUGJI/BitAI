<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { Message } from '@arco-design/web-vue';
import { useRouter } from 'vue-router';
import { userApi } from '../../api/user';
import { useAuthStore } from '../../stores/auth';
import { usePublicConfigStore } from '../../stores/publicConfig';
import bitaiLogo from '../../assets/bitai.svg';
import bitwhiteLogo from '../../assets/bitwhite.svg';
import portalBg from '../../assets/portal-bg.mp4';

interface PortalMetric {
  label: string;
  value: string;
}

interface PortalNavItem {
  label: string;
  target: string;
  icon?: string;
}

interface PortalAction {
  label: string;
  target: string;
  icon?: string;
  type?: 'primary' | 'secondary';
}

interface PortalSectionItem {
  title?: string;
  label?: string;
  value?: string;
  description?: string;
  icon?: string;
  badge?: string;
  target?: string;
}

interface PortalSection {
  id: string;
  type: 'feature-grid' | 'timeline' | 'model-grid' | 'cta';
  eyebrow?: string;
  title: string;
  description?: string;
  icon?: string;
  items?: PortalSectionItem[];
  actions?: PortalAction[];
}

interface PortalFooter {
  copyright: string;
  qrcodeTitle?: string;
  qrcodeDescription?: string;
  qrcodeImage?: string;
  links?: PortalAction[];
  friendLinks?: PortalAction[];
}

const defaultNav: PortalNavItem[] = [
  { label: '首页', target: '#top', icon: 'home' },
  { label: '核心能力', target: '#features', icon: 'apps' },
  { label: '接入流程', target: '#workflow', icon: 'branch' },
  { label: '模型能力', target: '#models', icon: 'storage' },
  { label: '费用中心', target: '/billing', icon: 'gift' }
];

const defaultMetrics: PortalMetric[] = [
  { label: '兼容接口', value: 'OpenAI' },
  { label: '计费方式', value: '余额扣费' },
  { label: '部署形态', value: '私有化' },
  { label: '运维视角', value: '可观测' }
];

const defaultSections: PortalSection[] = [
  {
    id: 'features',
    type: 'feature-grid',
    eyebrow: '核心能力',
    title: '把模型调用、额度、密钥和账单收进一个入口',
    description: '面向团队和企业的 AI API Gateway，提供统一入口、模型路由、精细计费和可观测运营能力。',
    items: [
      { icon: 'apps', title: '统一模型入口', description: '将多个上游账号聚合成稳定的 OpenAI 兼容接口，业务侧只需要接入一次。' },
      { icon: 'branch', title: '模型路由策略', description: '按分组、模型、优先级和权重调度上游账号，减少人工切换成本。' },
      { icon: 'gift', title: '余额计费体系', description: '记录令牌、耗时、扣费和充值流水，支持兑换码与多渠道支付。' },
      { icon: 'safe', title: '密钥与权限', description: '用户密钥、管理员权限、注册验证和邮箱验证码集中管理。' }
    ]
  },
  {
    id: 'workflow',
    type: 'timeline',
    eyebrow: '接入流程',
    title: '从上游账号到业务调用，四步完成交付',
    description: '适合内部团队、代理平台和私有化部署场景，配置后即可给用户签发调用密钥。',
    items: [
      { icon: 'cloud', title: '接入上游账号', description: '录入主账号、代理地址和模型列表，统一维护可用资源池。' },
      { icon: 'settings', title: '配置分组策略', description: '设置模型映射、倍率、额度限制和调用优先级。' },
      { icon: 'lock', title: '签发调用密钥', description: '用户在控制台创建密钥，按 OpenAI 兼容方式调用。' },
      { icon: 'bar-chart', title: '持续运营观测', description: '后台查看日志、消耗、充值订单和上游状态。' }
    ]
  },
  {
    id: 'models',
    type: 'model-grid',
    eyebrow: '模型能力',
    title: '兼容主流 OpenAI 风格调用',
    description: '面向聊天、响应式接口、流式输出和模型列表管理，后续可按业务继续扩展适配层。',
    items: [
      { icon: 'message', title: 'Chat Completions', description: '支持常见聊天补全接口和流式响应转发。' },
      { icon: 'code-square', title: 'Responses API', description: '提供统一响应入口，方便 Codex 等客户端接入。' },
      { icon: 'list', title: 'Models', description: '按分组返回可用模型，隐藏上游账号复杂度。' },
      { icon: 'thunderbolt', title: '高可用路由', description: '通过优先级、权重和状态控制提高调用稳定性。' }
    ]
  },
  {
    id: 'contact',
    type: 'cta',
    eyebrow: '开始使用',
    title: '让团队用一个地址调用所有模型',
    description: '进入控制台配置上游账号、创建分组并签发调用密钥。',
    actions: [
      { label: '进入控制台', target: '/dashboard', icon: 'right', type: 'primary' },
      { label: '查看费用中心', target: '/billing', icon: 'gift', type: 'secondary' }
    ]
  }
];

const defaultFooter: PortalFooter = {
  copyright: 'Copyright © 2026 BitAPI. All rights reserved.',
  qrcodeTitle: '联系与社群',
  qrcodeDescription: '可在后台配置二维码图片地址',
  qrcodeImage: '',
  links: [
    { label: '登录', target: '/auth/login' },
    { label: '注册', target: '/auth/register' },
    { label: '控制台', target: '/dashboard' }
  ],
  friendLinks: [
    { label: 'Arco Design', target: 'https://arco.design/vue' },
    { label: 'Gin', target: 'https://gin-gonic.com' },
    { label: 'GORM', target: 'https://gorm.io' }
  ]
};

const router = useRouter();
const auth = useAuthStore();
const config = usePublicConfigStore();
const profileVisible = ref(false);
const savingProfile = ref(false);
const uploadingAvatar = ref(false);
const profileForm = reactive({
  display_name: '',
  avatar_url: ''
});

const title = computed(() => config.text('portal.hero.title', '企业级 AI API 网关'));
const subtitle = computed(() => config.text('portal.hero.subtitle', '统一接入模型供应商，集中管理密钥、路由、额度、计费与运营。'));
const heroTag = computed(() => config.text('portal.hero.tag', 'AI API Gateway'));
const primaryText = computed(() => config.text('portal.hero.primary_text', '进入控制台'));
const secondaryText = computed(() => config.text('portal.hero.secondary_text', '创建账号'));
const primaryTarget = computed(() => config.text('portal.hero.primary_target', ''));
const secondaryTarget = computed(() => config.text('portal.hero.secondary_target', '/auth/register'));
const siteName = computed(() => config.text('site.name', 'BitAPI'));
const siteLogo = computed(() => config.text('site.logo_url', '') || bitaiLogo);
const siteDarkLogo = computed(() => config.text('site.logo_dark_url', '') || bitwhiteLogo);
const registerEnabled = computed(() => config.enabled('module.auth.register.enabled', true));
const userName = computed(() => auth.user?.display_name || auth.user?.email || '');
const avatarText = computed(() => (userName.value || '用').slice(0, 1).toUpperCase());

const navItems = computed<PortalNavItem[]>(() => parseList(config.settings['portal.nav'], defaultNav));
const metrics = computed<PortalMetric[]>(() => parseList(config.settings['portal.metrics'], defaultMetrics));
const sections = computed<PortalSection[]>(() => parseList(config.settings['portal.sections'], defaultSections));
const footer = computed<PortalFooter>(() => parseObject(config.settings['portal.footer'], defaultFooter));
const siteCopyright = computed(() => config.text('site.copyright', footer.value.copyright));

function parseList<T>(raw: string | undefined, fallback: T[]) {
  if (!raw) return fallback;
  try {
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? parsed : fallback;
  } catch {
    return fallback;
  }
}

function parseObject<T extends object>(raw: string | undefined, fallback: T) {
  if (!raw) return fallback;
  try {
    const parsed = JSON.parse(raw);
    return parsed && typeof parsed === 'object' && !Array.isArray(parsed) ? { ...fallback, ...parsed } : fallback;
  } catch {
    return fallback;
  }
}

function iconName(icon?: string) {
  const aliases: Record<string, string> = {
    wallet: 'gift',
    key: 'lock'
  };
  const value = aliases[(icon || '').trim()] || (icon || '').trim();
  if (!value) return 'icon-check-circle';
  if (value.startsWith('icon-')) return value;
  return `icon-${value.replace(/([a-z0-9])([A-Z])/g, '$1-$2').replace(/_/g, '-').toLowerCase()}`;
}

function actionClass(action: PortalAction) {
  return {
    'cta-primary-action': action.type === 'primary',
    'cta-secondary-action': action.type !== 'primary'
  };
}

function isExternalTarget(target: string) {
  return /^(https?:)?\/\//.test(target) || target.startsWith('mailto:') || target.startsWith('tel:');
}

function navigate(target?: string) {
  const value = (target || '').trim();
  if (!value) return;
  if (value.startsWith('#')) {
    const idTarget = document.getElementById(value.slice(1));
    if (idTarget) {
      idTarget.scrollIntoView({ behavior: 'smooth', block: 'start' });
      return;
    }
    try {
      document.querySelector(value)?.scrollIntoView({ behavior: 'smooth', block: 'start' });
    } catch {
      // Ignore invalid custom selectors entered in settings.
    }
    return;
  }
  if (isExternalTarget(value)) {
    if (value.startsWith('mailto:') || value.startsWith('tel:')) {
      window.location.href = value;
      return;
    }
    window.open(value, '_blank', 'noopener,noreferrer');
    return;
  }
  router.push(value);
}

function enterApp() {
  if (primaryTarget.value) {
    navigate(primaryTarget.value);
    return;
  }
  router.push(auth.isAuthenticated ? '/dashboard' : '/auth/login');
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

onMounted(async () => {
  if (auth.accessToken && !auth.user) {
    try {
      await auth.loadMe();
    } catch {
      auth.logout();
    }
  }
});
</script>

<template>
  <main class="portal">
    <header class="portal-nav">
      <div class="nav-inner">
        <button class="brand-button" type="button" @click="navigate('#top')">
          <img class="portal-logo" :src="siteLogo" :alt="siteName" />
        </button>
        <nav class="portal-menu" aria-label="主页导航">
          <button v-for="item in navItems" :key="`${item.label}-${item.target}`" class="nav-link" type="button" @click="navigate(item.target)">
            <component :is="iconName(item.icon)" />
            <span>{{ item.label }}</span>
          </button>
        </nav>
        <div class="nav-actions">
          <button v-if="auth.isAuthenticated && auth.user" class="portal-user" type="button" @click="openProfile">
            <a-avatar :size="32" :image-url="auth.user.avatar_url">{{ avatarText }}</a-avatar>
            <span>{{ userName }}</span>
          </button>
          <a-space v-else>
            <a-button type="text" @click="router.push('/auth/login')">登录</a-button>
            <a-button v-if="registerEnabled" type="primary" @click="router.push('/auth/register')">注册</a-button>
          </a-space>
        </div>
      </div>
    </header>

    <section id="top" class="hero-band">
      <video class="hero-video" :src="portalBg" autoplay muted loop playsinline></video>
      <div class="hero-overlay"></div>
      <div class="hero-inner">
        <div class="hero-copy">
          <a-tag color="arcoblue">{{ heroTag }}</a-tag>
          <h1>{{ title }}</h1>
          <p>{{ subtitle }}</p>
          <a-space wrap>
            <a-button type="primary" size="large" @click="enterApp">
              <template #icon><icon-right /></template>
              {{ primaryText }}
            </a-button>
            <a-button v-if="registerEnabled" size="large" @click="navigate(secondaryTarget)">
              <template #icon><icon-user-add /></template>
              {{ secondaryText }}
            </a-button>
          </a-space>
        </div>
      </div>
    </section>

    <section id="metrics" class="metric-band">
      <div v-for="item in metrics" :key="item.label" class="metric-item">
        <strong>{{ item.value }}</strong>
        <span>{{ item.label }}</span>
      </div>
    </section>

    <section
      v-for="(section, index) in sections"
      :id="section.id"
      :key="section.id"
      class="content-section"
      :class="[`section-${section.type}`, { 'section-alt': index % 2 === 1 }]"
    >
      <div class="section-inner">
        <div class="section-heading">
          <div v-if="section.eyebrow" class="section-eyebrow">
            <component :is="iconName(section.icon)" />
            <span>{{ section.eyebrow }}</span>
          </div>
          <h2>{{ section.title }}</h2>
          <p v-if="section.description">{{ section.description }}</p>
        </div>

        <div v-if="section.type === 'feature-grid'" class="feature-grid">
          <article v-for="item in section.items" :key="item.title" class="feature-card">
            <div class="icon-tile"><component :is="iconName(item.icon)" /></div>
            <h3>{{ item.title }}</h3>
            <p>{{ item.description }}</p>
          </article>
        </div>

        <div v-else-if="section.type === 'timeline'" class="timeline-grid">
          <article v-for="item in section.items" :key="item.title" class="timeline-item">
            <div class="icon-tile"><component :is="iconName(item.icon)" /></div>
            <h3>{{ item.title }}</h3>
            <p>{{ item.description }}</p>
          </article>
        </div>

        <div v-else-if="section.type === 'model-grid'" class="model-grid">
          <article v-for="item in section.items" :key="item.title" class="model-card">
            <div class="model-card-head">
              <div class="icon-tile"><component :is="iconName(item.icon)" /></div>
            </div>
            <h3>{{ item.title }}</h3>
            <p>{{ item.description }}</p>
          </article>
        </div>

        <div v-else-if="section.type === 'cta'" class="cta-panel">
          <a-space wrap>
            <a-button
              v-for="action in section.actions"
              :key="`${action.label}-${action.target}`"
              :type="action.type === 'primary' ? 'primary' : 'secondary'"
              :class="actionClass(action)"
              size="large"
              @click="navigate(action.target)"
            >
              <template #icon><component :is="iconName(action.icon)" /></template>
              {{ action.label }}
            </a-button>
          </a-space>
        </div>
      </div>
    </section>

    <footer class="portal-footer">
      <div class="footer-inner">
        <div class="footer-brand">
          <img class="footer-logo" :src="siteDarkLogo" :alt="siteName" />
          <p>{{ siteCopyright }}</p>
        </div>

        <div class="footer-column">
          <h3>链接</h3>
          <button v-for="item in footer.links" :key="`${item.label}-${item.target}`" type="button" @click="navigate(item.target)">
            {{ item.label }}
          </button>
        </div>

        <div class="footer-column">
          <h3>友链</h3>
          <button v-for="item in footer.friendLinks" :key="`${item.label}-${item.target}`" type="button" @click="navigate(item.target)">
            {{ item.label }}
          </button>
        </div>

        <div class="footer-qrcode">
          <div class="qrcode-box">
            <img v-if="footer.qrcodeImage" :src="footer.qrcodeImage" :alt="footer.qrcodeTitle || '二维码'" />
            <icon-qrcode v-else />
          </div>
          <strong>{{ footer.qrcodeTitle }}</strong>
          <span>{{ footer.qrcodeDescription }}</span>
        </div>
      </div>
    </footer>

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
  </main>
</template>

<style scoped>
.portal {
  min-height: 100vh;
  color: var(--bitapi-text);
  background: #f5f7fb;
}

.portal-nav {
  position: sticky;
  top: 0;
  z-index: 20;
  background: rgba(255, 255, 255, 0.94);
  border-bottom: 1px solid var(--bitapi-border);
  backdrop-filter: blur(14px);
}

.nav-inner {
  min-height: 72px;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 24px;
  align-items: center;
  padding: 0 7vw;
}

.brand-button,
.nav-link,
.footer-column button {
  border: 0;
  background: transparent;
  cursor: pointer;
  font: inherit;
}

.brand-button {
  display: inline-flex;
  align-items: center;
  padding: 0;
}

.portal-logo,
.footer-logo {
  width: 160px;
  height: 34px;
  object-fit: contain;
}

.portal-menu {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-width: 0;
}

.nav-link {
  height: 36px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 0 12px;
  border-radius: 8px;
  color: #3f4a5a;
  white-space: nowrap;
}

.nav-link:hover {
  color: var(--bitapi-brand);
  background: #eef4ff;
}

.nav-link svg {
  font-size: 16px;
}

.nav-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
}

.portal-user {
  height: 40px;
  max-width: 220px;
  display: inline-flex;
  align-items: center;
  gap: 10px;
  padding: 0 12px 0 4px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--bitapi-text);
  cursor: pointer;
  font: inherit;
}

.portal-user:hover {
  background: #eef4ff;
}

.portal-user span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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

.hero-band,
.content-section {
  scroll-margin-top: 72px;
}

.hero-band {
  position: relative;
  overflow: hidden;
  min-height: 620px;
  display: flex;
  align-items: center;
  padding: 64px 7vw 54px;
  background: #0f172a;
}

.hero-video {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.hero-overlay {
  position: absolute;
  inset: 0;
  background:
    linear-gradient(90deg, rgba(4, 10, 24, 0.86), rgba(4, 10, 24, 0.58) 46%, rgba(4, 10, 24, 0.25)),
    linear-gradient(180deg, rgba(4, 10, 24, 0.1), rgba(4, 10, 24, 0.42));
}

.hero-inner {
  position: relative;
  z-index: 1;
  width: 100%;
  max-width: 920px;
}

.hero-copy h1 {
  max-width: 790px;
  margin: 18px 0;
  color: #fff;
  font-size: 58px;
  line-height: 1.08;
  letter-spacing: 0;
}

.hero-copy p {
  max-width: 720px;
  margin: 0 0 30px;
  color: rgba(255, 255, 255, 0.84);
  font-size: 18px;
  line-height: 1.8;
}

.metric-band {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 1px;
  background: var(--bitapi-border);
  border-top: 1px solid var(--bitapi-border);
  border-bottom: 1px solid var(--bitapi-border);
}

.metric-item {
  min-height: 124px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  padding: 24px 7vw;
  background: #fff;
}

.metric-item strong {
  font-size: 28px;
  line-height: 1.2;
}

.metric-item span {
  margin-top: 8px;
  color: var(--bitapi-muted);
}

.content-section {
  padding: 72px 7vw;
  background: #fff;
}

.section-alt {
  background: #f7f9fc;
}

.section-inner {
  max-width: 1180px;
  margin: 0 auto;
}

.section-heading {
  max-width: 760px;
  margin-bottom: 28px;
}

.section-heading h2 {
  margin: 10px 0 12px;
  font-size: 34px;
  line-height: 1.22;
  letter-spacing: 0;
}

.section-heading p {
  margin: 0;
  color: var(--bitapi-muted);
  font-size: 16px;
  line-height: 1.8;
}

.section-eyebrow {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  color: var(--bitapi-brand);
  font-weight: 650;
}

.feature-grid,
.model-grid,
.timeline-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
}

.feature-card,
.model-card,
.timeline-item {
  min-height: 210px;
  padding: 24px;
  border: 1px solid var(--bitapi-border);
  border-radius: 8px;
  background: #fff;
}

.section-alt .feature-card,
.section-alt .model-card,
.section-alt .timeline-item {
  background: #fff;
}

.feature-card h3,
.model-card h3,
.timeline-item h3 {
  margin: 18px 0 10px;
  font-size: 18px;
}

.feature-card p,
.model-card p,
.timeline-item p {
  margin: 0;
  color: var(--bitapi-muted);
  line-height: 1.7;
}

.icon-tile {
  width: 42px;
  height: 42px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 8px;
  color: var(--bitapi-brand);
  background: #eef4ff;
}

.icon-tile svg {
  font-size: 22px;
}

.timeline-item {
  position: relative;
}

.model-card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.section-cta {
  color: #fff;
  background:
    linear-gradient(120deg, rgba(22, 93, 255, 0.94), rgba(19, 194, 194, 0.86)),
    #165dff;
}

.section-cta .section-heading p,
.section-cta .section-eyebrow {
  color: rgba(255, 255, 255, 0.82);
}

.cta-panel {
  margin-top: 12px;
}

.cta-primary-action {
  color: #1d2129;
  border-color: #f2f3f5;
  background: #f2f3f5;
}

.cta-primary-action:hover {
  color: #1d2129;
  border-color: #ffffff;
  background: #ffffff;
}

.cta-secondary-action {
  color: #fff;
  border-color: rgba(242, 243, 245, 0.82);
  background: transparent;
}

.cta-secondary-action:hover {
  color: #fff;
  border-color: #ffffff;
  background: rgba(255, 255, 255, 0.12);
}

.portal-footer {
  color: rgba(255, 255, 255, 0.78);
  background: #111827;
}

.footer-inner {
  display: grid;
  grid-template-columns: minmax(260px, 1.6fr) minmax(120px, 0.7fr) minmax(120px, 0.7fr) minmax(150px, 0.8fr);
  gap: 32px;
  padding: 44px 7vw;
}

.footer-brand p {
  max-width: 420px;
  margin: 16px 0 0;
  line-height: 1.7;
}

.footer-column {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.footer-column h3 {
  margin: 0 0 4px;
  color: #fff;
  font-size: 15px;
}

.footer-column button {
  padding: 0;
  color: rgba(255, 255, 255, 0.72);
  text-align: left;
}

.footer-column button:hover {
  color: #fff;
}

.footer-qrcode {
  display: flex;
  flex-direction: column;
  gap: 8px;
  align-items: flex-start;
}

.qrcode-box {
  width: 112px;
  height: 112px;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid rgba(255, 255, 255, 0.16);
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.08);
}

.qrcode-box img {
  width: 100%;
  height: 100%;
  border-radius: 8px;
  object-fit: cover;
}

.qrcode-box svg {
  color: rgba(255, 255, 255, 0.68);
  font-size: 42px;
}

.footer-qrcode strong {
  color: #fff;
}

.footer-qrcode span {
  color: rgba(255, 255, 255, 0.62);
  line-height: 1.6;
}

@media (max-width: 1120px) {
  .nav-inner {
    grid-template-columns: 1fr auto;
  }

  .portal-menu {
    grid-column: 1 / -1;
    justify-content: flex-start;
    overflow-x: auto;
    padding-bottom: 10px;
  }

  .feature-grid,
  .model-grid,
  .timeline-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .footer-inner {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

@media (max-width: 720px) {
  .nav-inner {
    gap: 12px;
    padding: 12px 18px;
  }

  .nav-actions {
    justify-content: flex-end;
  }

  .hero-band {
    min-height: 560px;
    padding: 58px 20px 46px;
  }

  .hero-copy h1 {
    font-size: 38px;
  }

  .metric-band,
  .feature-grid,
  .model-grid,
  .timeline-grid,
  .footer-inner {
    grid-template-columns: 1fr;
  }

  .metric-item,
  .content-section {
    padding-left: 20px;
    padding-right: 20px;
  }

  .section-heading h2 {
    font-size: 28px;
  }
}
</style>
