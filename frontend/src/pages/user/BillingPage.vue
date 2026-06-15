<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import { Message } from '@arco-design/web-vue';
import { userApi } from '../../api/user';
import type { PaymentOrder } from '../../types';
import { providerLabel, statusLabel } from '../../utils/display';

const loading = ref(false);
const submitting = ref(false);
const orders = ref<PaymentOrder[]>([]);
const form = reactive({ dollars: 10, provider: 'manual' });
const redeem = reactive({ code: '' });

async function load() {
  loading.value = true;
  try {
    orders.value = await userApi.orders();
  } finally {
    loading.value = false;
  }
}

async function createOrder() {
  submitting.value = true;
  try {
    await userApi.createOrder({ amount_micros: Math.round(form.dollars * 1_000_000), provider: form.provider });
    Message.success('充值订单已创建');
    await load();
  } catch (error: any) {
    Message.error(error?.response?.data?.message || '创建充值订单失败');
  } finally {
    submitting.value = false;
  }
}

async function redeemCode() {
  submitting.value = true;
  try {
    await userApi.redeem(redeem.code);
    Message.success('兑换成功');
    redeem.code = '';
    await load();
  } catch (error: any) {
    Message.error(error?.response?.data?.message || '兑换失败');
  } finally {
    submitting.value = false;
  }
}

onMounted(load);
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h1 class="page-title">费用中心</h1>
        <p class="page-subtitle">创建人工充值订单，或使用兑换码为账户充值。</p>
      </div>
      <a-button @click="load"><template #icon><icon-refresh /></template>刷新</a-button>
    </div>

    <a-row :gutter="16">
      <a-col :xs="24" :md="12">
        <a-card title="创建充值订单" :bordered="false">
          <a-form layout="vertical" :model="form">
            <a-form-item label="金额（美元）">
              <a-input-number v-model="form.dollars" :min="0.000001" :precision="6" />
            </a-form-item>
            <a-form-item label="处理方式">
              <a-select v-model="form.provider" :options="[{ label: '人工处理', value: 'manual' }]" />
            </a-form-item>
            <a-button type="primary" :loading="submitting" @click="createOrder">
              <template #icon><icon-plus /></template>
              创建订单
            </a-button>
          </a-form>
        </a-card>
      </a-col>
      <a-col :xs="24" :md="12">
        <a-card title="兑换码充值" :bordered="false">
          <a-form layout="vertical" :model="redeem">
            <a-form-item label="兑换码">
              <a-input v-model="redeem.code" class="mono" />
            </a-form-item>
            <a-button type="primary" status="success" :loading="submitting" @click="redeemCode">
              <template #icon><icon-gift /></template>
              立即兑换
            </a-button>
          </a-form>
        </a-card>
      </a-col>
    </a-row>

    <a-card title="充值订单" :bordered="false">
      <a-table :data="orders" :loading="loading" row-key="id">
        <template #columns>
          <a-table-column title="订单号" data-index="order_no">
            <template #cell="{ record }"><span class="mono">{{ record.order_no }}</span></template>
          </a-table-column>
          <a-table-column title="金额（美元）" :width="140">
            <template #cell="{ record }">{{ (record.amount_micros / 1_000_000).toFixed(4) }}</template>
          </a-table-column>
          <a-table-column title="状态" data-index="status" :width="120">
            <template #cell="{ record }"><a-tag :color="record.status === 'paid' ? 'green' : 'orange'">{{ statusLabel(record.status) }}</a-tag></template>
          </a-table-column>
          <a-table-column title="处理方式" data-index="provider" :width="120">
            <template #cell="{ record }">{{ providerLabel(record.provider) }}</template>
          </a-table-column>
          <a-table-column title="创建时间" data-index="created_at" :width="190" />
        </template>
      </a-table>
    </a-card>
  </div>
</template>
