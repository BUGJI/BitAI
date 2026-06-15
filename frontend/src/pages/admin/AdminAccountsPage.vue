<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { Message, Modal } from '@arco-design/web-vue';
import { adminApi } from '../../api/admin';
import type { Group, GroupAccount, UpstreamAccount } from '../../types';
import { platformLabel, platformOptions, statusLabel } from '../../utils/display';

const loading = ref(false);
const creating = ref(false);
const linking = ref(false);
const checking = ref<number | null>(null);
const visible = ref(false);
const linkVisible = ref(false);
const editingId = ref<number | null>(null);
const accounts = ref<UpstreamAccount[]>([]);
const groups = ref<Group[]>([]);
const links = ref<GroupAccount[]>([]);
const groupOptions = computed(() => groups.value.map((g) => ({ label: `${g.id} - ${g.name}`, value: g.id })));
const accountOptions = computed(() => accounts.value.map((a) => ({ label: `${a.id} - ${a.name}`, value: a.id })));
const form = reactive<Partial<UpstreamAccount>>({
  name: '',
  platform: 'openai',
  auth_type: 'api_key',
  credentials: '',
  base_url: 'https://api.openai.com',
  priority: 100,
  weight: 1,
  status: 'active',
  schedulable: true
});
const linkForm = reactive({
  group_id: undefined as number | undefined,
  upstream_account_id: undefined as number | undefined,
  weight: 1,
  priority: 100
});

async function load() {
  loading.value = true;
  try {
    [accounts.value, groups.value, links.value] = await Promise.all([adminApi.accounts(), adminApi.groups(), adminApi.groupAccounts()]);
  } finally {
    loading.value = false;
  }
}

async function createAccount() {
  creating.value = true;
  try {
    if (editingId.value) {
      await adminApi.updateAccount(editingId.value, form);
      Message.success('上游账号已更新');
    } else {
      await adminApi.createAccount(form);
      Message.success('上游账号已创建');
    }
    visible.value = false;
    editingId.value = null;
    await load();
  } catch (error: any) {
    Message.error(error?.response?.data?.message || '保存上游账号失败');
  } finally {
    creating.value = false;
  }
}

function openCreate() {
  editingId.value = null;
  Object.assign(form, {
    name: '',
    platform: 'openai',
    auth_type: 'api_key',
    credentials: '',
    base_url: 'https://api.openai.com',
    priority: 100,
    weight: 1,
    status: 'active',
    schedulable: true
  });
  visible.value = true;
}

function openEdit(record: UpstreamAccount) {
  editingId.value = record.id;
  Object.assign(form, record);
  visible.value = true;
}

function confirmDelete(record: UpstreamAccount) {
  Modal.warning({
    title: '删除上游账号',
    content: `确认删除 ${record.name}？`,
    hideCancel: false,
    onOk: async () => {
      await adminApi.deleteAccount(record.id);
      Message.success('上游账号已删除');
      await load();
    }
  });
}

async function checkAccount(record: UpstreamAccount) {
  checking.value = record.id;
  try {
    const result = await adminApi.checkAccount(record.id);
    if (result.ok) {
      Message.success(`上游账号检查通过，耗时 ${result.latency_ms} ms`);
    } else {
      Message.warning(result.error || `上游返回状态码 ${result.status}`);
    }
    await load();
  } catch (error: any) {
    Message.error(error?.response?.data?.message || '检查失败');
  } finally {
    checking.value = null;
  }
}

async function linkAccount() {
  if (!linkForm.group_id || !linkForm.upstream_account_id) return;
  linking.value = true;
  try {
    await adminApi.linkGroupAccount(linkForm as { group_id: number; upstream_account_id: number; weight: number; priority: number });
    Message.success('绑定已创建');
    linkVisible.value = false;
  } catch (error: any) {
    Message.error(error?.response?.data?.message || '绑定上游账号失败');
  } finally {
    linking.value = false;
  }
}

