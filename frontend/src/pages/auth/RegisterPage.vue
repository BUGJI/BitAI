<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import { Message } from '@arco-design/web-vue';
import { authApi } from '../../api/auth';
import { useAuthStore } from '../../stores/auth';

const router = useRouter();
const auth = useAuthStore();
const loading = ref(false);
const sendingCode = ref(false);
const captchaImage = ref('');
const form = reactive({
  email: '',
  display_name: '',
  password: '',
  captcha_token: '',
  captcha_code: '',
  email_token: '',
  email_code: ''
});

async function refreshCaptcha() {
  const captcha = await authApi.captcha();
  captchaImage.value = captcha.captcha_image;
  form.captcha_token = captcha.captcha_token;
  form.captcha_code = '';
}

async function sendEmailCode() {
  if (!form.email || !form.captcha_code) {
    Message.warning('请先填写邮箱和图形验证码');
    return;
  }
  sendingCode.value = true;
  try {
    const result = await authApi.sendEmailCode({
      email: form.email,
      captcha_token: form.captcha_token,
      captcha_code: form.captcha_code
    });
    form.email_token = result.email_token;
    Message.success('邮箱验证码已发送');
    await refreshCaptcha();
  } catch (error: any) {
    Message.error(error?.response?.data?.message || '发送邮箱验证码失败');
    await refreshCaptcha();
  } finally {
    sendingCode.value = false;
  }
}

async function submit() {
  loading.value = true;
  try {
    await auth.register(form);
    Message.success('账号已创建');
    router.push('/dashboard');
  } catch (error: any) {
    Message.error(error?.response?.data?.message || '注册失败');
    await refreshCaptcha();
  } finally {
    loading.value = false;
  }
}

onMounted(refreshCaptcha);
</script>

<template>
  <div>
    <h2 class="title">创建账号</h2>
    <p class="subtitle">先创建个人工作区，管理员可在之后授予更多权限。</p>
    <a-form layout="vertical" :model="form" @submit-success="submit">
      <a-form-item field="email" label="邮箱" required>
        <a-input v-model="form.email" autocomplete="email" />
      </a-form-item>
      <a-form-item field="display_name" label="显示名称">
        <a-input v-model="form.display_name" />
      </a-form-item>
      <a-form-item field="password" label="密码" required>
        <a-input-password v-model="form.password" autocomplete="new-password" />
      </a-form-item>
      <a-form-item field="captcha_code" label="图形验证码" required>
        <a-input v-model="form.captcha_code" autocomplete="off" placeholder="请输入右侧验证码">
          <template #append>
            <button class="captcha-button" type="button" @click="refreshCaptcha">
              <img v-if="captchaImage" :src="captchaImage" alt="图形验证码" />
            </button>
          </template>
        </a-input>
      </a-form-item>
      <a-form-item field="email_code" label="邮箱验证码" required>
        <div class="email-code-row">
          <a-input v-model="form.email_code" autocomplete="off" placeholder="请输入邮箱验证码" />
          <a-button type="primary" :loading="sendingCode" @click="sendEmailCode">获取验证码</a-button>
        </div>
      </a-form-item>
      <a-button type="primary" html-type="submit" long :loading="loading">
        <template #icon><icon-plus /></template>
        创建账号
      </a-button>
    </a-form>
    <a-divider />
    <a-button type="text" long @click="router.push('/auth/login')">返回登录</a-button>
  </div>
</template>

<style scoped>
.title {
  margin: 0;
  font-size: 24px;
}

.subtitle {
  margin: 8px 0 22px;
  color: var(--bitapi-muted);
}

.captcha-button {
  width: 150px;
  height: 40px;
  padding: 0;
  border: 0;
  background: transparent;
  cursor: pointer;
}

.captcha-button img {
  width: 150px;
  height: 40px;
  display: block;
  object-fit: cover;
}

.email-code-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 10px;
  width: 100%;
  align-items: center;
}

.email-code-row :deep(.arco-btn) {
  min-width: 104px;
}
</style>
