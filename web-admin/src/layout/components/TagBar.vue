<script setup lang="ts">
import { ref, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useTabs, type TabItem } from '@/stores/tabs'

const router = useRouter()
const { tabs, activeTabName, removeTab, closeOtherTabs, closeAllTabs, setActive } = useTabs()

const contextMenu = ref({
  visible: false,
  x: 0,
  y: 0,
  tabName: '',
})

function onTabClick(tab: TabItem) {
  if (tab.name !== activeTabName.value) {
    setActive(tab.name)
    router.push(tab.path)
  }
}

function onTabClose(tab: TabItem) {
  removeTab(tab.name)
}

function onContextMenu(e: MouseEvent, tab: TabItem) {
  e.preventDefault()
  contextMenu.value = {
    visible: true,
    x: e.clientX,
    y: e.clientY,
    tabName: tab.name,
  }

  nextTick(() => {
    document.addEventListener('click', closeContextMenu, { once: true })
  })
}

function closeContextMenu() {
  contextMenu.value.visible = false
}

function handleCloseThis() {
  removeTab(contextMenu.value.tabName)
  closeContextMenu()
}

function handleCloseOthers() {
  closeOtherTabs(contextMenu.value.tabName)
  closeContextMenu()
}

function handleCloseAll() {
  closeAllTabs()
  closeContextMenu()
}
</script>

<template>
  <nav class="tag-bar" aria-label="已打开的页面">
    <div class="tag-bar-inner" role="tablist">
      <el-tag
        v-for="tab in tabs"
        :key="tab.name"
        role="tab"
        :aria-selected="tab.name === activeTabName"
        :title="tab.title"
        :closable="tab.closable"
        :effect="tab.name === activeTabName ? 'dark' : 'plain'"
        :type="tab.name === activeTabName ? 'primary' : 'info'"
        round
        class="tab-tag"
        :class="{ 'is-active': tab.name === activeTabName }"
        @click="onTabClick(tab)"
        @close="onTabClose(tab)"
        @contextmenu="onContextMenu($event, tab)"
      >
        {{ tab.title }}
      </el-tag>
    </div>

    <!-- Right-click context menu -->
    <Teleport to="body">
      <Transition name="ctx-menu">
        <div
          v-if="contextMenu.visible"
          class="tab-context-menu"
          :style="{ left: contextMenu.x + 'px', top: contextMenu.y + 'px' }"
        >
          <div class="ctx-menu-item" @click="handleCloseThis">关闭当前</div>
          <div class="ctx-menu-item" @click="handleCloseOthers">关闭其他</div>
          <div class="ctx-menu-divider" />
          <div class="ctx-menu-item danger" @click="handleCloseAll">关闭所有</div>
        </div>
      </Transition>
    </Teleport>
  </nav>
</template>

<style scoped>
.tag-bar {
  height: 40px;
  display: flex;
  align-items: center;
  padding: 0 16px;
    background: var(--glass-card);
  backdrop-filter: blur(16px);
  /* -webkit-backdrop-filter: blur(16px); */
  border-bottom: 1px solid var(--glass-border);
  flex-shrink: 0;
  position: relative;
  z-index: 49;
}

.tag-bar-inner {
  display: flex;
  align-items: center;
  gap: 6px;
  overflow-x: auto;
  overflow-y: hidden;
  flex: 1;
  min-width: 0;
  scrollbar-width: none;
  -ms-overflow-style: none;
}

.tag-bar-inner::-webkit-scrollbar {
  display: none;
}

.tab-tag {
  cursor: pointer;
  flex-shrink: 0;
  font-size: 12.5px;
  transition: all 0.2s ease;
  user-select: none;
}

.tab-tag.is-active {
  box-shadow: 0 2px 10px var(--color-glow);
}

.tab-tag:not(.is-active):hover {
  opacity: 0.85;
}

/* ── Right-click context menu ── */
.tab-context-menu {
  position: fixed;
  z-index: 9999;
  min-width: 130px;
  background: var(--glass-bg);
  backdrop-filter: blur(24px) saturate(180%);
  -webkit-backdrop-filter: blur(24px) saturate(180%);
  border: 1px solid var(--glass-border);
  border-radius: 10px;
  padding: 5px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.25);
}

.ctx-menu-item {
  padding: 8px 14px;
  font-size: 13px;
  color: var(--text-secondary);
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.15s ease;
  white-space: nowrap;
}

.ctx-menu-item:hover {
  background: var(--glass-hover);
  color: var(--text-primary);
}

.ctx-menu-item.danger:hover {
  background: rgba(239, 68, 68, 0.12);
  color: #EF4444;
}

.ctx-menu-divider {
  height: 1px;
  margin: 4px 8px;
  background: var(--divider);
}

/* Context menu transition */
.ctx-menu-enter-active,
.ctx-menu-leave-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}

.ctx-menu-enter-from,
.ctx-menu-leave-to {
  opacity: 0;
  transform: scale(0.95);
}
</style>