function confirmUnlink(record: GroupAccount) {
  Modal.warning({
    title: '移除绑定',
    content: `确认从 ${record.group_name} 移除 ${record.upstream_name}？`,
    hideCancel: false,
    onOk: async () => {
      await adminApi.deleteGroupAccount(record.id);
      Message.success('绑定已移除');
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
        <h1 class="page-title">上游账号</h1>
        <p class="page-subtitle">管理供应商凭据、调度状态和分组绑定。</p>
      </div>
      <a-space>
        <a-button @click="linkVisible = true"><template #icon><icon-link /></template>绑定</a-button>
        <a-button type="primary" @click="openCreate"><template #icon><icon-plus /></template>创建</a-button>
      </a-space>
    </div>

    <a-card :bordered="false">
      <a-table :data="accounts" :loading="loading" row-key="id" :scroll="{ x: 1120 }">
        <template #columns>
          <a-table-column title="编号" data-index="id" :width="80" />
          <a-table-column title="名称" data-index="name" />
          <a-table-column title="平台" data-index="platform" :width="130">
            <template #cell="{ record }">{{ platformLabel(record.platform) }}</template>
          </a-table-column>
          <a-table-column title="基础地址" data-index="base_url" />
          <a-table-column title="优先级" data-index="priority" :width="100" />
          <a-table-column title="状态" data-index="status" :width="120">
            <template #cell="{ record }"><a-tag color="green">{{ statusLabel(record.status) }}</a-tag></template>
          </a-table-column>
          <a-table-column title="可调度" data-index="schedulable" :width="130">
            <template #cell="{ record }"><a-switch :model-value="record.schedulable" disabled /></template>
          </a-table-column>
          <a-table-column title="操作" :width="130">
            <template #cell="{ record }">
              <a-space>
                <a-button type="text" @click="openEdit(record)"><template #icon><icon-edit /></template></a-button>
                <a-button type="text" :loading="checking === record.id" @click="checkAccount(record)"><template #icon><icon-sync /></template></a-button>
                <a-button type="text" status="danger" @click="confirmDelete(record)"><template #icon><icon-delete /></template></a-button>
              </a-space>
            </template>
          </a-table-column>
        </template>
      </a-table>
    </a-card>

    <a-card title="分组绑定" :bordered="false">
      <a-table :data="links" :loading="loading" row-key="id" :pagination="{ pageSize: 8 }">
        <template #columns>
          <a-table-column title="分组" data-index="group_name" />
          <a-table-column title="上游账号" data-index="upstream_name" />
          <a-table-column title="优先级" data-index="priority" :width="100" />
          <a-table-column title="权重" data-index="weight" :width="100" />
          <a-table-column title="启用" data-index="enabled" :width="110">
            <template #cell="{ record }"><a-tag :color="record.enabled ? 'green' : 'gray'">{{ record.enabled ? '已启用' : '已禁用' }}</a-tag></template>
          </a-table-column>
          <a-table-column title="操作" :width="100">
            <template #cell="{ record }"><a-button type="text" status="danger" @click="confirmUnlink(record)"><template #icon><icon-delete /></template></a-button></template>
          </a-table-column>
        </template>
      </a-table>
    </a-card>

    <a-modal v-model:visible="visible" :title="editingId ? '编辑上游账号' : '创建上游账号'" :confirm-loading="creating" width="680px" @ok="createAccount">
      <a-form layout="vertical" :model="form">
        <a-row :gutter="12">
          <a-col :span="12"><a-form-item label="名称"><a-input v-model="form.name" /></a-form-item></a-col>
          <a-col :span="12"><a-form-item label="平台"><a-select v-model="form.platform" :options="platformOptions" /></a-form-item></a-col>
        </a-row>
        <a-form-item label="基础地址"><a-input v-model="form.base_url" /></a-form-item>
        <a-form-item label="接口密钥"><a-input-password v-model="form.credentials" /></a-form-item>
        <a-row :gutter="12">
          <a-col :span="12"><a-form-item label="优先级"><a-input-number v-model="form.priority" /></a-form-item></a-col>
          <a-col :span="12"><a-form-item label="权重"><a-input-number v-model="form.weight" :min="1" /></a-form-item></a-col>
        </a-row>
      </a-form>
    </a-modal>

    <a-modal v-model:visible="linkVisible" title="绑定上游账号到分组" :confirm-loading="linking" @ok="linkAccount">
      <a-form layout="vertical" :model="linkForm">
        <a-form-item label="分组"><a-select v-model="linkForm.group_id" :options="groupOptions" /></a-form-item>
        <a-form-item label="上游账号"><a-select v-model="linkForm.upstream_account_id" :options="accountOptions" /></a-form-item>
        <a-form-item label="优先级"><a-input-number v-model="linkForm.priority" /></a-form-item>
        <a-form-item label="权重"><a-input-number v-model="linkForm.weight" :min="1" /></a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>
