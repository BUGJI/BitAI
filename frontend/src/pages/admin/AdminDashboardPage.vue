<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { adminApi, type AdminStats } from '../../api/admin';
import type { UsageLog } from '../../types';

const loading = ref(false);
const stats = ref<AdminStats | null>(null);
const usage = ref<UsageLog[]>([]);
const charged = computed(() => (stats.value?.charged_micros || 0) / 1_000_000);

async function load() {
  loading.value = true;
  try {
    [stats.value, usage.value] = await Promise.all([adminApi.stats(), adminApi.usage(8)]);
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
        <h1 class="page-title">管理概览</h1>
        <p class="page-subtitle">查看网关资源、计费汇总和最新调用。</p>
      </div>
      <a-button @click="load"><template #icon><icon-refresh /></template>刷新</a-button>
    </div>

    <div class="metric-grid">
      <a-card :bordered="false"><a-statistic title="用户数" :value="stats?.users || 0" /></a-card>
      <a-card :bordered="false"><a-statistic title="调用密钥" :value="stats?.api_keys || 0" /></a-card>
      <a-card :bordered="false"><a-statistic title="上游账号" :value="stats?.accounts || 0" /></a-card>
      <a-card :bordered="false"><a-statistic title="扣费金额（美元）" :value="charged" :precision="4" /></a-card>
    </div>

    <a-card title="最新网关调用" :loading="loading" :bordered="false">
      <a-table :data="usage" row-key="id" :pagination="false">
        <template #columns>
          <a-table-column title="请求编号" data-index="request_id">
            <template #cell="{ record }"><span class="mono">{{ record.request_id }}</span></template>
          </a-table-column>
          <a-table-column title="用户" data-index="user_id" :width="90" />
          <a-table-column title="模型" data-index="model_used" />
          <a-table-column title="令牌数" data-index="total_tokens" :width="110" />
          <a-table-column title="状态码" data-index="status_code" :width="100" />
        </template>
      </a-table>
    </a-card>
  </div>
</template>
