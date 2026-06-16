<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import { Message } from '@arco-design/web-vue';
import { adminApi } from '../../api/admin';
import type { Setting } from '../../types';

type SectionType = 'feature-grid' | 'timeline' | 'model-grid' | 'cta';

interface PortalNavItem {
  label: string;
  target: string;
  icon: string;
}

interface PortalMetric {
  label: string;
  value: string;
}

interface PortalAction {
  label: string;
  target: string;
  icon: string;
  type: 'primary' | 'secondary';
}

interface PortalSectionItem {
  title: string;
  label: string;
  value: string;
  description: string;
  icon: string;
  badge: string;
  target: string;
}

interface PortalSection {
  id: string;
  type: SectionType;
  eyebrow: string;
  title: string;
  description: string;
  icon: string;
  items: PortalSectionItem[];
  actions: PortalAction[];
}

interface PortalFooter {
  copyright: string;
  qrcodeTitle: string;
  qrcodeDescription: string;
  qrcodeImage: string;
  links: PortalAction[];
  friendLinks: PortalAction[];
}

const loading = ref(false);
const saving = ref(false);
const visible = ref(false);
const rows = ref<Setting[]>([]);
const form = reactive({
  key: '',
  value: '',
  is_public: false
});
const smtpForm = reactive({
  enabled: false,
  host: '',
  port: 587,
  username: '',
  password: '',
  from_email: '',
  from_name: 'BitAPI',
  encryption: 'starttls'
});
const brandForm = reactive({
  name: 'BitAPI',
  logoUrl: '',
  logoDarkUrl: '',
  logoCollapsedUrl: '',
  copyright: 'Copyright © 2026 BitAPI. All rights reserved.'
});
const uploadingBrand = reactive({
  logoUrl: false,
  logoDarkUrl: false,
  logoCollapsedUrl: false
});
const moduleForm = reactive({
  portal: true,
  register: true,
  dashboard: true,
  apiKeys: true,
  billing: true,
  usage: true,
  paymentOrder: true,
  paymentRedeem: true
});
const portalForm = reactive({
  title: '',
  subtitle: '',
  tag: '',
  primaryText: '',
  primaryTarget: '',
  secondaryText: '',
  secondaryTarget: ''
});
const portalNav = ref<PortalNavItem[]>([]);
const portalMetrics = ref<PortalMetric[]>([]);
const portalSections = ref<PortalSection[]>([]);
const portalFooter = reactive<PortalFooter>({
  copyright: '',
  qrcodeTitle: '',
  qrcodeDescription: '',
  qrcodeImage: '',
  links: [],
  friendLinks: []
});
const dragState = reactive({
  kind: '',
  from: -1
});
const paymentForm = reactive({
  public_base_url: '',
  return_frontend_url: '',
  usd_cny_rate: '7.20',
  manual_enabled: true,
  epay_enabled: false,
  epay_gateway: '',
  epay_pid: '',
  epay_key: '',
  codepay_enabled: false,
  codepay_gateway: '',
  codepay_id: '',
  codepay_key: '',
  xunhupay_enabled: false,
  xunhupay_gateway: '',
  xunhupay_appid: '',
  xunhupay_appsecret: '',
  alipay_enabled: false,
  alipay_gateway: '',
  alipay_app_id: '',
  alipay_private_key: '',
  alipay_public_key: '',
  wechat_enabled: false,
  wechat_gateway: '',
  wechat_appid: '',
  wechat_mchid: '',
  wechat_serial_no: '',
  wechat_private_key: '',
  wechat_api_v3_key: ''
});
const defaultPublicBaseURL = window.location.origin;

const smtpEncryptionOptions = [
  { label: 'STARTTLS', value: 'starttls' },
  { label: 'SSL/TLS', value: 'tls' },
  { label: '不加密', value: 'none' }
];
const sectionTypeOptions = [
  { label: '功能网格', value: 'feature-grid' },
  { label: '流程模块', value: 'timeline' },
  { label: '模型能力', value: 'model-grid' },
  { label: '行动按钮', value: 'cta' }
];
const actionTypeOptions = [
  { label: '主按钮', value: 'primary' },
  { label: '次按钮', value: 'secondary' }
];
const defaultPortalNav: PortalNavItem[] = [
  { label: '首页', target: '#top', icon: 'home' },
  { label: '核心能力', target: '#features', icon: 'apps' },
  { label: '接入流程', target: '#workflow', icon: 'branch' },
  { label: '模型能力', target: '#models', icon: 'storage' },
  { label: '费用中心', target: '/billing', icon: 'gift' }
];
const defaultPortalMetrics: PortalMetric[] = [
  { label: '兼容接口', value: 'OpenAI' },
  { label: '计费方式', value: '余额扣费' },
  { label: '部署形态', value: '私有化' },
  { label: '运维视角', value: '可观测' }
];
const defaultPortalSections: PortalSection[] = [
  {
    id: 'features',
    type: 'feature-grid',
    eyebrow: '核心能力',
    title: '把模型调用、额度、密钥和账单收进一个入口',
    icon: 'apps',
    description: '面向团队和企业的 AI API Gateway，提供统一入口、模型路由、精细计费和可观测运营能力。',
    items: [
      { icon: 'apps', title: '统一模型入口', label: '', value: '', badge: '', target: '', description: '将多个上游账号聚合成稳定的 OpenAI 兼容接口，业务侧只需要接入一次。' },
      { icon: 'branch', title: '模型路由策略', label: '', value: '', badge: '', target: '', description: '按分组、模型、优先级和权重调度上游账号，减少人工切换成本。' },
      { icon: 'gift', title: '余额计费体系', label: '', value: '', badge: '', target: '', description: '记录令牌、耗时、扣费和充值流水，支持兑换码与多渠道支付。' },
      { icon: 'safe', title: '密钥与权限', label: '', value: '', badge: '', target: '', description: '用户密钥、管理员权限、注册验证和邮箱验证码集中管理。' }
    ],
    actions: []
  },
  {
    id: 'workflow',
    type: 'timeline',
    eyebrow: '接入流程',
    title: '从上游账号到业务调用，四步完成交付',
    icon: 'branch',
    description: '适合内部团队、代理平台和私有化部署场景，配置后即可给用户签发调用密钥。',
    items: [
      { icon: 'cloud', title: '接入上游账号', label: '', value: '', badge: '', target: '', description: '录入主账号、代理地址和模型列表，统一维护可用资源池。' },
      { icon: 'settings', title: '配置分组策略', label: '', value: '', badge: '', target: '', description: '设置模型映射、倍率、额度限制和调用优先级。' },
      { icon: 'lock', title: '签发调用密钥', label: '', value: '', badge: '', target: '', description: '用户在控制台创建密钥，按 OpenAI 兼容方式调用。' },
      { icon: 'bar-chart', title: '持续运营观测', label: '', value: '', badge: '', target: '', description: '后台查看日志、消耗、充值订单和上游状态。' }
    ],
    actions: []
  },
  {
    id: 'models',
    type: 'model-grid',
    eyebrow: '模型能力',
    title: '兼容主流 OpenAI 风格调用',
    icon: 'storage',
    description: '面向聊天、响应式接口、流式输出和模型列表管理，后续可按业务继续扩展适配层。',
    items: [
      { icon: 'message', title: 'Chat Completions', label: '', value: '', badge: '', target: '', description: '支持常见聊天补全接口和流式响应转发。' },
      { icon: 'code-square', title: 'Responses API', label: '', value: '', badge: '', target: '', description: '提供统一响应入口，方便 Codex 等客户端接入。' },
      { icon: 'list', title: 'Models', label: '', value: '', badge: '', target: '', description: '按分组返回可用模型，隐藏上游账号复杂度。' },
      { icon: 'thunderbolt', title: '高可用路由', label: '', value: '', badge: '', target: '', description: '通过优先级、权重和状态控制提高调用稳定性。' }
    ],
    actions: []
  },
  {
    id: 'contact',
    type: 'cta',
    eyebrow: '开始使用',
    title: '让团队用一个地址调用所有模型',
    icon: 'right',
    description: '进入控制台配置上游账号、创建分组并签发调用密钥。',
    items: [],
    actions: [
      { label: '进入控制台', target: '/dashboard', icon: 'right', type: 'primary' },
      { label: '查看费用中心', target: '/billing', icon: 'gift', type: 'secondary' }
    ]
  }
];
const defaultPortalFooter: PortalFooter = {
  copyright: 'Copyright © 2026 BitAPI. All rights reserved.',
  qrcodeTitle: '联系与社群',
  qrcodeDescription: '可在后台配置二维码图片地址',
  qrcodeImage: '',
  links: [
    { label: '登录', target: '/auth/login', icon: 'link', type: 'secondary' },
    { label: '注册', target: '/auth/register', icon: 'link', type: 'secondary' },
    { label: '控制台', target: '/dashboard', icon: 'link', type: 'secondary' }
  ],
  friendLinks: [
    { label: 'Arco Design', target: 'https://arco.design/vue', icon: 'link', type: 'secondary' },
    { label: 'Gin', target: 'https://gin-gonic.com', icon: 'link', type: 'secondary' },
    { label: 'GORM', target: 'https://gorm.io', icon: 'link', type: 'secondary' }
  ]
};

