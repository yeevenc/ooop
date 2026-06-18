<script setup lang="ts">
import { watch } from 'vue'
import { useRoute } from 'vue-router'
import { useTheme } from '@/stores/theme'
import { useTabs } from '@/stores/tabs'
import Sidebar from '@/layout/components/Sidebar.vue'
import Header from '@/layout/components/Header.vue'
import TagBar from '@/layout/components/TagBar.vue'

const { state } = useTheme()
const route = useRoute()
const { cachedViews, addTab } = useTabs()

watch(route, (newRoute) => {
  addTab(newRoute)
}, { immediate: true })
</script>

<template>
  <div class="admin-layout">
    <!-- 星空背景层 -->
    <div class="bg-scene" aria-hidden="true">
      <div class="stars stars-1" />
      <div class="stars stars-2" />
      <div class="stars stars-3" />
      <div class="aurora aurora-1" />
      <div class="aurora aurora-2" />
      <!-- Primary-color nebula clouds -->
      <div class="cloud cp1" />
      <div class="cloud cp2" />
      <div class="cloud cp3" />
      <!-- White/silver wisp clouds -->
      <div class="cloud cw1" />
      <div class="cloud cw2" />
      <div class="cloud cw3" />
      <div class="cloud cw4" />
    </div>

    <el-container class="layout-container">
      <el-aside
        :width="state.collapsed ? '64px' : '240px'"
        class="layout-aside"
      >
        <Sidebar />
      </el-aside>

      <el-container direction="vertical">
        <el-header height="64px" class="layout-header-wrap">
          <Header />
        </el-header>

        <TagBar />

        <el-main class="layout-main">
          <router-view v-slot="{ Component }">
            <transition name="el-fade-in-linear" mode="out-in">
              <keep-alive :include="cachedViews">
                <component :is="Component" />
              </keep-alive>
            </transition>
          </router-view>
        </el-main>
      </el-container>
    </el-container>
  </div>
</template>

<style scoped>
.admin-layout {
  width: 100%;
  height: 100vh;
  overflow: hidden;
  position: relative;
  background: var(--gradient-bg);
  background-attachment: fixed;
  transition: background 0.55s ease;
}

.bg-scene {
  position: fixed;
  inset: 0;
  pointer-events: none;
  z-index: 0;
}

.stars,
.aurora {
  position: absolute;
  inset: 0;
  transition: opacity 0.6s ease, background 0.6s ease;
}

.stars {
  background-repeat: repeat;
  animation: twinkle-drift linear infinite;
}

.stars-1 {
  opacity: var(--star-opacity-main, 0.85);
  background-image:
    radial-gradient(circle at 12% 20%, var(--star-color) 0 1px, transparent 1.6px),
    radial-gradient(circle at 34% 72%, var(--star-soft-color) 0 1.2px, transparent 1.8px),
    radial-gradient(circle at 58% 28%, var(--star-color) 0 1px, transparent 1.6px),
    radial-gradient(circle at 80% 18%, var(--star-soft-color) 0 1.3px, transparent 1.9px),
    radial-gradient(circle at 88% 66%, var(--star-color) 0 1px, transparent 1.6px),
    radial-gradient(circle at 22% 88%, var(--star-soft-color) 0 1.2px, transparent 1.8px);
  background-size: 360px 360px;
  animation-duration: 26s;
}

.stars-2 {
  opacity: var(--star-opacity-soft, 0.5);
  background-image:
    radial-gradient(circle at 18% 14%, var(--star-soft-color) 0 0.9px, transparent 1.5px),
    radial-gradient(circle at 42% 52%, var(--star-color) 0 0.8px, transparent 1.4px),
    radial-gradient(circle at 64% 82%, var(--star-soft-color) 0 1px, transparent 1.6px),
    radial-gradient(circle at 76% 34%, var(--star-color) 0 0.8px, transparent 1.4px),
    radial-gradient(circle at 8% 64%, var(--star-soft-color) 0 1px, transparent 1.6px);
  background-size: 520px 520px;
  animation-duration: 34s;
  animation-direction: reverse;
}

.stars-3 {
  opacity: calc(var(--star-opacity-soft, 0.5) * 0.7);
  background-image:
    radial-gradient(circle at 24% 24%, var(--star-color) 0 0.7px, transparent 1.3px),
    radial-gradient(circle at 48% 18%, var(--star-soft-color) 0 0.9px, transparent 1.5px),
    radial-gradient(circle at 68% 58%, var(--star-color) 0 0.7px, transparent 1.3px),
    radial-gradient(circle at 90% 40%, var(--star-soft-color) 0 0.9px, transparent 1.5px);
  background-size: 680px 680px;
  animation-duration: 42s;
}

.aurora {
  opacity: var(--aurora-opacity, 0.3);
  filter: blur(80px);
}

.aurora-1 {
  background:
    radial-gradient(ellipse 34% 20% at 18% 14%, var(--night-haze) 0%, transparent 72%);
}

