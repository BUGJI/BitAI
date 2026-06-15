<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { userApi } from '../../api/user';
import type { UsageLog } from '../../types';

const loading = ref(false);
const rows = ref<UsageLog[]>([]);

function modelName(row: UsageLog) {
  return row.model_used || row.model_requested || '未记录';
}

function tokenCount(row: UsageLog) {
  return Number(row.total_tokens || 0);
}

async function load() {
  loading.value = true;
  try {
    rows.value = await userApi.usage();
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
        <h1 class="page-title">使用明细</h1>
        <p class="page-subtitle">查看当前账户近期的网关请求和计费记录。</p>
      </div>
      <a-button @click="load"><template #icon><icon-refresh /></template>刷新</a-button>
    </div>

    <a-card :bordered="false">
      <a-table :data="rows" :loading="loading" row-key="id" :scroll="{ x: 1180 }">
        <template #columns>
          <a-table-column title="时间" data-index="created_at" :width="190" />
          <a-table-column title="请求编号" data-index="request_id" :width="270">
            <template #cell="{ record }"><span class="mono">{{ record.request_id }}</span></template>
          </a-table-column>
          <a-table-column title="平台" data-index="platform" :width="110" />
          <a-table-column title="模型" :width="180">
            <template #cell="{ record }">{{ modelName(record) }}</template>
          </a-table-column>
          <a-table-column title="令牌数" :width="110">
            <template #cell="{ record }">{{ tokenCount(record) }}</template>
          </a-table-column>
          <a-table-column title="扣费" :width="120">
            <template #cell="{ record }">{{ (record.charged_micros / 1_000_000).toFixed(6) }}</template>
          </a-table-column>
          <a-table-column title="状态码" data-index="status_code" :width="100">
            <template #cell="{ record }">
              <a-tag :color="record.status_code < 400 ? 'green' : 'red'">{{ record.status_code }}</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="耗时" data-index="latency_ms" :width="110">
            <template #cell="{ record }">{{ record.latency_ms }} ms</template>
          </a-table-column>
        </template>
      </a-table>
    </a-card>
  </div>
</template>
