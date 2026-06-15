<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import { Message } from '@arco-design/web-vue';
import { adminApi } from '../../api/admin';
import type { Setting } from '../../types';

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

const smtpEncryptionOptions = [
  { label: 'STARTTLS', value: 'starttls' },
  { label: 'SSL/TLS', value: 'tls' },
  { label: '不加密', value: 'none' }
];

function settingValue(key: string, fallback = '') {
  return rows.value.find((item) => item.key === key)?.value ?? fallback;
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

async function load() {
  loading.value = true;
  try {
    rows.value = await adminApi.settings();
    syncSMTPForm();
  } finally {
    loading.value = false;
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
</style>
