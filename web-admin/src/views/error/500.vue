<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()
</script>

<template>
  <div class="error-page">
    <div class="error-orb" aria-hidden="true" />
    <div class="error-content">
      <div class="error-code">500</div>
      <p class="error-title">服务器开了个小差</p>
      <p class="error-desc">服务器内部错误，请稍后重试。如果问题持续存在，请联系管理员。</p>
      <div class="error-actions">
        <button class="btn-primary" @click="router.push('/')">返回首页</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.error-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  overflow: hidden;
}

.error-orb {
  position: fixed;
  top: -80px; left: -80px;
  width: 500px; height: 320px;
  background: radial-gradient(ellipse at 40% 50%, #EF4444 0%, transparent 65%);
  opacity: 0.18;
  filter: blur(90px);
  border-radius: 50%;
  pointer-events: none;
  z-index: 0;
}

.error-content {
  position: relative;
  z-index: 1;
  text-align: center;
}

.error-code {
  font-family: 'Raleway', sans-serif;
  font-size: clamp(96px, 18vw, 160px);
  font-weight: 900;
  line-height: 1;
  background: linear-gradient(135deg, #EF4444 0%, #F97316 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  margin-bottom: 16px;
  user-select: none;
  filter: drop-shadow(0 0 40px rgba(239,68,68,0.4));
}

.error-title {
  font-size: 20px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 8px;
}

.error-desc {
  font-size: 14px;
  color: var(--text-muted);
  margin-bottom: 32px;
  max-width: 360px;
}

.error-actions {
  display: flex;
  gap: 12px;
  justify-content: center;
}

.btn-primary {
  height: 40px;
  padding: 0 24px;
  background: linear-gradient(135deg, #EF4444, #F97316);
  border: none;
  border-radius: 10px;
  font-size: 14px;
  font-weight: 600;
  color: #fff;
  cursor: pointer;
  box-shadow: 0 4px 20px rgba(239,68,68,0.35);
  transition: opacity 0.2s;
  font-family: inherit;
}

.btn-primary:hover { opacity: 0.88; }
</style>
