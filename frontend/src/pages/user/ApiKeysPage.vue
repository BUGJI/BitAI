<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { Message, Modal } from '@arco-design/web-vue';
import { userApi } from '../../api/user';
import type { APIKey } from '../../types';
import { statusLabel } from '../../utils/display';

const loading = ref(false);
const creating = ref(false);
const keys = ref<APIKey[]>([]);
const visible = ref(false);
const newKey = ref('');
const gatewayBaseURL = computed(() => `${window.location.protocol}//${window.location.hostname}:8080`);
const openAIBaseURL = computed(() => `${gatewayBaseURL.value}/v1`);
const responsesURL = computed(() => `${gatewayBaseURL.value}/responses`);
const form = reactive({
  name: '',
  group_id: undefined as number | undefined,
  quota_limit_micros: 0
});

async function load() {
  loading.value = true;
  try {
    keys.value = await userApi.keys();
  } finally {
    loading.value = false;
  }
}

async function createKey() {
  creating.value = true;
  try {
    const result = await userApi.createKey(form);
    newKey.value = result.key;
    visible.value = false;
    form.name = '';
    form.group_id = undefined;
    form.quota_limit_micros = 0;
    await load();
  } catch (error: any) {
    Message.error(error?.response?.data?.message || '创建密钥失败');
  } finally {
    creating.value = false;
  }
}

async function copyText(text: string, success = '已复制') {
  if (!text) {
    Message.warning('该密钥创建于旧版本，未保存完整密钥，请重新创建后复制');
    return;
  }
  await navigator.clipboard.writeText(text);
  Message.success(success);
}

function keyValue(record: APIKey) {
  return record.key || '';
}

function maskKey(record: APIKey) {
  const key = keyValue(record);
  if (!key) return `${record.key_prefix}（旧密钥仅保留前缀）`;
  if (key.length <= 18) return key;
  return `${key.slice(0, 10)}...${key.slice(-8)}`;
}

