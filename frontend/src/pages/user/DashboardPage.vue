<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { userApi } from '../../api/user';
import type { APIKey, UsageLog } from '../../types';
import { useAuthStore } from '../../stores/auth';

const auth = useAuthStore();
const keys = ref<APIKey[]>([]);
const usage = ref<UsageLog[]>([]);
const loading = ref(false);

const balance = computed(() => (auth.user?.balance_micros || 0) / 1_000_000);
const totalCharged = computed(() => usage.value.reduce((sum, row) => sum + row.charged_micros, 0) / 1_000_000);
const totalTokens = computed(() => usage.value.reduce((sum, row) => sum + tokenCount(row), 0));

function modelName(row: UsageLog) {
  return row.model_used || row.model_requested || '未记录';
}

function tokenCount(row: UsageLog) {
  return Number(row.total_tokens || 0);
}

async function load() {
  loading.value = true;
  try {
    [keys.value, usage.value] = await Promise.all([userApi.keys(), userApi.usage()]);
  } finally {
    loading.value = false;
  }
}

onMounted(load);
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h1 class="page-title">网关控制台</h1>
        <p class="page-subtitle">查看账户余额、密钥状态和最近的网关调用。</p>
      </div>
      <a-button type="primary" @click="$router.push('/api-keys')">
        <template #icon><icon-plus /></template>
        新建调用密钥
      </a-button>
    </div>

    <div class="metric-grid">
      <a-card :bordered="false">
        <a-statistic title="余额（美元）" :value="balance" :precision="4" />
      </a-card>
      <a-card :bordered="false">
        <a-statistic title="调用密钥" :value="keys.length" />
      </a-card>
      <a-card :bordered="false">
        <a-statistic title="近期令牌数" :value="totalTokens" />
      </a-card>
      <a-card :bordered="false">
        <a-statistic title="近期消费（美元）" :value="totalCharged" :precision="4" />
      </a-card>
    </div>

    <a-card title="最近请求" :loading="loading" :bordered="false">
      <a-table :data="usage.slice(0, 8)" :pagination="false" row-key="id">
        <template #columns>
          <a-table-column title="请求编号" data-index="request_id" :width="260">
            <template #cell="{ record }"><span class="mono">{{ record.request_id }}</span></template>
          </a-table-column>
          <a-table-column title="模型">
            <template #cell="{ record }">{{ modelName(record) }}</template>
          </a-table-column>
          <a-table-column title="令牌数" :width="110">
            <template #cell="{ record }">{{ tokenCount(record) }}</template>
          </a-table-column>
          <a-table-column title="状态码" data-index="status_code" :width="100">
            <template #cell="{ record }">
              <a-tag :color="record.status_code < 400 ? 'green' : 'red'">{{ record.status_code }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="耗时" data-index="latency_ms" :width="120">
            <template #cell="{ record }">{{ record.latency_ms }} ms</template>
          </a-table-column>
        </template>
      </a-table>
    </a-card>
  </div>
</template>
