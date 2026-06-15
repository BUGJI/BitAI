export function roleLabel(role?: string) {
  const labels: Record<string, string> = {
    owner: '平台所有者',
    admin: '管理员',
    operator: '运营人员',
    user: '普通用户',
    auditor: '审计员'
  };
  return labels[role || ''] || '未知角色';
}

export function statusLabel(status?: string) {
  const labels: Record<string, string> = {
    active: '正常',
    disabled: '已禁用',
    suspended: '已暂停',
    pending: '待处理',
    paid: '已支付',
    cancelled: '已取消',
    rejected: '已驳回',
    expired: '已过期'
  };
  return labels[status || ''] || '未知状态';
}

export function platformLabel(platform?: string) {
  const labels: Record<string, string> = {
    openai: '兼容接口',
    anthropic: 'Anthropic',
    gemini: 'Gemini',
    custom: '自定义'
  };
  return labels[platform || ''] || '未知平台';
}

export function modeLabel(mode?: string) {
  const labels: Record<string, string> = {
    balance: '余额扣费',
    subscription: '订阅额度'
  };
  return labels[mode || ''] || '未知模式';
}

export function providerLabel(provider?: string) {
  const labels: Record<string, string> = {
    manual: '人工处理'
  };
  return labels[provider || ''] || '未知方式';
}

export const roleOptions = [
  { label: '平台所有者', value: 'owner' },
  { label: '管理员', value: 'admin' },
  { label: '运营人员', value: 'operator' },
  { label: '普通用户', value: 'user' },
  { label: '审计员', value: 'auditor' }
];

export const statusOptions = [
  { label: '正常', value: 'active' },
  { label: '已禁用', value: 'disabled' },
  { label: '已暂停', value: 'suspended' }
];

export const platformOptions = [
  { label: '兼容接口', value: 'openai' },
  { label: 'Anthropic', value: 'anthropic' },
  { label: 'Gemini', value: 'gemini' },
  { label: '自定义', value: 'custom' }
];

export const modeOptions = [
  { label: '余额扣费', value: 'balance' },
  { label: '订阅额度', value: 'subscription' }
];