function settingValue(key: string, fallback = '') {
  return rows.value.find((item) => item.key === key)?.value ?? fallback;
}

function clone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value));
}

function parseList<T>(raw: string, fallback: T[], normalize: (item: any) => T) {
  if (!raw) return clone(fallback);
  try {
    const parsed = JSON.parse(raw);
    return Array.isArray(parsed) ? parsed.map(normalize) : clone(fallback);
  } catch {
    return clone(fallback);
  }
}

function parseFooter(raw: string) {
  if (!raw) return clone(defaultPortalFooter);
  try {
    const parsed = JSON.parse(raw);
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) return clone(defaultPortalFooter);
    return normalizeFooter(parsed);
  } catch {
    return clone(defaultPortalFooter);
  }
}

function normalizeNavItem(item: any): PortalNavItem {
  return {
    label: String(item?.label || ''),
    target: String(item?.target || ''),
    icon: String(item?.icon || '')
  };
}

function normalizeMetric(item: any): PortalMetric {
  return {
    label: String(item?.label || ''),
    value: String(item?.value || '')
  };
}

function normalizeAction(item: any): PortalAction {
  return {
    label: String(item?.label || ''),
    target: String(item?.target || ''),
    icon: String(item?.icon || ''),
    type: item?.type === 'primary' ? 'primary' : 'secondary'
  };
}

function normalizeSectionItem(item: any): PortalSectionItem {
  return {
    title: String(item?.title || ''),
    label: String(item?.label || ''),
    value: String(item?.value || ''),
    description: String(item?.description || ''),
    icon: String(item?.icon || ''),
    badge: String(item?.badge || ''),
    target: String(item?.target || '')
  };
}

function normalizeSection(item: any): PortalSection {
  const typeOptions = sectionTypeOptions.map((option) => option.value);
  const type = typeOptions.includes(item?.type) ? item.type : 'feature-grid';
  return {
    id: String(item?.id || `section-${Date.now()}`),
    type,
    eyebrow: String(item?.eyebrow || ''),
    title: String(item?.title || ''),
    description: String(item?.description || ''),
    icon: String(item?.icon || ''),
    items: Array.isArray(item?.items) ? item.items.map(normalizeSectionItem) : [],
    actions: Array.isArray(item?.actions) ? item.actions.map(normalizeAction) : []
  };
}

function normalizeFooter(item: any): PortalFooter {
  return {
    copyright: String(item?.copyright || defaultPortalFooter.copyright),
    qrcodeTitle: String(item?.qrcodeTitle || ''),
    qrcodeDescription: String(item?.qrcodeDescription || ''),
    qrcodeImage: String(item?.qrcodeImage || ''),
    links: Array.isArray(item?.links) ? item.links.map(normalizeAction) : [],
    friendLinks: Array.isArray(item?.friendLinks) ? item.friendLinks.map(normalizeAction) : []
  };
}

function compactAction(action: PortalAction) {
  return {
    label: action.label,
    target: action.target,
    icon: action.icon,
    type: action.type
  };
}

function compactSectionItem(item: PortalSectionItem) {
  const output: Record<string, string> = {};
  for (const key of ['icon', 'title', 'label', 'value', 'description', 'badge', 'target'] as const) {
    if (item[key]) output[key] = item[key];
  }
  return output;
}

function compactSection(section: PortalSection) {
  const output: Record<string, any> = {
    id: section.id,
    type: section.type,
    eyebrow: section.eyebrow,
    title: section.title,
    description: section.description,
    icon: section.icon
  };
  if (section.items.length) output.items = section.items.map(compactSectionItem);
  if (section.actions.length) output.actions = section.actions.map(compactAction);
  return output;
}

function syncSMTPForm() {
  smtpForm.enabled = settingValue('smtp.enabled', 'false') === 'true';
  smtpForm.host = settingValue('smtp.host');
  smtpForm.port = Number(settingValue('smtp.port', '587')) || 587;
  smtpForm.username = settingValue('smtp.username');
  smtpForm.password = settingValue('smtp.password');
  smtpForm.from_email = settingValue('smtp.from_email');
  smtpForm.from_name = settingValue('smtp.from_name', 'BitAPI');
  smtpForm.encryption = settingValue('smtp.encryption', 'starttls');
}

function syncBrandForm() {
  const footerValue = parseFooter(settingValue('portal.footer'));
  brandForm.name = settingValue('site.name', 'BitAPI');
  brandForm.logoUrl = settingValue('site.logo_url');
  brandForm.logoDarkUrl = settingValue('site.logo_dark_url');
  brandForm.logoCollapsedUrl = settingValue('site.logo_collapsed_url');
  brandForm.copyright = settingValue('site.copyright', footerValue.copyright);
}

function syncModuleForm() {
  moduleForm.portal = settingValue('module.portal.enabled', 'true') === 'true';
  moduleForm.register = settingValue('module.auth.register.enabled', 'true') === 'true';
  moduleForm.dashboard = settingValue('module.user.dashboard.enabled', 'true') === 'true';
  moduleForm.apiKeys = settingValue('module.user.api_keys.enabled', 'true') === 'true';
  moduleForm.billing = settingValue('module.user.billing.enabled', 'true') === 'true';
  moduleForm.usage = settingValue('module.user.usage.enabled', 'true') === 'true';
  moduleForm.paymentOrder = settingValue('module.payment.order.enabled', 'true') === 'true';
  moduleForm.paymentRedeem = settingValue('module.payment.redeem.enabled', 'true') === 'true';
}

