<script setup lang="ts">
import { computed, ref, watch, onMounted, onUnmounted } from 'vue'
import { useTheme, THEMES, setCustomPrimary } from '@/stores/theme'
import { useOooPUserStore } from '@/stores/user'
import { Search, SwitchButton, Monitor, Moon, Sunny,ArrowDown } from '@element-plus/icons-vue'
import SearchModal from './SearchModal.vue'

const { state, currentPreset, setPreference } = useTheme()
const userStore = useOooPUserStore()

const pickerColor = ref(currentPreset.value.color)
const avatarLoadFailed = ref(false)
const searchModalVisible = ref(false)

watch(() => currentPreset.value.color, (val) => { pickerColor.value = val })

const MODE_OPTIONS = [
  { value: 'light', label: '明亮', icon: Sunny },
  { value: 'dark', label: '暗黑', icon: Moon },
  { value: 'system', label: '跟随系统', icon: Monitor },
] as const

const currentModeOption = computed(
  () => MODE_OPTIONS.find((item) => item.value === state.preference) ?? MODE_OPTIONS[1],
)

function handleModeCommand(command: 'light' | 'dark' | 'system') {
  setPreference(command)
}

function onColorChange(color: string | null) {
  if (color) setCustomPrimary(color)
}

function handleCommand(command: string) {
  switch (command) {
    case 'logout':
      userStore.logout()
      break
  }
}

const presetColors = THEMES.map(t => t.color)

// 统一从用户仓库里取展示名称，避免 header 内再写死用户名
const displayName = computed(() => userStore.userInfo?.name?.trim() || 'Panda')
const avatarText = computed(() => displayName.value.slice(0, 1).toUpperCase())
const avatarSrc = computed(() => {
  const value = (userStore.userInfo as { avatar?: string } | undefined)?.avatar
  if (!value || avatarLoadFailed.value) {
    return ''
  }
  return value
})

const handleAvatarError = () => {
  avatarLoadFailed.value = true
  return false
}

// Ctrl+K 快捷键打开搜索
const handleKeydown = (e: KeyboardEvent) => {
  if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
    e.preventDefault()
    searchModalVisible.value = true
  }
}

onMounted(() => window.addEventListener('keydown', handleKeydown))
onUnmounted(() => window.removeEventListener('keydown', handleKeydown))
</script>

<template>
  <div class="layout-header">
    <div class="header-search">
      <button class="search-trigger" type="button" @click="searchModalVisible = true">
        <el-icon class="search-trigger-icon"><Search /></el-icon>
        <span class="search-trigger-text">搜索菜单名称</span>
        <kbd class="search-trigger-kbd">Ctrl K</kbd>
      </button>
      <SearchModal v-model:visible="searchModalVisible" />
    </div>

    <!-- Right: controls -->
    <div class="header-right">
      <!-- Element Plus color picker -->
      <div class="color-picker-wrap" aria-label="主题颜色选择器">
        <el-color-picker
          v-model="pickerColor"
          size="small"
          :predefine="presetColors"
          color-format="hex"
          @change="onColorChange"
        />
      </div>

      <div class="header-divider" aria-hidden="true" />

      <!-- Theme mode selector：支持明亮 / 暗黑 / 跟随系统 -->
      <el-dropdown trigger="click" @command="handleModeCommand">
        <button
          class="mode-toggle"
          :aria-label="`主题模式：${currentModeOption.label}`"
          :title="currentModeOption.label"
          type="button"
        >
          <el-icon :size="16" aria-hidden="true">
            <component :is="currentModeOption.icon" />
          </el-icon>
          <span class="mode-label">{{ currentModeOption.label }}</span>
        </button>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item
              v-for="item in MODE_OPTIONS"
              :key="item.value"
              :command="item.value"
              :icon="item.icon"
              :class="{ 'is-active': state.preference === item.value }"
            >
              {{ item.label }}
            </el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>

      <div class="header-divider" aria-hidden="true" />

      <!-- User dropdown (Element Plus) -->
      <el-dropdown trigger="click" @command="handleCommand">
        <div class="user-area">
          <el-avatar
            :src="avatarSrc || undefined"
            class="user-avatar"
            :style="{ background: `linear-gradient(135deg, ${currentPreset.color}, ${currentPreset.light})` }"
            @error="handleAvatarError"
          >
            {{ avatarText }}
          </el-avatar>
          <div class="user-info">
            <span class="user-name">{{ displayName }}</span>
          </div>
          <el-icon class="chevron"><ArrowDown /></el-icon>
        </div>
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item divided command="logout" :icon="SwitchButton">退出登录</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
    </div>
  </div>