function shellValue(value: string) {
  return value.replace(/'/g, `'\\''`);
}

function shellScript(record: APIKey) {
  const key = keyValue(record);
  if (!key) return '';
  return [
    '#!/usr/bin/env sh',
    'set -eu',
    '',
    'CODEX_HOME="${CODEX_HOME:-$HOME/.codex}"',
    'mkdir -p "$CODEX_HOME"',
    '[ -f "$CODEX_HOME/config.toml" ] && cp "$CODEX_HOME/config.toml" "$CODEX_HOME/config.toml.bak"',
    '[ -f "$CODEX_HOME/auth.json" ] && cp "$CODEX_HOME/auth.json" "$CODEX_HOME/auth.json.bak"',
    '',
    'cat > "$CODEX_HOME/config.toml" <<EOF',
    'model_provider = "bitapi"',
    'model = "gpt-4.1-mini"',
    '',
    '[model_providers.bitapi]',
    'name = "BitAPI"',
    'wire_api = "responses"',
    'requires_openai_auth = true',
    `base_url = "${gatewayBaseURL.value}"`,
    'EOF',
    '',
    'cat > "$CODEX_HOME/auth.json" <<EOF',
    '{',
    `  "OPENAI_API_KEY": "${shellValue(key)}"`,
    '}',
    'EOF',
    '',
    `echo 'Codex 已配置到 BitAPI：${shellValue(gatewayBaseURL.value)}'`
  ].join('\n');
}

function batchScript(record: APIKey) {
  const key = keyValue(record);
  if (!key) return '';
  return [
    '@echo off',
    'setlocal',
    'set "CODEX_HOME=%USERPROFILE%\\.codex"',
    'if not exist "%CODEX_HOME%" mkdir "%CODEX_HOME%"',
    'if exist "%CODEX_HOME%\\config.toml" copy /Y "%CODEX_HOME%\\config.toml" "%CODEX_HOME%\\config.toml.bak" >nul',
    'if exist "%CODEX_HOME%\\auth.json" copy /Y "%CODEX_HOME%\\auth.json" "%CODEX_HOME%\\auth.json.bak" >nul',
    '',
    '> "%CODEX_HOME%\\config.toml" echo model_provider = "bitapi"',
    '>> "%CODEX_HOME%\\config.toml" echo model = "gpt-4.1-mini"',
    '>> "%CODEX_HOME%\\config.toml" echo.',
    '>> "%CODEX_HOME%\\config.toml" echo [model_providers.bitapi]',
    '>> "%CODEX_HOME%\\config.toml" echo name = "BitAPI"',
    '>> "%CODEX_HOME%\\config.toml" echo wire_api = "responses"',
    '>> "%CODEX_HOME%\\config.toml" echo requires_openai_auth = true',
    `>> "%CODEX_HOME%\\config.toml" echo base_url = "${gatewayBaseURL.value}"`,
    '',
    '> "%CODEX_HOME%\\auth.json" echo {',
    `>> "%CODEX_HOME%\\auth.json" echo   "OPENAI_API_KEY": "${key}"`,
    '>> "%CODEX_HOME%\\auth.json" echo }',
    '',
    `echo Codex 已配置到 BitAPI：${gatewayBaseURL.value}`,
    'endlocal'
  ].join('\r\n');
}

function confirmDelete(record: APIKey) {
  Modal.warning({
    title: '删除调用密钥',
    content: `确认删除 ${record.name}？此操作无法撤销。`,
    hideCancel: false,
    onOk: async () => {
      await userApi.deleteKey(record.id);
      Message.success('已删除');
      await load();
    }
  });
}

onMounted(load);
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h1 class="page-title">调用密钥</h1>
        <p class="page-subtitle">创建用于兼容模型接口网关访问的受控密钥。</p>
      </div>
      <a-button type="primary" @click="visible = true">
        <template #icon><icon-plus /></template>
        创建密钥
      </a-button>
    </div>

    <a-alert v-if="newKey" type="success" show-icon>
      <template #title>请立即复制该密钥</template>
      <a-space direction="vertical" fill>
        <div class="mono new-key">{{ newKey }}</div>
        <a-space>
          <a-button size="small" @click="copyText(newKey, '密钥已复制')">
            <template #icon><icon-copy /></template>
            复制密钥
          </a-button>
          <a-button size="small" @click="copyText(openAIBaseURL, '对接地址已复制')">
            <template #icon><icon-link /></template>
            复制 OpenAI 地址
          </a-button>
          <a-button size="small" @click="copyText(gatewayBaseURL, 'Codex 地址已复制')">
            <template #icon><icon-link /></template>
            复制 Codex 地址
          </a-button>
        </a-space>
      </a-space>
    </a-alert>

    <a-alert type="info" show-icon>
      <template #title>模型接口对接地址</template>
      <div class="gateway-info">
        <div>Codex base_url：<span class="mono">{{ gatewayBaseURL }}</span></div>
        <div>Responses 实际入口：<span class="mono">{{ responsesURL }}</span></div>
        <div>OpenAI 兼容 base_url：<span class="mono">{{ openAIBaseURL }}</span></div>
      </div>
    </a-alert>

    <a-card :bordered="false">
      <a-table :data="keys" :loading="loading" row-key="id" :scroll="{ x: 1080 }">
        <template #columns>
          <a-table-column title="名称" data-index="name" :width="180" />
          <a-table-column title="调用密钥" data-index="key" :width="300">
            <template #cell="{ record }"><span class="mono">{{ maskKey(record) }}</span></template>
          </a-table-column>
          <a-table-column title="状态" data-index="status" :width="110">
            <template #cell="{ record }"><a-tag color="green">{{ statusLabel(record.status) }}</a-tag></template>
          </a-table-column>
          <a-table-column title="已用额度" :width="150">
            <template #cell="{ record }">{{ (record.quota_used_micros / 1_000_000).toFixed(4) }} 美元</template>
          </a-table-column>
          <a-table-column title="创建时间" data-index="created_at" :width="190" />
          <a-table-column title="操作" :width="320">
            <template #cell="{ record }">
              <a-space>
                <a-button type="text" @click="copyText(keyValue(record), '密钥已复制')">
                  <template #icon><icon-copy /></template>
                </a-button>
                <a-button type="text" @click="copyText(shellScript(record), 'Codex sh 配置脚本已复制')">
                  <template #icon><icon-code /></template>
                  Codex sh
                </a-button>
                <a-button type="text" @click="copyText(batchScript(record), 'Codex bat 配置脚本已复制')">
                  <template #icon><icon-code /></template>
                  Codex bat
                </a-button>
                <a-button status="danger" type="text" @click="confirmDelete(record)">
                  <template #icon><icon-delete /></template>
                </a-button>
              </a-space>
            </template>
          </a-table-column>
        </template>
      </a-table>
    </a-card>

    <a-modal v-model:visible="visible" title="创建调用密钥" :confirm-loading="creating" @ok="createKey">
      <a-form layout="vertical" :model="form">
        <a-form-item label="名称" required>
          <a-input v-model="form.name" placeholder="生产环境密钥" />
        </a-form-item>
        <a-form-item label="分组编号">
          <a-input-number v-model="form.group_id" placeholder="留空使用默认分组" />
        </a-form-item>
        <a-form-item label="额度上限（微美元）">
          <a-input-number v-model="form.quota_limit_micros" :min="0" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<style scoped>
.new-key {
  margin-top: 8px;
  word-break: break-all;
}

.gateway-info {
  display: grid;
  gap: 6px;
}
</style>