function syncPortalForm() {
  portalForm.title = settingValue('portal.hero.title', '企业级 AI API 网关');
  portalForm.subtitle = settingValue('portal.hero.subtitle', '统一接入模型供应商，集中管理密钥、路由、额度、计费与运营。');
  portalForm.tag = settingValue('portal.hero.tag', 'AI API Gateway');
  portalForm.primaryText = settingValue('portal.hero.primary_text', '进入控制台');
  portalForm.primaryTarget = settingValue('portal.hero.primary_target', '');
  portalForm.secondaryText = settingValue('portal.hero.secondary_text', '创建账号');
  portalForm.secondaryTarget = settingValue('portal.hero.secondary_target', '/auth/register');
  portalNav.value = parseList(settingValue('portal.nav'), defaultPortalNav, normalizeNavItem);
  portalMetrics.value = parseList(settingValue('portal.metrics'), defaultPortalMetrics, normalizeMetric);
  portalSections.value = parseList(settingValue('portal.sections'), defaultPortalSections, normalizeSection);
  Object.assign(portalFooter, parseFooter(settingValue('portal.footer')));
}

function syncPaymentForm() {
  paymentForm.public_base_url = settingValue('payment.public_base_url', defaultPublicBaseURL);
  paymentForm.return_frontend_url = settingValue('payment.return_frontend_url', `${defaultPublicBaseURL}/billing`);
  paymentForm.usd_cny_rate = settingValue('payment.usd_cny_rate', '7.20');
  paymentForm.manual_enabled = settingValue('payment.manual.enabled', 'true') === 'true';
  paymentForm.epay_enabled = settingValue('payment.epay.enabled', 'false') === 'true';
  paymentForm.epay_gateway = settingValue('payment.epay.gateway', 'https://ezfp.cn');
  paymentForm.epay_pid = settingValue('payment.epay.pid');
  paymentForm.epay_key = settingValue('payment.epay.key');
  paymentForm.codepay_enabled = settingValue('payment.codepay.enabled', 'false') === 'true';
  paymentForm.codepay_gateway = settingValue('payment.codepay.gateway', 'https://codepay.fateqq.com/creat_order/');
  paymentForm.codepay_id = settingValue('payment.codepay.id');
  paymentForm.codepay_key = settingValue('payment.codepay.key');
  paymentForm.xunhupay_enabled = settingValue('payment.xunhupay.enabled', 'false') === 'true';
  paymentForm.xunhupay_gateway = settingValue('payment.xunhupay.gateway', 'https://api.xunhupay.com/payment/do.html');
  paymentForm.xunhupay_appid = settingValue('payment.xunhupay.appid');
  paymentForm.xunhupay_appsecret = settingValue('payment.xunhupay.appsecret');
  paymentForm.alipay_enabled = settingValue('payment.alipay_f2f.enabled', 'false') === 'true';
  paymentForm.alipay_gateway = settingValue('payment.alipay_f2f.gateway', 'https://openapi.alipay.com/gateway.do');
  paymentForm.alipay_app_id = settingValue('payment.alipay_f2f.app_id');
  paymentForm.alipay_private_key = settingValue('payment.alipay_f2f.private_key');
  paymentForm.alipay_public_key = settingValue('payment.alipay_f2f.alipay_public_key');
  paymentForm.wechat_enabled = settingValue('payment.wechat_native.enabled', 'false') === 'true';
  paymentForm.wechat_gateway = settingValue('payment.wechat_native.gateway', 'https://api.mch.weixin.qq.com');
  paymentForm.wechat_appid = settingValue('payment.wechat_native.appid');
  paymentForm.wechat_mchid = settingValue('payment.wechat_native.mchid');
  paymentForm.wechat_serial_no = settingValue('payment.wechat_native.serial_no');
  paymentForm.wechat_private_key = settingValue('payment.wechat_native.private_key');
  paymentForm.wechat_api_v3_key = settingValue('payment.wechat_native.api_v3_key');
}

async function load() {
  loading.value = true;
  try {
    rows.value = await adminApi.settings();
    syncBrandForm();
    syncSMTPForm();
    syncModuleForm();
    syncPortalForm();
    syncPaymentForm();
  } finally {
    loading.value = false;
  }
}

async function saveItems(items: Array<[string, string, boolean]>, successText: string) {
  saving.value = true;
  try {
    await Promise.all(items.map(([key, value, is_public]) => adminApi.upsertSetting({ key, value, is_public })));
    Message.success(successText);
    await load();
  } catch (error: any) {
    Message.error(error?.response?.data?.message || '保存失败');
  } finally {
    saving.value = false;
  }
}

function openCreate() {
  form.key = '';
  form.value = '';
  form.is_public = false;
  visible.value = true;
}

function openEdit(record: Setting) {
  form.key = record.key;
  form.value = record.value;
  form.is_public = record.is_public;
  visible.value = true;
}

function createSection(type: SectionType = 'feature-grid'): PortalSection {
  const id = `section-${Date.now()}`;
  return {
    id,
    type,
    eyebrow: '新区块',
    title: '新的展示区块',
    description: '',
    icon: 'apps',
    items: type === 'cta' ? [] : [createSectionItem()],
    actions: type === 'cta' ? [createAction('primary')] : []
  };
}

function createSectionItem(): PortalSectionItem {
  return {
    title: '新条目',
    label: '',
    value: '',
    description: '',
    icon: 'apps',
    badge: '',
    target: ''
  };
}

function createAction(type: 'primary' | 'secondary' = 'secondary'): PortalAction {
  return {
    label: type === 'primary' ? '进入控制台' : '了解更多',
    target: type === 'primary' ? '/dashboard' : '#features',
    icon: type === 'primary' ? 'right' : 'link',
    type
  };
}

function addNavItem() {
  portalNav.value.push({ label: '新菜单', target: '#top', icon: 'link' });
}

function addMetric() {
  portalMetrics.value.push({ label: '指标名称', value: '指标值' });
}

function addSection(type: SectionType = 'feature-grid') {
  portalSections.value.push(createSection(type));
}

function addFooterLink(target: PortalAction[]) {
  target.push(createAction());
}

function removeAt<T>(list: T[], index: number) {
  list.splice(index, 1);
}

function moveItem<T>(list: T[], from: number, to: number) {
  if (from === to || from < 0 || to < 0 || from >= list.length || to >= list.length) return;
  const [item] = list.splice(from, 1);
  list.splice(to, 0, item);
}

function startDrag(kind: string, index: number) {
  dragState.kind = kind;
  dragState.from = index;
}

function dropItem<T>(kind: string, list: T[], index: number) {
  if (dragState.kind === kind) {
    moveItem(list, dragState.from, index);
  }
  dragState.kind = '';
  dragState.from = -1;
}

async function save() {
  saving.value = true;
  try {
    await adminApi.upsertSetting(form);
    Message.success('设置已保存');
    visible.value = false;
    await load();
  } catch (error: any) {
    Message.error(error?.response?.data?.message || '保存设置失败');
  } finally {
    saving.value = false;
  }
}