</template>

<style scoped lang="css">
.layout-header {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 24px;
  background: var(--glass-card);
  backdrop-filter: blur(16px);
  /* -webkit-backdrop-filter: blur(16px); */
  /* border-bottom: 1px solid var(--glass-border); */
}

.header-search {
  width: 280px;
  min-width: 0;
}

/* 搜索触发按钮 */
.search-trigger {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 7px 12px;
  border-radius: 8px;
  border: 1px solid var(--glass-border);
  background: var(--glass-bg);
  color: var(--text-muted);
  cursor: pointer;
  font-size: 13px;
  transition: all 0.2s ease;
  outline: none;
  text-align: left;
}

.search-trigger:hover {
  background: var(--glass-hover);
  color: var(--text-secondary);
  border-color: var(--color-primary);
  box-shadow: 0 0 10px var(--color-glow);
}

.search-trigger:focus-visible {
  box-shadow: 0 0 0 2px var(--color-primary);
}

.search-trigger-icon {
  font-size: 14px;
  flex-shrink: 0;
}

.search-trigger-text {
  flex: 1;
  letter-spacing: 0.2px;
}

.search-trigger-kbd {
  display: inline-flex;
  align-items: center;
  padding: 2px 6px;
  border-radius: 5px;
  border: 1px solid var(--glass-border);
  background: var(--glass-card);
  font-size: 11px;
  font-family: inherit;
  color: var(--text-muted);
  line-height: 1.5;
  flex-shrink: 0;
}

/* Right */
.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

/* Color picker wrapper */
.color-picker-wrap {
  display: flex;
  align-items: center;
}

.color-picker-wrap :deep(.el-color-picker__trigger) {
  border: 1px solid var(--glass-border) !important;
  background: var(--glass-bg) !important;
  border-radius: 8px !important;
  padding: 2px !important;
  transition: all 0.2s ease;
}

.color-picker-wrap :deep(.el-color-picker__trigger):hover {
  background: var(--glass-hover) !important;
  box-shadow: 0 0 10px var(--color-glow) !important;
}

.color-picker-wrap :deep(.el-color-picker__color) {
  border-radius: 5px !important;
  border: none !important;
}

/* Divider */
.header-divider {
  width: 1px;
  height: 20px;
  background: var(--divider);
  flex-shrink: 0;
}

/* Dark/Light toggle */
.mode-toggle {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  border-radius: 20px;
  border: 1px solid var(--glass-border);
  background: var(--glass-bg);
  color: var(--text-secondary);
  cursor: pointer;
  font-size: 12.5px;
  font-weight: 500;
  transition: all 0.2s ease;
  white-space: nowrap;
  outline: none;
}

.mode-toggle:hover {
  background: var(--glass-hover);
  color: var(--text-primary);
  box-shadow: 0 0 12px var(--color-glow);
}

.mode-toggle:focus-visible {
  box-shadow: 0 0 0 2px var(--color-primary);
}

.mode-label { letter-spacing: 0.3px; }

/* User area (el-dropdown trigger) */
.user-area {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 8px 4px 4px;
  border-radius: 24px;
  border: 1px solid var(--glass-border);
  background: var(--glass-bg);
  cursor: pointer;
  transition: all 0.2s ease;
  user-select: none;
  outline: none;
}

.user-area:hover {
  background: var(--glass-hover);
}

.user-avatar {
  width: 32px;
  height: 32px;
  flex-shrink: 0;
  font-size: 13px;
  font-weight: 700;
  color: #fff;
  box-shadow: 0 2px 8px var(--color-glow);
  transition: background 0.4s ease, box-shadow 0.4s ease;
}

.user-info {
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.user-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
  line-height: 1;
}


.chevron {
  color: var(--text-muted);
  font-size: 12px;
}

@media (max-width: 960px) {
  .layout-header {
    gap: 12px;
    padding: 0 16px;
  }

  .header-search {
    width: 180px;
  }

  .search-trigger-kbd {
    display: none;
  }
}
</style>
