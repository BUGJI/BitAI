<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import { Message, Modal } from '@arco-design/web-vue';
import { adminApi } from '../../api/admin';
import type { Group } from '../../types';
import { modeLabel, modeOptions, platformLabel, platformOptions, statusLabel } from '../../utils/display';

const loading = ref(false);
const creating = ref(false);
const visible = ref(false);
const editingId = ref<number | null>(null);
const rows = ref<Group[]>([]);
const form = reactive<Partial<Group>>({
  name: '',
  description: '',
  platform: 'openai',
  mode: 'balance',
  status: 'active',
  rate_multiplier_ppm: 1000000,
  model_mapping_json: '{}',
  model_list_json: '["gpt-4o-mini"]',
  features_json: '{"chat":true,"stream":true}',
  sort_order: 10
});

async function load() {
  loading.value = true;
  try {
    rows.value = await adminApi.groups();
  } finally {
    loading.value = false;
  }
}

async function createGroup() {
	creating.value = true;
	try {
		if (editingId.value) {
			await adminApi.updateGroup(editingId.value, form);
			Message.success('分组已更新');
		} else {
			await adminApi.createGroup(form);
			Message.success('分组已创建');
		}
		visible.value = false;
		editingId.value = null;
		await load();
  } catch (error: any) {
    Message.error(error?.response?.data?.message || '保存分组失败');
  } finally {
    creating.value = false;
  }
}

function openCreate() {
	editingId.value = null;
	Object.assign(form, {
		name: '',
		description: '',
		platform: 'openai',
		mode: 'balance',
		status: 'active',
		rate_multiplier_ppm: 1000000,
		model_mapping_json: '{}',
		model_list_json: '["gpt-4o-mini"]',
		features_json: '{"chat":true,"stream":true}',
		sort_order: 10
	});
	visible.value = true;
}

function openEdit(record: Group) {
	editingId.value = record.id;
	Object.assign(form, record);
	visible.value = true;
}

function confirmDelete(record: Group) {
	Modal.warning({
		title: '删除分组',
		content: `确认删除 ${record.name}？已有调用密钥可能失去路由分组。`,
		hideCancel: false,
		onOk: async () => {
			await adminApi.deleteGroup(record.id);
			Message.success('分组已删除');
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
        <h1 class="page-title">分组管理</h1>
        <p class="page-subtitle">配置路由策略、模型列表、计费模式和功能开关。</p>
      </div>
      <a-button type="primary" @click="openCreate">
        <template #icon><icon-plus /></template>
        创建分组
      </a-button>
    </div>

    <a-card :bordered="false">
      <a-table :data="rows" :loading="loading" row-key="id" :scroll="{ x: 1180 }">
        <template #columns>
          <a-table-column title="编号" data-index="id" :width="80" />
          <a-table-column title="名称" data-index="name" :width="190" />
          <a-table-column title="平台" data-index="platform" :width="130">
            <template #cell="{ record }">{{ platformLabel(record.platform) }}</template>
          </a-table-column>
          <a-table-column title="模式" data-index="mode" :width="120">
            <template #cell="{ record }">{{ modeLabel(record.mode) }}</template>
          </a-table-column>
          <a-table-column title="状态" data-index="status" :width="120">
            <template #cell="{ record }"><a-tag color="green">{{ statusLabel(record.status) }}</a-tag></template>
          </a-table-column>
          <a-table-column title="费率" :width="110">
            <template #cell="{ record }">{{ (record.rate_multiplier_ppm / 1_000_000).toFixed(2) }}x</template>
          </a-table-column>
          <a-table-column title="模型列表" data-index="model_list_json" />
          <a-table-column title="操作" :width="130">
            <template #cell="{ record }">
              <a-space>
                <a-button type="text" @click="openEdit(record)"><template #icon><icon-edit /></template></a-button>
                <a-button type="text" status="danger" @click="confirmDelete(record)"><template #icon><icon-delete /></template></a-button>
              </a-space>
            </template>
          </a-table-column>
        </template>
      </a-table>
    </a-card>

    <a-modal v-model:visible="visible" :title="editingId ? '编辑分组' : '创建分组'" :confirm-loading="creating" width="720px" @ok="createGroup">
      <a-form layout="vertical" :model="form">
        <a-row :gutter="12">
          <a-col :span="12"><a-form-item label="名称"><a-input v-model="form.name" /></a-form-item></a-col>
          <a-col :span="12"><a-form-item label="平台"><a-select v-model="form.platform" :options="platformOptions" /></a-form-item></a-col>
        </a-row>
        <a-row :gutter="12">
          <a-col :span="12"><a-form-item label="计费模式"><a-select v-model="form.mode" :options="modeOptions" /></a-form-item></a-col>
          <a-col :span="12"><a-form-item label="费率倍率（ppm）"><a-input-number v-model="form.rate_multiplier_ppm" :min="1" /></a-form-item></a-col>
        </a-row>
        <a-form-item label="描述"><a-textarea v-model="form.description" /></a-form-item>
        <a-form-item label="模型列表配置"><a-textarea v-model="form.model_list_json" class="mono" /></a-form-item>
        <a-form-item label="功能开关配置"><a-textarea v-model="form.features_json" class="mono" /></a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>