async function saveBrand() {
  portalFooter.copyright = brandForm.copyright;
  await saveItems([
    ['site.name', brandForm.name.trim() || 'BitAPI', true],
    ['site.logo_url', brandForm.logoUrl.trim(), true],
    ['site.logo_dark_url', brandForm.logoDarkUrl.trim(), true],
    ['site.logo_collapsed_url', brandForm.logoCollapsedUrl.trim(), true],
    ['site.copyright', brandForm.copyright.trim(), true],
    ['portal.footer', JSON.stringify(normalizeFooter(portalFooter)), true]
  ], '品牌配置已保存');
}

function uploadBranding(field: 'logoUrl' | 'logoDarkUrl' | 'logoCollapsedUrl') {
  return (option: any) => {
    uploadingBrand[field] = true;
    void (async () => {
      try {
        const result = await adminApi.uploadBrandingAsset(option.fileItem?.file || option.file);
        brandForm[field] = result.url;
        Message.success('品牌图片已上传');
        option.onSuccess?.(result);
      } catch (error: any) {
        Message.error(error?.response?.data?.message || '上传品牌图片失败');
        option.onError?.(error);
      } finally {
        uploadingBrand[field] = false;
      }
    })();
    return { abort() {} };
  };
}

async function saveSMTP() {
  saving.value = true;
  try {
    const items = [
      ['smtp.enabled', smtpForm.enabled ? 'true' : 'false'],
      ['smtp.host', smtpForm.host],
      ['smtp.port', String(smtpForm.port)],
      ['smtp.username', smtpForm.username],
      ['smtp.password', smtpForm.password],
      ['smtp.from_email', smtpForm.from_email],
      ['smtp.from_name', smtpForm.from_name],
      ['smtp.encryption', smtpForm.encryption]
    ];
    await Promise.all(items.map(([key, value]) => adminApi.upsertSetting({ key, value, is_public: false })));
    Message.success('SMTP 配置已保存');
    await load();
  } catch (error: any) {
    Message.error(error?.response?.data?.message || '保存 SMTP 配置失败');
  } finally {
    saving.value = false;
  }
}

async function saveModules() {
  await saveItems([
    ['module.portal.enabled', String(moduleForm.portal), true],
    ['module.auth.register.enabled', String(moduleForm.register), true],
    ['module.user.dashboard.enabled', String(moduleForm.dashboard), true],
    ['module.user.api_keys.enabled', String(moduleForm.apiKeys), true],
    ['module.user.billing.enabled', String(moduleForm.billing), true],
    ['module.user.usage.enabled', String(moduleForm.usage), true],
    ['module.payment.order.enabled', String(moduleForm.paymentOrder), true],
    ['module.payment.redeem.enabled', String(moduleForm.paymentRedeem), true]
  ], '模块配置已保存');
}

async function savePortal() {
  await saveItems([
    ['portal.hero.title', portalForm.title, true],
    ['portal.hero.subtitle', portalForm.subtitle, true],
    ['portal.hero.tag', portalForm.tag, true],
    ['portal.hero.primary_text', portalForm.primaryText, true],
    ['portal.hero.primary_target', portalForm.primaryTarget, true],
    ['portal.hero.secondary_text', portalForm.secondaryText, true],
    ['portal.hero.secondary_target', portalForm.secondaryTarget, true],
    ['portal.nav', JSON.stringify(portalNav.value.map(normalizeNavItem)), true],
    ['portal.metrics', JSON.stringify(portalMetrics.value.map(normalizeMetric)), true],
    ['portal.sections', JSON.stringify(portalSections.value.map(compactSection)), true],
    ['site.copyright', portalFooter.copyright, true],
    ['portal.footer', JSON.stringify(normalizeFooter(portalFooter)), true],
    ['portal.features', '[]', true]
  ], '展示页配置已保存');
}

async function savePayments() {
  await saveItems([
    ['payment.public_base_url', paymentForm.public_base_url, false],
    ['payment.return_frontend_url', paymentForm.return_frontend_url, false],
    ['payment.usd_cny_rate', paymentForm.usd_cny_rate, false],
    ['payment.manual.enabled', String(paymentForm.manual_enabled), true],
    ['payment.epay.enabled', String(paymentForm.epay_enabled), true],
    ['payment.epay.gateway', paymentForm.epay_gateway, false],
    ['payment.epay.pid', paymentForm.epay_pid, false],
    ['payment.epay.key', paymentForm.epay_key, false],
    ['payment.codepay.enabled', String(paymentForm.codepay_enabled), true],
    ['payment.codepay.gateway', paymentForm.codepay_gateway, false],
    ['payment.codepay.id', paymentForm.codepay_id, false],
    ['payment.codepay.key', paymentForm.codepay_key, false],
    ['payment.xunhupay.enabled', String(paymentForm.xunhupay_enabled), true],
    ['payment.xunhupay.gateway', paymentForm.xunhupay_gateway, false],
    ['payment.xunhupay.appid', paymentForm.xunhupay_appid, false],
    ['payment.xunhupay.appsecret', paymentForm.xunhupay_appsecret, false],
    ['payment.alipay_f2f.enabled', String(paymentForm.alipay_enabled), true],
    ['payment.alipay_f2f.gateway', paymentForm.alipay_gateway, false],
    ['payment.alipay_f2f.app_id', paymentForm.alipay_app_id, false],
    ['payment.alipay_f2f.private_key', paymentForm.alipay_private_key, false],
    ['payment.alipay_f2f.alipay_public_key', paymentForm.alipay_public_key, false],
    ['payment.wechat_native.enabled', String(paymentForm.wechat_enabled), true],
    ['payment.wechat_native.gateway', paymentForm.wechat_gateway, false],
    ['payment.wechat_native.appid', paymentForm.wechat_appid, false],
    ['payment.wechat_native.mchid', paymentForm.wechat_mchid, false],
    ['payment.wechat_native.serial_no', paymentForm.wechat_serial_no, false],
    ['payment.wechat_native.private_key', paymentForm.wechat_private_key, false],
    ['payment.wechat_native.api_v3_key', paymentForm.wechat_api_v3_key, false]
  ], '支付配置已保存');
}

