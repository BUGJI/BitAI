<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import { Message } from '@arco-design/web-vue';
import { authApi } from '../../api/auth';
import { useAuthStore } from '../../stores/auth';
import { usePublicConfigStore } from '../../stores/publicConfig';

const router = useRouter();
const auth = useAuthStore();
const config = usePublicConfigStore();
const loading = ref(false);
const captchaImage = ref('');
const siteName = computed(() => config.text('site.name', 'BitAPI'));
const form = reactive({
  email: '',
  password: '',
  captcha_token: '',
  captcha_code: ''
});

async function refreshCaptcha() {
  const captcha = await authApi.captcha();
  captchaImage.value = captcha.captcha_image;
  form.captcha_token = captcha.captcha_token;
  form.captcha_code = '';
}

async function submit() {
  loading.value = true;
  try {
    await auth.login(form);
    Message.success('登录成功');
    router.push('/dashboard');
  } catch (error: any) {
    Message.error(error?.response?.data?.message || '登录失败');
    await refreshCaptcha();
  } finally {
    loading.value = false;
  }
}

onMounted(refreshCaptcha);
</script>

<template>
  <div>
    <h2 class="title">登录</h2>
    <p class="subtitle">使用 {{ siteName }} 账号管理密钥、路由和调用明细。</p>
    <a-form layout="vertical" :model="form" @submit-success="submit">
      <a-form-item field="email" label="邮箱" required>
        <a-input v-model="form.email" autocomplete="off" />
      </a-form-item>
      <a-form-item field="password" label="密码" required>
        <a-input-password v-model="form.password" autocomplete="off" />
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
      <a-button type="primary" html-type="submit" long :loading="loading">
        <template #icon><icon-lock /></template>
        登录
      </a-button>
    </a-form>
    <a-divider />
    <a-button type="text" long @click="router.push('/auth/register')">创建账号</a-button>
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
</style>
