<script setup lang="ts">
import { computed } from 'vue'
import { useRouter, useRoute, type RouteRecordRaw } from 'vue-router'
import { useTheme } from '@/stores/theme'
import { buildMenuTree } from '@/layout/menu'
import SidebarMenuNode from '@/layout/components/SidebarMenuNode.vue'
const { state, toggleCollapse } = useTheme()
const router = useRouter()
const route = useRoute()
import type { MenuNode } from '@/layout/menu'

const menuTree = computed<MenuNode[]>(() => {
  return buildMenuTree(router.options.routes as readonly RouteRecordRaw[])
})

const defaultOpeneds = computed(() =>
  menuTree.value.flatMap((node) => {
    const hasActiveChild = node.children?.some((child) => route.path.startsWith(child.fullPath))
    return hasActiveChild ? [node.id] : []
  })
)

function onSelect(index: string) {
  router.push(index)
}
</script>

<template>
  <div class="sidebar-wrap">
    <!-- Logo -->
    <div class="sidebar-logo">
      <div class="logo-icon">
       <el-image src="../../../public/favicon.ico" style="border-radius: 6px;"></el-image>
      </div>
      <Transition name="fade-slide">
        <span v-if="!state.collapsed" class="logo-text">Ooop</span>
      </Transition>
    </div>

    <div class="sidebar-divider" />

    <!-- el-menu navigation -->
    <el-menu
      :default-active="route.path"
      :collapse="state.collapsed"
      :default-openeds="defaultOpeneds"
      class="sidebar-menu"
      @select="onSelect"
    >
      <SidebarMenuNode
        v-for="node in menuTree"
        :key="node.id"
        :node="node"
      />
    </el-menu>

    <!-- Collapse toggle -->
    <div class="sidebar-footer">
      <button
        class="collapse-btn"
        :aria-label="state.collapsed ? '展开菜单' : '收起菜单'"
        @click="toggleCollapse"
      >
        <svg viewBox="0 0 24 24" fill="currentColor" class="collapse-icon" :class="{ rotated: state.collapsed }" aria-hidden="true">
          <path d="M11.67 3.87L9.9 2.1 0 12l9.9 9.9 1.77-1.77L3.54 12z" />
        </svg>
        <Transition name="fade-slide">
          <span v-if="!state.collapsed" class="collapse-label">收起菜单</span>
        </Transition>
      </button>
    </div>
  </div>
</template>

<style scoped>
.sidebar-wrap {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--glass-card)!important;
  backdrop-filter: blur(40px) !important;
  /* -webkit-backdrop-filter: blur(16px) !important; */
   /* background: var(--glass-card);
    backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px); */
  border-right: 1px solid var(--glass-border);
  overflow: hidden;
}

/* Logo */
.sidebar-logo {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 14px;
  height: 64px;
  flex-shrink: 0;
}

.logo-icon {
  width: 34px;
  height: 34px;
  flex-shrink: 0;
  color: var(--color-primary);
  filter: drop-shadow(0 0 10px var(--color-glow));
  transition: filter 0.4s ease;
}

.logo-text {
  font-size: 15px;
  font-weight: 700;
  color: var(--text-primary);
  white-space: nowrap;
  letter-spacing: 0.4px;
}

.sidebar-divider {
  height: 1px;
  margin: 0 14px;
  background: var(--divider);
  flex-shrink: 0;
}

/* ── el-menu overrides ── */
.sidebar-menu {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 8px;
  box-sizing: border-box;
  --el-menu-bg-color: transparent;
  --el-menu-hover-bg-color: var(--glass-hover);
  --el-menu-active-color: #fff;
  --el-menu-text-color: var(--text-secondary);
  --el-menu-item-height: 44px;
  --el-menu-sub-item-height: 40px;
  --el-menu-item-font-size: 13.5px;
  --el-menu-icon-width: 20px;
}

.sidebar-menu::-webkit-scrollbar { width: 3px; }
.sidebar-menu::-webkit-scrollbar-track { background: transparent; }
.sidebar-menu::-webkit-scrollbar-thumb {
  background: var(--glass-border);
  border-radius: 2px;
}

.sidebar-menu.el-menu {
  border-right: none !important;
}

.sidebar-menu :deep(.el-menu-item),
.sidebar-menu :deep(.el-sub-menu__title) {
  border-radius: 10px !important;
  margin: 2px 0 !important;
  color: var(--text-secondary) !important;
  background-color: transparent !important;
  transition: background 0.2s ease, color 0.2s ease, box-shadow 0.2s ease !important;
}

.sidebar-menu :deep(.el-menu-item:hover),
.sidebar-menu :deep(.el-sub-menu__title:hover) {
  background-color: var(--glass-hover) !important;
  color: var(--text-primary) !important;
}

.sidebar-menu :deep(.el-menu-item.is-active) {
  background: linear-gradient(135deg, var(--color-primary) 0%, var(--color-primary-light) 100%) !important;
  color: #fff !important;
  box-shadow: 0 4px 18px var(--color-glow) !important;
}

.sidebar-menu :deep(.el-sub-menu.is-active > .el-sub-menu__title) {
  color: var(--color-primary-light) !important;
}

.sidebar-menu :deep(.el-sub-menu .el-menu) {
  background-color: transparent !important;
}

.menu-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  flex-shrink: 0;
  margin-right: 8px;
}

.sidebar-menu.el-menu--collapse :deep(.menu-icon) {
  margin-right: 0;
}

/* Footer */
.sidebar-footer {
  padding: 12px 8px;
  border-top: 1px solid var(--divider);
  flex-shrink: 0;
}

.collapse-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 8px 10px;
  border-radius: 8px;
  border: 1px solid var(--glass-border);
  background: var(--glass-bg);
  color: var(--text-muted);
  cursor: pointer;
  transition: all 0.2s ease;
  white-space: nowrap;
  outline: none;
}

.collapse-btn:hover {
  background: var(--glass-hover);
  color: var(--text-primary);
}

.collapse-btn:focus-visible {
  box-shadow: 0 0 0 2px var(--color-primary);
}

.collapse-icon {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
  transition: transform 0.32s ease;
}

.collapse-icon.rotated { transform: rotate(180deg); }

.collapse-label {
  font-size: 12.5px;
  font-weight: 500;
}

.fade-slide-enter-active,
.fade-slide-leave-active {
  transition: opacity 0.18s ease, transform 0.18s ease;
}

.fade-slide-enter-from,
.fade-slide-leave-to {
  opacity: 0;
  transform: translateX(-6px);
}
</style>
