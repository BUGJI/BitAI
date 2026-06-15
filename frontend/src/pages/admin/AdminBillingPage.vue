<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import { Message, Modal } from '@arco-design/web-vue';
import { adminApi } from '../../api/admin';
import type { PaymentOrder, RedeemCode } from '../../types';
import { statusLabel } from '../../utils/display';

const loading = ref(false);
const saving = ref(false);
const orders = ref<PaymentOrder[]>([]);
const codes = ref<RedeemCode[]>([]);
const newCode = ref('');
const form = reactive({ dollars: 5, max_uses: 1 });

async function load() {
  loading.value = true;
  try {
    [orders.value, codes.value] = await Promise.all([adminApi.orders(), adminApi.redeemCodes()]);
  } finally {
    loading.value = false;
  }
}

async function markPaid(id: number) {
  saving.value = true;
  try {
    await adminApi.markOrderPaid(id);
    Message.success('订单已标记为已支付');
    await load();
  } catch (error: any) {
    Message.error(error?.response?.data?.message || '更新订单失败');
  } finally {
    saving.value = false;
  }
}

function rejectOrder(id: number) {
  Modal.warning({
    title: '驳回订单',
    content: '确认驳回这笔充值订单？驳回后不会为用户增加余额。',
    hideCancel: false,
    okText: '驳回订单',
    cancelText: '取消',
    onOk: async () => {
      saving.value = true;
      try {
        await adminApi.rejectOrder(id);
        Message.success('订单已驳回');
        await load();
      } catch (error: any) {
        Message.error(error?.response?.data?.message || '驳回订单失败');
      } finally {
        saving.value = false;
      }
    }
  });
}

function orderStatusColor(status: string) {
  const colors: Record<string, string> = {
    pending: 'orange',
    paid: 'green',
    rejected: 'red',
    cancelled: 'gray'
  };
  return colors[status] || 'gray';
}

async function createCode() {
  saving.value = true;
  try {
    const result = await adminApi.createRedeemCode({
      amount_micros: Math.round(form.dollars * 1_000_000),
      max_uses: form.max_uses
    });
    newCode.value = result.code;
    Message.success('兑换码已创建');
    await load();
  } catch (error: any) {
    Message.error(error?.response?.data?.message || '创建兑换码失败');
  } finally {
    saving.value = false;
  }
}

function disableCode(record: RedeemCode) {
  Modal.warning({
    title: '封禁兑换码',
    content: `确认封禁兑换码 ${displayCode(record)}？封禁后用户将无法继续兑换。`,
    hideCancel: false,
    okText: '封禁',
    cancelText: '取消',
    onOk: async () => {
      saving.value = true;
      try {
        await adminApi.disableRedeemCode(record.id);
        Message.success('兑换码已封禁');
        await load();
      } catch (error: any) {
        Message.error(error?.response?.data?.message || '封禁兑换码失败');
      } finally {
        saving.value = false;
      }
    }
  });
}

function enableCode(record: RedeemCode) {
  Modal.info({
    title: '解封兑换码',
    content: `确认解封兑换码 ${displayCode(record)}？解封后用户可以继续兑换。`,
    hideCancel: false,
    okText: '解封',
    cancelText: '取消',
    onOk: async () => {
      saving.value = true;
      try {
        await adminApi.enableRedeemCode(record.id);
        Message.success('兑换码已解封');
        await load();
      } catch (error: any) {
        Message.error(error?.response?.data?.message || '解封兑换码失败');
      } finally {
        saving.value = false;
      }
    }
  });
}

function deleteCode(record: RedeemCode) {
  Modal.warning({
    title: '删除兑换码',
    content: `确认删除兑换码 ${displayCode(record)}？删除后不可恢复。`,
    hideCancel: false,
    okText: '删除',
    cancelText: '取消',
    onOk: async () => {
      saving.value = true;
      try {
        await adminApi.deleteRedeemCode(record.id);
        Message.success('兑换码已删除');
        await load();
      } catch (error: any) {
        Message.error(error?.response?.data?.message || '删除兑换码失败');
      } finally {
        saving.value = false;
      }
    }
  });
}

