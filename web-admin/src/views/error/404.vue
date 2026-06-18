<script setup lang="ts">
import { useRouter } from 'vue-router'

const router = useRouter()
</script>

<template>
  <div class="error-page">
    <div class="error-orb" aria-hidden="true" />
    <div class="error-content">
      <div class="error-code">404</div>
      <p class="error-title">页面未找到</p>
      <p class="error-desc">您访问的页面不存在，可能已被移除或链接有误。</p>
      <div class="error-actions">
        <button class="btn-primary" @click="router.push('/')">返回首页</button>
        <button class="btn-secondary" @click="router.back()">上一页</button>
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
  background: radial-gradient(ellipse at 40% 50%, var(--color-primary) 0%, transparent 65%);
  opacity: var(--cloud-nebula-op, 0.24);
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
  background: linear-gradient(135deg, var(--color-primary) 0%, #6366F1 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  margin-bottom: 16px;
  user-select: none;
  filter: drop-shadow(0 0 40px var(--color-glow));
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
  max-width: 340px;
}

.error-actions {
  display: flex;
  gap: 12px;
  justify-content: center;
}

.btn-primary {
  height: 40px;
  padding: 0 24px;
  background: linear-gradient(135deg, var(--color-primary), #6366F1);
  border: none;
  border-radius: 10px;
  font-size: 14px;
  font-weight: 600;
  color: #fff;
  cursor: pointer;
  box-shadow: 0 4px 20px var(--color-glow);
  transition: opacity 0.2s;
  font-family: inherit;
}

.btn-primary:hover { opacity: 0.88; }

.btn-secondary {
  height: 40px;
  padding: 0 24px;
  background: var(--glass-bg);
  border: 1px solid var(--glass-border);
  border-radius: 10px;
  font-size: 14px;
  font-weight: 500;
  color: var(--text-secondary);
  cursor: pointer;
  backdrop-filter: blur(8px);
  transition: background 0.2s, color 0.2s;
  font-family: inherit;
}

.btn-secondary:hover {
  background: var(--glass-hover);
  color: var(--text-primary);
}
</style>
