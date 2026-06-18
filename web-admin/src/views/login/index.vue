<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useSleepUserStore } from '@/stores/user'
import { applyTheme } from '@/stores/theme'

// Ensure theme CSS vars are applied (in case of direct navigation)
applyTheme()

const router = useRouter()
const route = useRoute()
const userStore = useSleepUserStore()

const form = ref({ username: '', password: '' })
const loading = ref(false)

function isSafeRedirect(redirect: string | undefined): redirect is string {
  return typeof redirect === 'string' && redirect.startsWith('/') && !redirect.startsWith('//')
}

async function handleLogin() {
  if (!form.value.username.trim() || !form.value.password) {
    ElMessage.warning('请输入账号和密码')
    return
  }
  loading.value = true
  try {
    await userStore.loginAction(form.value)
    const redirect = route.query.redirect as string | undefined
    router.push(isSafeRedirect(redirect) ? redirect : '/')
  } catch {
    ElMessage.error('账号或密码错误')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <!-- Nebula background (matches layout/index.vue cloud system) -->
    <div class="bg-scene" aria-hidden="true">
      <div class="cloud cp1" />
      <div class="cloud cp2" />
      <div class="cloud cp3" />
      <div class="cloud cw1" />
      <div class="cloud cw2" />
    </div>

    <!-- Login card -->
    <div class="login-card glass-card">
      <!-- Header -->
      <div class="login-header">
        <span class="login-logo-icon">O</span>
        <h1 class="login-title">Ooop Admin</h1>
        <p class="login-subtitle">Ooop管理平台</p>
      </div>

      <!-- Form -->
      <div class="login-form">
        <div class="form-group">
          <label class="form-label" for="login-username">账号</label>
          <input
            id="login-username"
            v-model="form.username"
            class="form-input"
            type="text"
            placeholder="请输入账号"
            autocomplete="username"
            @keyup.enter="handleLogin"
          >
        </div>
        <div class="form-group">
          <label class="form-label" for="login-password">密码</label>
          <input
            id="login-password"
            v-model="form.password"
            class="form-input"
            type="password"
            placeholder="请输入密码"
            autocomplete="current-password"
            @keyup.enter="handleLogin"
          >
        </div>
        <button
          class="login-btn"
          :class="{ loading }"
          :disabled="loading"
          @click="handleLogin"
        >
          <span v-if="!loading">登 录</span>
          <span v-else class="btn-loading">
            <svg class="spin-icon" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" width="16" height="16" aria-hidden="true">
              <path d="M12 2v4M12 18v4M4.93 4.93l2.83 2.83M16.24 16.24l2.83 2.83M2 12h4M18 12h4M4.93 19.07l2.83-2.83M16.24 7.76l2.83-2.83"/>
            </svg>
            登录中
          </span>
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
}

/* ── Nebula background (same pattern as layout/index.vue) ── */
.bg-scene {
  position: fixed;
  inset: 0;
  pointer-events: none;
  z-index: 0;
}

.cloud {
  position: absolute;
  transition: background 0.6s ease;
  animation: cloud-drift ease-in-out infinite;
}

.cp1 {
  width: 900px; height: 240px;
  top: -100px; left: -160px;
  background: radial-gradient(ellipse at 40% 50%, var(--color-primary) 0%, transparent 68%);
  opacity: var(--cloud-nebula-op, 0.24);
  filter: blur(90px);
  border-radius: 60% 40% 55% 45% / 70% 50% 80% 40%;
  animation-duration: 40s; animation-delay: 0s;
}

.cp2 {
  width: 700px; height: 180px;
  bottom: 60px; right: -80px;
  background: radial-gradient(ellipse at 55% 50%, var(--color-primary-light) 0%, transparent 70%);
  opacity: var(--cloud-nebula-op, 0.24);
  filter: blur(80px);
  border-radius: 45% 55% 40% 60% / 55% 65% 45% 70%;
  animation-duration: 33s; animation-delay: -12s;
}

.cp3 {
  width: 500px; height: 160px;
  top: 48%; left: 32%;
  background: radial-gradient(ellipse at 50% 60%, var(--color-primary) 0%, transparent 72%);
  opacity: var(--cloud-nebula-op, 0.24);
  filter: blur(75px);
  border-radius: 50% 65% 42% 58% / 60% 42% 68% 50%;
  animation-duration: 46s; animation-delay: -24s;
}

.cw1 {
  width: 680px; height: 150px;
  top: 100px; left: 80px;
  background: radial-gradient(ellipse at 45% 50%, var(--cloud-wisp-color, rgba(255,255,255,0.09)) 0%, transparent 65%);
  filter: blur(60px);
  border-radius: 62% 38% 58% 42% / 52% 68% 42% 62%;
  animation-duration: 30s; animation-delay: -6s;
}

.cw2 {
  width: 500px; height: 120px;
  top: 220px; right: 180px;
  background: radial-gradient(ellipse at 50% 45%, var(--cloud-wisp-color, rgba(255,255,255,0.09)) 0%, transparent 65%);
  filter: blur(52px);
  border-radius: 48% 52% 60% 40% / 65% 38% 58% 48%;
  animation-duration: 26s; animation-delay: -18s;
}

@keyframes cloud-drift {
  0%, 100% { transform: translate(0px, 0px) scale(1); }
  20%       { transform: translate(18px, -9px) scale(1.01); }
  40%       { transform: translate(34px, 6px) scale(0.99); }
  60%       { transform: translate(22px, -13px) scale(1.02); }
  80%       { transform: translate(6px, 10px) scale(0.98); }
}

/* ── Login card ── */
.login-card {
  position: relative;
  z-index: 1;
  width: 360px;
  padding: 40px 36px 36px;
}

.login-header {
  text-align: center;
  margin-bottom: 32px;
}

.login-logo-icon {
  font-size: 40px;
  line-height: 1;
  margin-bottom: 12px;
  display: block;
  filter: drop-shadow(0 0 16px var(--color-glow));
}

.login-title {
  font-family: 'Raleway', sans-serif;
  font-size: 20px;
  font-weight: 700;
  color: var(--text-primary);
  margin-bottom: 4px;
}

.login-subtitle {
  font-size: 12px;
  color: var(--text-muted);
}

/* ── Form ── */
.login-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-label {
  font-size: 12px;
  font-weight: 500;
  color: var(--text-secondary);
  letter-spacing: 0.3px;
}

.form-input {
  width: 100%;
  height: 42px;
  padding: 0 14px;
  background: var(--glass-bg);
  border: 1px solid var(--glass-border);
  border-radius: 10px;
  font-size: 14px;
  color: var(--text-primary);
  outline: none;
  transition: border-color 0.2s, box-shadow 0.2s;
  font-family: inherit;
}

.form-input::placeholder { color: var(--text-muted); }

.form-input:focus {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px var(--color-glow);
}

/* ── Login button ── */
.login-btn {
  width: 100%;
  height: 44px;
  margin-top: 4px;
  background: linear-gradient(135deg, var(--color-primary), var(--el-color-primary-dark-2));
  border: none;
  border-radius: 10px;
  font-size: 15px;
  font-weight: 600;
  color: #fff;
  cursor: pointer;
  transition: opacity 0.2s, box-shadow 0.2s;
  box-shadow: 0 4px 20px var(--color-glow);
  font-family: inherit;
}

.login-btn:hover:not(:disabled) {
  opacity: 0.9;
  box-shadow: 0 6px 28px var(--color-glow);
}

.login-btn:disabled {
  cursor: not-allowed;
  opacity: 0.7;
}

.btn-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.spin-icon { animation: spin 0.8s linear infinite; }

@keyframes spin {
  from { transform: rotate(0deg); }
  to   { transform: rotate(360deg); }
}
</style>