function displayCode(record: RedeemCode) {
  return record.code || record.code_prefix;
}

onMounted(load);
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h1 class="page-title">充值兑换</h1>
        <p class="page-subtitle">处理人工充值订单，并创建余额兑换码。</p>
      </div>
      <a-button @click="load"><template #icon><icon-refresh /></template>刷新</a-button>
    </div>

    <a-alert v-if="newCode" type="success" show-icon>
      <template #title>请立即复制兑换码</template>
      <div class="mono">{{ newCode }}</div>
    </a-alert>

    <a-card title="创建兑换码" :bordered="false">
      <a-form layout="inline" :model="form">
        <a-form-item label="金额（美元）">
          <a-input-number v-model="form.dollars" :min="0.000001" :precision="6" />
        </a-form-item>
        <a-form-item label="最大使用次数">
          <a-input-number v-model="form.max_uses" :min="1" />
        </a-form-item>
        <a-button type="primary" :loading="saving" @click="createCode">
          <template #icon><icon-plus /></template>
          创建兑换码
        </a-button>
      </a-form>
    </a-card>

    <a-card title="充值订单" :bordered="false">
      <a-table :data="orders" :loading="loading" row-key="id" :scroll="{ x: 980 }">
        <template #columns>
          <a-table-column title="订单号" data-index="order_no" :width="300">
            <template #cell="{ record }"><span class="mono">{{ record.order_no }}</span></template>
          </a-table-column>
          <a-table-column title="用户" data-index="user_id" :width="90" />
          <a-table-column title="金额（美元）" :width="130">
            <template #cell="{ record }">{{ (record.amount_micros / 1_000_000).toFixed(4) }}</template>
          </a-table-column>
          <a-table-column title="状态" data-index="status" :width="120">
            <template #cell="{ record }"><a-tag :color="orderStatusColor(record.status)">{{ statusLabel(record.status) }}</a-tag></template>
          </a-table-column>
          <a-table-column title="操作" :width="240">
            <template #cell="{ record }">
              <a-space v-if="record.status === 'pending'">
                <a-button type="text" :loading="saving" @click="markPaid(record.id)">
                  <template #icon><icon-check /></template>
                  标记已支付
                </a-button>
                <a-button type="text" status="danger" :loading="saving" @click="rejectOrder(record.id)">
                  <template #icon><icon-close /></template>
                  驳回订单
                </a-button>
              </a-space>
            </template>
          </a-table-column>
        </template>
      </a-table>
    </a-card>

    <a-card title="兑换码" :bordered="false">
      <a-table :data="codes" :loading="loading" row-key="id" :scroll="{ x: 900 }">
        <template #columns>
          <a-table-column title="兑换码" data-index="code" :width="220">
            <template #cell="{ record }"><span class="mono">{{ displayCode(record) }}</span></template>
          </a-table-column>
          <a-table-column title="金额（美元）" :width="130">
            <template #cell="{ record }">{{ (record.amount_micros / 1_000_000).toFixed(4) }}</template>
          </a-table-column>
          <a-table-column title="使用次数" :width="110">
            <template #cell="{ record }">{{ record.used_count }} / {{ record.max_uses }}</template>
          </a-table-column>
          <a-table-column title="状态" data-index="status" :width="120">
            <template #cell="{ record }">{{ statusLabel(record.status) }}</template>
          </a-table-column>
          <a-table-column title="创建时间" data-index="created_at" :width="190" />
          <a-table-column title="操作" :width="240">
            <template #cell="{ record }">
              <a-space>
                <a-button v-if="record.status === 'active'" type="text" status="warning" :loading="saving" @click="disableCode(record)">
                  <template #icon><icon-stop /></template>
                  封禁
                </a-button>
                <a-button v-if="record.status === 'disabled'" type="text" status="success" :loading="saving" @click="enableCode(record)">
                  <template #icon><icon-check-circle /></template>
                  解封
                </a-button>
                <a-button type="text" status="danger" :loading="saving" @click="deleteCode(record)">
                  <template #icon><icon-delete /></template>
                  删除
                </a-button>
              </a-space>
            </template>
          </a-table-column>
        </template>
      </a-table>
    </a-card>
  </div>
</template>