.aurora-2 {
  background:
    radial-gradient(ellipse 26% 18% at 82% 22%, color-mix(in srgb, var(--night-haze) 70%, rgba(143, 211, 255, 0.14)) 0%, transparent 74%);
}

/* ── Cloud gradient background ── */
.cloud {
  position: absolute;
  transition: background 0.6s ease;
  animation: cloud-drift ease-in-out infinite;
}

/* Nebula clouds – primary color tint */
.cp1 {
  width: 1000px;
  height: 260px;
  top: -100px;
  left: -180px;
  background: radial-gradient(ellipse at 40% 50%, var(--color-primary) 0%, transparent 68%);
  opacity: var(--cloud-nebula-op, 0.24);
  filter: blur(100px);
  border-radius: 60% 40% 55% 45% / 70% 50% 80% 40%;
  animation-duration: 40s;
  animation-delay: 0s;
}

.cp2 {
  width: 780px;
  height: 200px;
  bottom: 60px;
  right: -100px;
  background: radial-gradient(ellipse at 55% 50%, var(--color-primary-light) 0%, transparent 70%);
  opacity: var(--cloud-nebula-op, 0.24);
  filter: blur(90px);
  border-radius: 45% 55% 40% 60% / 55% 65% 45% 70%;
  animation-duration: 33s;
  animation-delay: -12s;
}

.cp3 {
  width: 560px;
  height: 170px;
  top: -60px;
  right: 24%;
  background: radial-gradient(ellipse at 50% 60%, var(--color-primary) 0%, transparent 72%);
  opacity: var(--cloud-nebula-op, 0.24);
  filter: blur(80px);
  border-radius: 50% 65% 42% 58% / 60% 42% 68% 50%;
  animation-duration: 46s;
  animation-delay: -24s;
}

/* White / silver wisps – the actual "cloud" texture */
.cw1 {
  width: 720px;
  height: 160px;
  top: 100px;
  left: 80px;
  background: radial-gradient(ellipse at 45% 50%, var(--cloud-wisp-color, rgba(255,255,255,0.09)) 0%, transparent 65%);
  filter: blur(65px);
  border-radius: 62% 38% 58% 42% / 52% 68% 42% 62%;
  animation-duration: 30s;
  animation-delay: -6s;
}

.cw2 {
  width: 520px;
  height: 130px;
  top: 220px;
  right: 180px;
  background: radial-gradient(ellipse at 50% 45%, var(--cloud-wisp-color, rgba(255,255,255,0.09)) 0%, transparent 65%);
  filter: blur(55px);
  border-radius: 48% 52% 60% 40% / 65% 38% 58% 48%;
  animation-duration: 26s;
  animation-delay: -18s;
}

.cw3 {
  width: 660px;
  height: 150px;
  bottom: 180px;
  left: 120px;
  background: radial-gradient(ellipse at 42% 55%, var(--cloud-wisp-color, rgba(255,255,255,0.09)) 0%, transparent 68%);
  filter: blur(70px);
  border-radius: 55% 45% 50% 60% / 42% 62% 52% 70%;
  animation-duration: 38s;
  animation-delay: -9s;
}

.cw4 {
  width: 420px;
  height: 110px;
  bottom: 120px;
  right: 60px;
  background: radial-gradient(ellipse at 55% 50%, var(--cloud-wisp-color, rgba(255,255,255,0.09)) 0%, transparent 65%);
  filter: blur(50px);
  border-radius: 58% 42% 46% 54% / 50% 62% 45% 68%;
  animation-duration: 34s;
  animation-delay: -28s;
}

@keyframes cloud-drift {
  0%, 100% { transform: translate(0px,   0px)  scale(1); }
  20%       { transform: translate(18px,  -9px) scale(1.01); }
  40%       { transform: translate(34px,   6px) scale(0.99); }
  60%       { transform: translate(22px, -13px) scale(1.02); }
  80%       { transform: translate(6px,   10px) scale(0.98); }
}

@keyframes twinkle-drift {
  0%, 100% { transform: translate3d(0, 0, 0); opacity: calc(var(--star-opacity-main, 0.85) * 0.95); }
  25% { transform: translate3d(-6px, 4px, 0); opacity: var(--star-opacity-main, 0.85); }
  50% { transform: translate3d(4px, -3px, 0); opacity: calc(var(--star-opacity-main, 0.85) * 0.78); }
  75% { transform: translate3d(-3px, -5px, 0); opacity: calc(var(--star-opacity-main, 0.85) * 0.9); }
}

/* ── el-container layout ── */
.layout-container {
  position: relative;
  z-index: 1;
  height: 100vh;
}

.layout-aside {
  transition: width 0.32s cubic-bezier(0.4, 0, 0.2, 1) !important;
  overflow: hidden;
  height: 100%;
  flex-shrink: 0;
}

.layout-header-wrap {
  padding: 0 !important;
  flex-shrink: 0;
}

.layout-main {
  padding: 15px;
  overflow-y: auto !important;
  flex: 1;
  min-height: 0;
}
</style>
