<script setup lang="ts">
import { reactive, ref, onMounted } from 'vue';
import { Message } from '@arco-design/web-vue';
import { adminApi } from '../../api/admin';
import type { User } from '../../types';
import { roleLabel, roleOptions, statusLabel, statusOptions } from '../../utils/display';

const loading = ref(false);
const users = ref<User[]>([]);
const editVisible = ref(false);
const rechargeVisible = ref(false);
const saving = ref(false);
const current = ref<User | null>(null);
const editForm = reactive({
  role: '',
  status: '',
  concurrency_limit: 0,
  rpm_limit: 0
});
const rechargeForm = reactive({
  dollars: 10
});

async function load() {
  loading.value = true;
  try {
    users.value = await adminApi.users();
  } finally {
    loading.value = false;
  }
}

function openEdit(record: User) {
  current.value = record;
  editForm.role = record.role;
  editForm.status = record.status;
  editForm.concurrency_limit = record.concurrency_limit;
  editForm.rpm_limit = record.rpm_limit;
  editVisible.value = true;
}

function openRecharge(record: User) {
  current.value = record;
  rechargeForm.dollars = 10;
  rechargeVisible.value = true;
}

async function saveUser() {
  if (!current.value) return;
  saving.value = true;
  try {
    await adminApi.updateUser(current.value.id, {
      role: editForm.role as User['role'],
      status: editForm.status,
      concurrency_limit: editForm.concurrency_limit,
      rpm_limit: editForm.rpm_limit
    });
    Message.success('用户已更新');
    editVisible.value = false;
    await load();
  } catch (error: any) {
    Message.error(error?.response?.data?.message || '更新用户失败');
  } finally {
    saving.value = false;
  }
}

async function rechargeUser() {
  if (!current.value) return;
  saving.value = true;
  try {
    await adminApi.rechargeUser(current.value.id, Math.round(rechargeForm.dollars * 1_000_000));
    Message.success('余额已更新');
    rechargeVisible.value = false;
    await load();
  } catch (error: any) {
    Message.error(error?.response?.data?.message || '用户充值失败');
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
        <h1 class="page-title">用户管理</h1>
        <p class="page-subtitle">管理账号、角色、余额和账号状态。</p>
      </div>
      <a-button @click="load"><template #icon><icon-refresh /></template>刷新</a-button>
    </div>

    <a-card :bordered="false">
      <a-table :data="users" :loading="loading" row-key="id">
        <template #columns>
          <a-table-column title="编号" data-index="id" :width="80" />
          <a-table-column title="邮箱" data-index="email" />
          <a-table-column title="名称" data-index="display_name" />
          <a-table-column title="角色" data-index="role" :width="120">
            <template #cell="{ record }"><a-tag>{{ roleLabel(record.role) }}</a-tag></template>
          </a-table-column>
          <a-table-column title="状态" data-index="status" :width="120">
            <template #cell="{ record }"><a-tag color="green">{{ statusLabel(record.status) }}</a-tag></template>
          </a-table-column>
          <a-table-column title="余额（美元）" :width="140">
            <template #cell="{ record }">{{ (record.balance_micros / 1_000_000).toFixed(4) }}</template>
          </a-table-column>
          <a-table-column title="操作" :width="180">
            <template #cell="{ record }">
              <a-space>
                <a-button type="text" @click="openEdit(record)"><template #icon><icon-edit /></template></a-button>
                <a-button type="text" @click="openRecharge(record)"><template #icon><icon-plus-circle /></template></a-button>
              </a-space>
            </template>
          </a-table-column>
        </template>
      </a-table>
    </a-card>

    <a-modal v-model:visible="editVisible" title="编辑用户" :confirm-loading="saving" @ok="saveUser">
      <a-form layout="vertical" :model="editForm">
        <a-form-item label="角色">
          <a-select v-model="editForm.role" :options="roleOptions" />
        </a-form-item>
        <a-form-item label="状态">
          <a-select v-model="editForm.status" :options="statusOptions" />
        </a-form-item>
        <a-row :gutter="12">
          <a-col :span="12"><a-form-item label="并发限制"><a-input-number v-model="editForm.concurrency_limit" :min="0" /></a-form-item></a-col>
          <a-col :span="12"><a-form-item label="每分钟请求数"><a-input-number v-model="editForm.rpm_limit" :min="0" /></a-form-item></a-col>
        </a-row>
      </a-form>
    </a-modal>

    <a-modal v-model:visible="rechargeVisible" title="用户充值" :confirm-loading="saving" @ok="rechargeUser">
      <a-form layout="vertical" :model="rechargeForm">
        <a-form-item label="金额（美元）">
          <a-input-number v-model="rechargeForm.dollars" :min="0.000001" :precision="6" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>