onMounted(load);
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h1 class="page-title">系统设置</h1>
        <p class="page-subtitle">管理系统开关和前端可读取的公开配置。</p>
      </div>
      <a-button type="primary" @click="openCreate">
        <template #icon><icon-plus /></template>
        新增设置
      </a-button>
    </div>

    <a-card title="模块化开关" :bordered="false">
      <a-form layout="vertical" :model="moduleForm">
        <a-row :gutter="16">
          <a-col :span="6"><a-form-item label="企业展示页"><a-switch v-model="moduleForm.portal" /></a-form-item></a-col>
          <a-col :span="6"><a-form-item label="用户注册"><a-switch v-model="moduleForm.register" /></a-form-item></a-col>
          <a-col :span="6"><a-form-item label="控制台"><a-switch v-model="moduleForm.dashboard" /></a-form-item></a-col>
          <a-col :span="6"><a-form-item label="调用密钥"><a-switch v-model="moduleForm.apiKeys" /></a-form-item></a-col>
          <a-col :span="6"><a-form-item label="费用中心"><a-switch v-model="moduleForm.billing" /></a-form-item></a-col>
          <a-col :span="6"><a-form-item label="使用明细"><a-switch v-model="moduleForm.usage" /></a-form-item></a-col>
          <a-col :span="6"><a-form-item label="充值订单"><a-switch v-model="moduleForm.paymentOrder" /></a-form-item></a-col>
          <a-col :span="6"><a-form-item label="兑换码充值"><a-switch v-model="moduleForm.paymentRedeem" /></a-form-item></a-col>
        </a-row>
        <a-button type="primary" :loading="saving" @click="saveModules">
          <template #icon><icon-save /></template>
          保存模块配置
        </a-button>
      </a-form>
    </a-card>

    <a-card title="产品品牌配置" :bordered="false">
      <a-form layout="vertical" :model="brandForm">
        <a-row :gutter="16">
          <a-col :span="8">
            <a-form-item label="站点名称">
              <a-input v-model="brandForm.name" placeholder="例如 FitAI、AaaAI" />
            </a-form-item>
          </a-col>
          <a-col :span="16">
            <a-form-item label="版权信息">
              <a-input v-model="brandForm.copyright" placeholder="Copyright © 2026 YourBrand. All rights reserved." />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="浅色 Logo 地址">
              <a-input v-model="brandForm.logoUrl" placeholder="/uploads/branding/logo.svg 或 https://...">
                <template #append>
                  <a-upload :show-file-list="false" accept="image/svg+xml,image/png,image/jpeg,image/gif,image/webp,image/x-icon" :custom-request="uploadBranding('logoUrl')">
                    <template #upload-button>
                      <a-button :loading="uploadingBrand.logoUrl"><template #icon><icon-upload /></template>上传</a-button>
                    </template>
                  </a-upload>
                </template>
              </a-input>
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="深色 Logo 地址">
              <a-input v-model="brandForm.logoDarkUrl" placeholder="用于深色底边栏">
                <template #append>
                  <a-upload :show-file-list="false" accept="image/svg+xml,image/png,image/jpeg,image/gif,image/webp,image/x-icon" :custom-request="uploadBranding('logoDarkUrl')">
                    <template #upload-button>
                      <a-button :loading="uploadingBrand.logoDarkUrl"><template #icon><icon-upload /></template>上传</a-button>
                    </template>
                  </a-upload>
                </template>
              </a-input>
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="侧边栏收缩 Logo 地址">
              <a-input v-model="brandForm.logoCollapsedUrl" placeholder="建议使用方形图标">
                <template #append>
                  <a-upload :show-file-list="false" accept="image/svg+xml,image/png,image/jpeg,image/gif,image/webp,image/x-icon" :custom-request="uploadBranding('logoCollapsedUrl')">
                    <template #upload-button>
                      <a-button :loading="uploadingBrand.logoCollapsedUrl"><template #icon><icon-upload /></template>上传</a-button>
                    </template>
                  </a-upload>
                </template>
              </a-input>
            </a-form-item>
          </a-col>
        </a-row>
        <div class="brand-preview-grid">
          <div class="brand-preview">
            <span>浅色 Logo</span>
            <img v-if="brandForm.logoUrl" :src="brandForm.logoUrl" :alt="brandForm.name" />
            <strong v-else>{{ brandForm.name }}</strong>
          </div>
          <div class="brand-preview brand-preview-dark">
            <span>深色 Logo</span>
            <img v-if="brandForm.logoDarkUrl" :src="brandForm.logoDarkUrl" :alt="brandForm.name" />
            <strong v-else>{{ brandForm.name }}</strong>
          </div>
          <div class="brand-preview">
            <span>收缩 Logo</span>
            <img v-if="brandForm.logoCollapsedUrl" class="brand-preview-icon" :src="brandForm.logoCollapsedUrl" :alt="brandForm.name" />
            <strong v-else>{{ brandForm.name.slice(0, 1) }}</strong>
          </div>
        </div>
        <a-button type="primary" :loading="saving" @click="saveBrand">
          <template #icon><icon-save /></template>
          保存品牌配置
        </a-button>
      </a-form>
    </a-card>

    <a-card title="企业展示页配置" :bordered="false">
      <a-form layout="vertical" :model="portalForm">
        <a-tabs>
          <a-tab-pane key="hero" title="首屏文案">
            <a-row :gutter="16">
              <a-col :span="12">
                <a-form-item label="主标题">
                  <a-input v-model="portalForm.title" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="副标题">
                  <a-input v-model="portalForm.subtitle" />
                </a-form-item>
              </a-col>
              <a-col :span="8">
                <a-form-item label="顶部标签">
                  <a-input v-model="portalForm.tag" placeholder="AI API Gateway" />
                </a-form-item>
              </a-col>
              <a-col :span="8">
                <a-form-item label="主按钮文案">
                  <a-input v-model="portalForm.primaryText" />
                </a-form-item>
              </a-col>
              <a-col :span="8">
                <a-form-item label="主按钮跳转">
                  <a-input v-model="portalForm.primaryTarget" placeholder="留空则自动进入控制台或登录页" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="次按钮文案">
                  <a-input v-model="portalForm.secondaryText" />
                </a-form-item>
              </a-col>
              <a-col :span="12">
                <a-form-item label="次按钮跳转">
                  <a-input v-model="portalForm.secondaryTarget" placeholder="/auth/register、#features 或 https://example.com" />
                </a-form-item>
              </a-col>
            </a-row>
          </a-tab-pane>

          <a-tab-pane key="nav" title="导航菜单">
            <div class="editor-toolbar">
              <a-alert type="info" show-icon>菜单支持拖动排序；跳转地址可填写站内路径、外部网址或首页区块锚点。</a-alert>
              <a-button type="primary" @click="addNavItem">
                <template #icon><icon-plus /></template>
                新增菜单
              </a-button>
            </div>
            <div class="editor-list">
              <div
                v-for="(item, index) in portalNav"
                :key="`${item.label}-${index}`"
                class="editor-row"
                draggable="true"
                @dragstart="startDrag('nav', index)"
                @dragover.prevent
                @drop="dropItem('nav', portalNav, index)"
              >
                <div class="drag-handle"><icon-drag-dot-vertical /></div>
                <a-row :gutter="12" class="editor-row-fields">
                  <a-col :span="6"><a-form-item label="名称"><a-input v-model="item.label" /></a-form-item></a-col>
                  <a-col :span="10"><a-form-item label="跳转"><a-input v-model="item.target" /></a-form-item></a-col>
                  <a-col :span="5"><a-form-item label="图标"><a-input v-model="item.icon" placeholder="home" /></a-form-item></a-col>
                  <a-col :span="3" class="editor-actions">
                    <a-button type="text" :disabled="index === 0" @click="moveItem(portalNav, index, index - 1)"><template #icon><icon-up /></template></a-button>
                    <a-button type="text" :disabled="index === portalNav.length - 1" @click="moveItem(portalNav, index, index + 1)"><template #icon><icon-down /></template></a-button>
                    <a-button type="text" status="danger" @click="removeAt(portalNav, index)"><template #icon><icon-delete /></template></a-button>
                  </a-col>
                </a-row>
              </div>
            </div>
          </a-tab-pane>

          <a-tab-pane key="metrics" title="数据指标">
            <div class="editor-toolbar">
              <a-alert type="info" show-icon>指标会显示在首屏下方，例如“OpenAI / 兼容接口”。</a-alert>
              <a-button type="primary" @click="addMetric">
                <template #icon><icon-plus /></template>
                新增指标
              </a-button>
            </div>
            <div class="editor-list">
              <div
                v-for="(item, index) in portalMetrics"
                :key="`${item.label}-${index}`"
                class="editor-row"
                draggable="true"
                @dragstart="startDrag('metric', index)"
                @dragover.prevent
                @drop="dropItem('metric', portalMetrics, index)"
              >
                <div class="drag-handle"><icon-drag-dot-vertical /></div>
                <a-row :gutter="12" class="editor-row-fields">
                  <a-col :span="8"><a-form-item label="指标值"><a-input v-model="item.value" /></a-form-item></a-col>
                  <a-col :span="10"><a-form-item label="指标名称"><a-input v-model="item.label" /></a-form-item></a-col>
                  <a-col :span="6" class="editor-actions">
                    <a-button type="text" :disabled="index === 0" @click="moveItem(portalMetrics, index, index - 1)"><template #icon><icon-up /></template></a-button>
                    <a-button type="text" :disabled="index === portalMetrics.length - 1" @click="moveItem(portalMetrics, index, index + 1)"><template #icon><icon-down /></template></a-button>
                    <a-button type="text" status="danger" @click="removeAt(portalMetrics, index)"><template #icon><icon-delete /></template></a-button>
                  </a-col>
                </a-row>
              </div>
            </div>
          </a-tab-pane>

          <a-tab-pane key="sections" title="主页区块">
            <div class="editor-toolbar">
              <a-alert type="info" show-icon>区块和区块内条目都支持拖动排序，图标填写 Arco 图标名，例如 apps、gift、lock。</a-alert>
              <a-space>
                <a-dropdown @select="(value) => addSection(value as SectionType)">
                  <a-button type="primary">
                    <template #icon><icon-plus /></template>
                    新增区块
                  </a-button>
                  <template #content>
                    <a-doption v-for="option in sectionTypeOptions" :key="option.value" :value="option.value">{{ option.label }}</a-doption>
                  </template>
                </a-dropdown>
              </a-space>
            </div>
            <div class="section-editor-list">
              <section
                v-for="(section, sectionIndex) in portalSections"
                :key="section.id"
                class="section-editor"
                draggable="true"
                @dragstart="startDrag('section', sectionIndex)"
                @dragover.prevent
                @drop="dropItem('section', portalSections, sectionIndex)"
              >
                <div class="section-editor-head">
                  <div class="section-title-line">
                    <icon-drag-dot-vertical />
                    <strong>{{ section.title || '未命名区块' }}</strong>
                    <a-tag>{{ sectionTypeOptions.find((item) => item.value === section.type)?.label || section.type }}</a-tag>
                  </div>
                  <a-space>
                    <a-button type="text" :disabled="sectionIndex === 0" @click="moveItem(portalSections, sectionIndex, sectionIndex - 1)"><template #icon><icon-up /></template></a-button>
                    <a-button type="text" :disabled="sectionIndex === portalSections.length - 1" @click="moveItem(portalSections, sectionIndex, sectionIndex + 1)"><template #icon><icon-down /></template></a-button>
                    <a-button type="text" status="danger" @click="removeAt(portalSections, sectionIndex)"><template #icon><icon-delete /></template></a-button>
                  </a-space>
                </div>

                <a-row :gutter="12">
                  <a-col :span="5"><a-form-item label="区块 ID"><a-input v-model="section.id" /></a-form-item></a-col>
                  <a-col :span="5"><a-form-item label="类型"><a-select v-model="section.type" :options="sectionTypeOptions" /></a-form-item></a-col>
                  <a-col :span="5"><a-form-item label="眉标"><a-input v-model="section.eyebrow" /></a-form-item></a-col>
                  <a-col :span="5"><a-form-item label="图标"><a-input v-model="section.icon" /></a-form-item></a-col>
                  <a-col :span="24"><a-form-item label="标题"><a-input v-model="section.title" /></a-form-item></a-col>
                  <a-col :span="24"><a-form-item label="描述"><a-textarea v-model="section.description" :auto-size="{ minRows: 2, maxRows: 4 }" /></a-form-item></a-col>
                </a-row>

                <div v-if="section.type !== 'cta'" class="nested-editor">
                  <div class="nested-editor-head">
                    <strong>区块条目</strong>
                    <a-button size="small" @click="section.items.push(createSectionItem())">
                      <template #icon><icon-plus /></template>
                      新增条目
                    </a-button>
                  </div>
                  <div
                    v-for="(item, itemIndex) in section.items"
                    :key="`${section.id}-${itemIndex}`"
                    class="nested-row"
                    draggable="true"
                    @dragstart="startDrag(`section-items-${section.id}`, itemIndex)"
                    @dragover.prevent
                    @drop="dropItem(`section-items-${section.id}`, section.items, itemIndex)"
                  >
                    <div class="drag-handle"><icon-drag-dot-vertical /></div>
                    <a-row :gutter="12" class="editor-row-fields">
                      <a-col :span="5"><a-form-item label="标题"><a-input v-model="item.title" /></a-form-item></a-col>
                      <a-col :span="4"><a-form-item label="图标"><a-input v-model="item.icon" /></a-form-item></a-col>
                      <a-col :span="15"><a-form-item label="描述"><a-input v-model="item.description" /></a-form-item></a-col>
                      <a-col :span="24" class="editor-actions editor-actions-inline">
                        <a-button type="text" :disabled="itemIndex === 0" @click="moveItem(section.items, itemIndex, itemIndex - 1)"><template #icon><icon-up /></template></a-button>
                        <a-button type="text" :disabled="itemIndex === section.items.length - 1" @click="moveItem(section.items, itemIndex, itemIndex + 1)"><template #icon><icon-down /></template></a-button>
                        <a-button type="text" status="danger" @click="removeAt(section.items, itemIndex)"><template #icon><icon-delete /></template></a-button>
                      </a-col>
                    </a-row>
                  </div>
                </div>

                <div v-else class="nested-editor">
                  <div class="nested-editor-head">
                    <strong>行动按钮</strong>
                    <a-button size="small" @click="section.actions.push(createAction())">
                      <template #icon><icon-plus /></template>
                      新增按钮
                    </a-button>
                  </div>
                  <div
                    v-for="(action, actionIndex) in section.actions"
                    :key="`${section.id}-action-${actionIndex}`"
                    class="nested-row"
                    draggable="true"
                    @dragstart="startDrag(`section-actions-${section.id}`, actionIndex)"
                    @dragover.prevent
                    @drop="dropItem(`section-actions-${section.id}`, section.actions, actionIndex)"
                  >
                    <div class="drag-handle"><icon-drag-dot-vertical /></div>
                    <a-row :gutter="12" class="editor-row-fields">
                      <a-col :span="5"><a-form-item label="文案"><a-input v-model="action.label" /></a-form-item></a-col>
                      <a-col :span="8"><a-form-item label="跳转"><a-input v-model="action.target" /></a-form-item></a-col>
                      <a-col :span="4"><a-form-item label="图标"><a-input v-model="action.icon" /></a-form-item></a-col>
                      <a-col :span="4"><a-form-item label="样式"><a-select v-model="action.type" :options="actionTypeOptions" /></a-form-item></a-col>
                      <a-col :span="3" class="editor-actions">
                        <a-button type="text" :disabled="actionIndex === 0" @click="moveItem(section.actions, actionIndex, actionIndex - 1)"><template #icon><icon-up /></template></a-button>
                        <a-button type="text" :disabled="actionIndex === section.actions.length - 1" @click="moveItem(section.actions, actionIndex, actionIndex + 1)"><template #icon><icon-down /></template></a-button>
                        <a-button type="text" status="danger" @click="removeAt(section.actions, actionIndex)"><template #icon><icon-delete /></template></a-button>
                      </a-col>
                    </a-row>
                  </div>
                </div>
              </section>
            </div>
          </a-tab-pane>

          <a-tab-pane key="footer" title="底边栏">
            <a-row :gutter="16">
              <a-col :span="12"><a-form-item label="版权信息"><a-input v-model="portalFooter.copyright" @change="brandForm.copyright = portalFooter.copyright" /></a-form-item></a-col>
              <a-col :span="12"><a-form-item label="二维码标题"><a-input v-model="portalFooter.qrcodeTitle" /></a-form-item></a-col>
              <a-col :span="12"><a-form-item label="二维码描述"><a-input v-model="portalFooter.qrcodeDescription" /></a-form-item></a-col>
              <a-col :span="12"><a-form-item label="二维码图片地址"><a-input v-model="portalFooter.qrcodeImage" /></a-form-item></a-col>
            </a-row>

            <div class="footer-link-grid">
              <div class="footer-link-editor">
                <div class="nested-editor-head">
                  <strong>底部链接</strong>
                  <a-button size="small" @click="addFooterLink(portalFooter.links)">
                    <template #icon><icon-plus /></template>
                    新增链接
                  </a-button>
                </div>
                <div
                  v-for="(item, index) in portalFooter.links"
                  :key="`footer-link-${index}`"
                  class="footer-link-row"
                  draggable="true"
                  @dragstart="startDrag('footer-links', index)"
                  @dragover.prevent
                  @drop="dropItem('footer-links', portalFooter.links, index)"
                >
                  <icon-drag-dot-vertical />
                  <a-input v-model="item.label" placeholder="名称" />
                  <a-input v-model="item.target" placeholder="跳转地址" />
                  <a-button type="text" status="danger" @click="removeAt(portalFooter.links, index)"><template #icon><icon-delete /></template></a-button>
                </div>
              </div>
              <div class="footer-link-editor">
                <div class="nested-editor-head">
                  <strong>友情链接</strong>
                  <a-button size="small" @click="addFooterLink(portalFooter.friendLinks)">
                    <template #icon><icon-plus /></template>
                    新增友链
                  </a-button>
                </div>
                <div
                  v-for="(item, index) in portalFooter.friendLinks"
                  :key="`friend-link-${index}`"
                  class="footer-link-row"
                  draggable="true"
                  @dragstart="startDrag('friend-links', index)"
                  @dragover.prevent
                  @drop="dropItem('friend-links', portalFooter.friendLinks, index)"
                >
                  <icon-drag-dot-vertical />
                  <a-input v-model="item.label" placeholder="名称" />
                  <a-input v-model="item.target" placeholder="跳转地址" />
                  <a-button type="text" status="danger" @click="removeAt(portalFooter.friendLinks, index)"><template #icon><icon-delete /></template></a-button>
                </div>
              </div>
            </div>
          </a-tab-pane>
        </a-tabs>

        <div class="portal-save-bar">
          <a-alert class="settings-help" type="info" show-icon>
            图标填写 Arco 图标名，例如 apps、gift、lock；跳转支持 /billing、https://example.com、#features。
          </a-alert>
          <a-button type="primary" :loading="saving" @click="savePortal">
            <template #icon><icon-save /></template>
            保存展示页配置
          </a-button>
        </div>
      </a-form>
    </a-card>

    <a-card title="支付渠道配置" :bordered="false">
      <a-form layout="vertical" :model="paymentForm">
        <a-row :gutter="16">
          <a-col :span="8"><a-form-item label="公网后端地址"><a-input v-model="paymentForm.public_base_url" /></a-form-item></a-col>
          <a-col :span="8"><a-form-item label="支付返回前端地址"><a-input v-model="paymentForm.return_frontend_url" /></a-form-item></a-col>
          <a-col :span="8"><a-form-item label="美元兑人民币汇率"><a-input v-model="paymentForm.usd_cny_rate" /></a-form-item></a-col>
        </a-row>

        <a-tabs>
          <a-tab-pane key="manual" title="人工处理">
            <a-form-item label="启用人工处理"><a-switch v-model="paymentForm.manual_enabled" /></a-form-item>
          </a-tab-pane>
          <a-tab-pane key="epay" title="易支付">
            <a-row :gutter="16">
              <a-col :span="6"><a-form-item label="启用"><a-switch v-model="paymentForm.epay_enabled" /></a-form-item></a-col>
              <a-col :span="18"><a-form-item label="网关地址"><a-input v-model="paymentForm.epay_gateway" /></a-form-item></a-col>
              <a-col :span="12"><a-form-item label="商户 PID"><a-input v-model="paymentForm.epay_pid" /></a-form-item></a-col>
              <a-col :span="12"><a-form-item label="商户 Key"><a-input-password v-model="paymentForm.epay_key" /></a-form-item></a-col>
            </a-row>
          </a-tab-pane>
          <a-tab-pane key="codepay" title="码支付">
            <a-row :gutter="16">
              <a-col :span="6"><a-form-item label="启用"><a-switch v-model="paymentForm.codepay_enabled" /></a-form-item></a-col>
              <a-col :span="18"><a-form-item label="网关地址"><a-input v-model="paymentForm.codepay_gateway" /></a-form-item></a-col>
              <a-col :span="12"><a-form-item label="商户 ID"><a-input v-model="paymentForm.codepay_id" /></a-form-item></a-col>
              <a-col :span="12"><a-form-item label="通信 Key"><a-input-password v-model="paymentForm.codepay_key" /></a-form-item></a-col>
            </a-row>
          </a-tab-pane>
          <a-tab-pane key="xunhupay" title="虎皮椒">
            <a-row :gutter="16">
              <a-col :span="6"><a-form-item label="启用"><a-switch v-model="paymentForm.xunhupay_enabled" /></a-form-item></a-col>
              <a-col :span="18"><a-form-item label="网关地址"><a-input v-model="paymentForm.xunhupay_gateway" /></a-form-item></a-col>
              <a-col :span="12"><a-form-item label="AppID"><a-input v-model="paymentForm.xunhupay_appid" /></a-form-item></a-col>
              <a-col :span="12"><a-form-item label="AppSecret"><a-input-password v-model="paymentForm.xunhupay_appsecret" /></a-form-item></a-col>
            </a-row>
          </a-tab-pane>
          <a-tab-pane key="alipay" title="支付宝当面付">
            <a-row :gutter="16">
              <a-col :span="6"><a-form-item label="启用"><a-switch v-model="paymentForm.alipay_enabled" /></a-form-item></a-col>
              <a-col :span="18"><a-form-item label="网关地址"><a-input v-model="paymentForm.alipay_gateway" /></a-form-item></a-col>
              <a-col :span="12"><a-form-item label="AppID"><a-input v-model="paymentForm.alipay_app_id" /></a-form-item></a-col>
              <a-col :span="12"><a-form-item label="支付宝公钥"><a-textarea v-model="paymentForm.alipay_public_key" :auto-size="{ minRows: 2, maxRows: 6 }" /></a-form-item></a-col>
              <a-col :span="24"><a-form-item label="应用私钥"><a-textarea v-model="paymentForm.alipay_private_key" class="mono" :auto-size="{ minRows: 4, maxRows: 10 }" /></a-form-item></a-col>
            </a-row>
          </a-tab-pane>
          <a-tab-pane key="wechat" title="微信官方支付">
            <a-row :gutter="16">
              <a-col :span="6"><a-form-item label="启用"><a-switch v-model="paymentForm.wechat_enabled" /></a-form-item></a-col>
              <a-col :span="18"><a-form-item label="网关地址"><a-input v-model="paymentForm.wechat_gateway" /></a-form-item></a-col>
              <a-col :span="8"><a-form-item label="AppID"><a-input v-model="paymentForm.wechat_appid" /></a-form-item></a-col>
              <a-col :span="8"><a-form-item label="商户号"><a-input v-model="paymentForm.wechat_mchid" /></a-form-item></a-col>
              <a-col :span="8"><a-form-item label="证书序列号"><a-input v-model="paymentForm.wechat_serial_no" /></a-form-item></a-col>
              <a-col :span="12"><a-form-item label="APIv3 Key"><a-input-password v-model="paymentForm.wechat_api_v3_key" /></a-form-item></a-col>
              <a-col :span="24"><a-form-item label="商户私钥"><a-textarea v-model="paymentForm.wechat_private_key" class="mono" :auto-size="{ minRows: 4, maxRows: 10 }" /></a-form-item></a-col>
            </a-row>
          </a-tab-pane>
        </a-tabs>
        <a-button type="primary" :loading="saving" @click="savePayments">
          <template #icon><icon-save /></template>
          保存支付配置
        </a-button>
      </a-form>
    </a-card>

    <a-card title="SMTP 邮件配置" :bordered="false">
      <a-form layout="vertical" :model="smtpForm">
        <a-row :gutter="16">
          <a-col :span="8">
            <a-form-item label="启用 SMTP">
              <a-switch v-model="smtpForm.enabled" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="加密方式">
              <a-select v-model="smtpForm.encryption" :options="smtpEncryptionOptions" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="端口">
              <a-input-number v-model="smtpForm.port" :min="1" :max="65535" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="SMTP 主机">
              <a-input v-model="smtpForm.host" placeholder="smtp.example.com" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="用户名">
              <a-input v-model="smtpForm.username" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="密码或授权码">
              <a-input-password v-model="smtpForm.password" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="发件邮箱">
              <a-input v-model="smtpForm.from_email" placeholder="no-reply@example.com" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="发件名称">
              <a-input v-model="smtpForm.from_name" />
            </a-form-item>
          </a-col>
          <a-col :span="12" class="smtp-actions">
            <a-button type="primary" :loading="saving" @click="saveSMTP">
              <template #icon><icon-save /></template>
              保存 SMTP 配置
            </a-button>
          </a-col>
        </a-row>
      </a-form>
    </a-card>

    <a-card :bordered="false">
      <a-table :data="rows" :loading="loading" row-key="id">
        <template #columns>
          <a-table-column title="键名" data-index="key" :width="240">
            <template #cell="{ record }"><span class="mono">{{ record.key }}</span></template>
          </a-table-column>
          <a-table-column title="值" data-index="value" />
          <a-table-column title="公开" data-index="is_public" :width="110">
            <template #cell="{ record }"><a-switch :model-value="record.is_public" disabled /></template>
          </a-table-column>
          <a-table-column title="更新时间" data-index="updated_at" :width="190" />
          <a-table-column title="操作" :width="100">
            <template #cell="{ record }">
              <a-button type="text" @click="openEdit(record)">
                <template #icon><icon-edit /></template>
              </a-button>
            </template>
          </a-table-column>
        </template>
      </a-table>
    </a-card>

    <a-modal v-model:visible="visible" title="保存设置" :confirm-loading="saving" @ok="save">
      <a-form layout="vertical" :model="form">
        <a-form-item label="键名" required>
          <a-input v-model="form.key" class="mono" />
        </a-form-item>
        <a-form-item label="值">
          <a-textarea v-model="form.value" :auto-size="{ minRows: 3, maxRows: 8 }" />
        </a-form-item>
        <a-form-item label="公开">
          <a-switch v-model="form.is_public" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<style scoped>
