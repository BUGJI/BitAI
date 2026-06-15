<script setup lang="ts">
import bitLogo from '../assets/Bit.svg';
</script>

<template>
  <main class="auth-shell">
    <section class="auth-copy">
      <img class="brand-mark" :src="bitLogo" alt="BitAPI" />
      <h1>BitAPI</h1>
      <p>统一管理 AI 模型接口的接入、路由、计费与运营。</p>
    </section>
    <section class="auth-panel surface">
      <router-view />
    </section>
  </main>
</template>

<style scoped>
.auth-shell {
  min-height: 100vh;
  display: grid;
  grid-template-columns: minmax(0, 560px) 420px;
  gap: 56px;
  align-items: center;
  justify-content: center;
  padding: 48px;
  background:
    linear-gradient(135deg, rgba(22, 93, 255, 0.08), transparent 36%),
    linear-gradient(315deg, rgba(0, 180, 42, 0.08), transparent 34%),
    var(--bitapi-bg);
}

.auth-copy {
  max-width: 620px;
}

.brand-mark {
  width: 56px;
  height: 56px;
  display: block;
  object-fit: contain;
}

.auth-copy h1 {
  margin: 24px 0 10px;
  font-size: 56px;
  letter-spacing: 0;
}

.auth-copy p {
  max-width: 520px;
  margin: 0 0 20px;
  color: var(--bitapi-muted);
  font-size: 18px;
  line-height: 1.7;
}

.auth-panel {
  padding: 28px;
}

@media (max-width: 840px) {
  .auth-shell {
    grid-template-columns: 1fr;
    padding: 28px;
  }

  .auth-copy h1 {
    font-size: 40px;
  }
}
</style>
