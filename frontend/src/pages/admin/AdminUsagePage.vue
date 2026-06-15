<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { adminApi } from '../../api/admin';
import type { UsageLog } from '../../types';

const loading = ref(false);
const rows = ref<UsageLog[]>([]);

async function load() {
  loading.value = true;
  try {
    rows.value = await adminApi.usage(500);
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
        <h1 class="page-title">调用日志</h1>
        <p class="page-subtitle">查看全局请求流水、计费记录和上游归因。</p>
      </div>
      <a-button @click="load"><template #icon><icon-refresh /></template>刷新</a-button>
    </div>

    <a-card :bordered="false">
      <a-table :data="rows" :loading="loading" row-key="id" :scroll="{ x: 1320 }">
        <template #columns>
          <a-table-column title="时间" data-index="created_at" :width="190" />
          <a-table-column title="请求编号" data-index="request_id" :width="260">
            <template #cell="{ record }"><span class="mono">{{ record.request_id }}</span></template>
          </a-table-column>
          <a-table-column title="用户" data-index="user_id" :width="90" />
          <a-table-column title="密钥" data-index="api_key_id" :width="90" />
          <a-table-column title="分组" data-index="group_id" :width="90" />
          <a-table-column title="上游" data-index="upstream_account_id" :width="110" />
          <a-table-column title="模型" data-index="model_used" :width="170" />
          <a-table-column title="令牌数" data-index="total_tokens" :width="110" />
          <a-table-column title="扣费" :width="120">
            <template #cell="{ record }">{{ (record.charged_micros / 1_000_000).toFixed(6) }}</template>
          </a-table-column>
          <a-table-column title="状态码" data-index="status_code" :width="100">
            <template #cell="{ record }">
              <a-tag :color="record.status_code < 400 ? 'green' : 'red'">{{ record.status_code }}</a-tag>
            </template>
          </a-table-column>
        </template>
      </a-table>
    </a-card>
  </div>
</template>