.smtp-actions {
  display: flex;
  align-items: flex-end;
  justify-content: flex-end;
  padding-bottom: 20px;
}

.brand-preview-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 16px;
}

.brand-preview {
  min-height: 96px;
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 10px;
  padding: 16px;
  border: 1px solid var(--bitapi-border);
  border-radius: 8px;
  background: #fff;
}

.brand-preview-dark {
  color: #fff;
  background: #111827;
}

.brand-preview span {
  color: var(--bitapi-muted);
  font-size: 12px;
}

.brand-preview-dark span {
  color: rgba(255, 255, 255, 0.68);
}

.brand-preview img {
  width: 160px;
  height: 34px;
  object-fit: contain;
}

.brand-preview .brand-preview-icon {
  width: 34px;
  height: 34px;
}

.editor-toolbar,
.portal-save-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}

.editor-list,
.section-editor-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.editor-row,
.nested-row,
.section-editor {
  border: 1px solid var(--bitapi-border);
  border-radius: 8px;
  background: #fff;
}

.editor-row,
.nested-row {
  display: flex;
  align-items: stretch;
  gap: 10px;
  padding: 12px;
}

.editor-row:hover,
.nested-row:hover,
.section-editor:hover {
  border-color: #bedaff;
}

.drag-handle,
.footer-link-row > svg,
.section-title-line > svg {
  flex: 0 0 auto;
  margin-top: 30px;
  color: var(--bitapi-muted);
  cursor: grab;
}

.editor-row-fields {
  flex: 1;
}

.editor-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 2px;
  padding-top: 25px;
}

.editor-actions-inline {
  justify-content: flex-start;
  padding-top: 0;
}

.section-editor {
  padding: 16px;
}

.section-editor-head,
.nested-editor-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 14px;
}

.section-title-line {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.section-title-line > svg {
  margin-top: 0;
}

.nested-editor {
  padding: 14px;
  border-radius: 8px;
  background: #f7f9fc;
}

.footer-link-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}

.footer-link-editor {
  padding: 14px;
  border: 1px solid var(--bitapi-border);
  border-radius: 8px;
}

.footer-link-row {
  display: grid;
  grid-template-columns: auto minmax(0, 0.8fr) minmax(0, 1.2fr) auto;
  gap: 8px;
  align-items: center;
  margin-top: 10px;
}

.footer-link-row > svg {
  margin-top: 0;
}

@media (max-width: 900px) {
  .editor-toolbar,
  .portal-save-bar {
    align-items: stretch;
    flex-direction: column;
  }

  .footer-link-grid {
    grid-template-columns: 1fr;
  }
}
</style>
