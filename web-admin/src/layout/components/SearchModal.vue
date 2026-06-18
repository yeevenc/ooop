<script setup lang="ts">
import { ref, computed, nextTick, watch, onMounted, onUnmounted } from 'vue'
import { useRouter, type RouteRecordRaw } from 'vue-router'
import { Search, Promotion } from '@element-plus/icons-vue'

interface MenuOption {
  label: string
  value: string
}

const props = defineProps<{ visible: boolean }>()
const emit = defineEmits<{ 'update:visible': [val: boolean] }>()

const router = useRouter()
const searchKeyword = ref('')
const activeIndex = ref(-1)
const inputRef = ref<HTMLInputElement>()

// 递归构建带父级路径的菜单选项（label 格式：父级 / 子级）
const buildOptions = (
  routes: RouteRecordRaw[],
  basePath = '',
  parentTitle = '',
): MenuOption[] => {
  return routes.reduce<MenuOption[]>((result, item) => {
    if (item.meta?.hidden) return result

    const fullPath = basePath
      ? `${basePath}/${item.path}`.replace(/\/+/g, '/')
      : item.path.startsWith('/')
        ? item.path
        : `/${item.path}`

    const title = item.meta?.title ? String(item.meta.title) : ''
    const label = parentTitle && title ? `${parentTitle} / ${title}` : title || parentTitle
    const children = (item.children || []).filter((c) => !c.meta?.hidden)

    if (children.length) {
      result.push(...buildOptions(children, fullPath, label))
      return result
    }

    if (title) result.push({ label, value: fullPath })
    return result
  }, [])
}

const allMenuOptions = computed(() =>
  buildOptions(router.options.routes as RouteRecordRaw[]),
)

// 模糊匹配：keyword 每个字符都在 label 中按顺序出现即命中
const fuzzyMatch = (text: string, kw: string): boolean => {
  let ti = 0
  const t = text.toLowerCase()
  const k = kw.toLowerCase()
  for (let i = 0; i < k.length; i++) {
    const idx = t.indexOf(k[i], ti)
    if (idx === -1) return false
    ti = idx + 1
  }
  return true
}

const filteredOptions = computed(() => {
  const kw = searchKeyword.value.trim()
  if (!kw) return allMenuOptions.value
  return allMenuOptions.value.filter((item) => fuzzyMatch(item.label, kw))
})

// 高亮匹配字符
const highlightLabel = (label: string, kw: string): string => {
  if (!kw.trim()) return label
  const k = kw.trim().toLowerCase()
  let result = ''
  let ti = 0
  for (let i = 0; i < label.length; i++) {
    if (ti < k.length && label[i].toLowerCase() === k[ti]) {
      result += `<mark>${label[i]}</mark>`
      ti++
    } else {
      result += label[i]
    }
  }
  return result
}

const close = () => {
  emit('update:visible', false)
}

const handleSelect = (path: string) => {
  if (!path) return
  router.push(path)
  close()
}

const moveSelection = (dir: 1 | -1) => {
  const len = filteredOptions.value.length
  if (!len) return
  activeIndex.value = (activeIndex.value + dir + len) % len
}

const confirmSelection = () => {
  const item = filteredOptions.value[activeIndex.value]
  if (item) handleSelect(item.value)
}

watch(searchKeyword, () => { activeIndex.value = -1 })

const handleKeydown = (e: KeyboardEvent) => {
  if (!props.visible) return
  if (e.key === 'Escape') { e.preventDefault(); close() }
}

onMounted(() => document.addEventListener('keydown', handleKeydown))
onUnmounted(() => document.removeEventListener('keydown', handleKeydown))

watch(
  () => props.visible,
  async (val) => {
    if (val) {
      searchKeyword.value = ''
      activeIndex.value = -1
      await nextTick()
      inputRef.value?.focus()
    }
  },
)
</script>

<template>
  <Teleport to="body">
    <Transition name="el-zoom-in-center">
      <div v-if="visible" class="search-overlay" @click.self="close">
        <Transition name="search-modal">
          <div v-if="visible" class="search-modal">
            <!-- 搜索输入区 -->
            <div class="search-input-wrap">
              <el-icon class="search-icon"><Search /></el-icon>
              <input
                ref="inputRef"
                v-model="searchKeyword"
                class="search-input"
                placeholder="输入菜单名称快速跳转..."
                autocomplete="off"
                @keydown.down.prevent="moveSelection(1)"
                @keydown.up.prevent="moveSelection(-1)"
                @keydown.enter.prevent="confirmSelection"
              />
              <kbd class="esc-hint" @click="close">ESC</kbd>
            </div>

            <!-- 搜索结果列表 -->
            <div v-if="filteredOptions.length" class="search-results">
              <div
                v-for="(item, index) in filteredOptions"
                :key="item.value"
                class="result-item"
                :class="{ 'is-active': index === activeIndex }"
                @click="handleSelect(item.value)"
                @mouseenter="activeIndex = index"
              >
                <el-icon class="result-icon"><Promotion /></el-icon>
                <!-- eslint-disable-next-line vue/no-v-html -->
                <span class="result-label" v-html="highlightLabel(item.label, searchKeyword)" />
                <el-icon class="result-enter"><Search /></el-icon>
              </div>
            </div>

            <!-- 无结果提示 -->
            <div v-else-if="searchKeyword.trim()" class="search-empty">
              没有找到 "<strong>{{ searchKeyword }}</strong>" 相关菜单
            </div>

            <!-- 底部快捷键提示 -->
            <div class="search-tips">
              <span class="tip-item"><kbd>↑</kbd><kbd>↓</kbd> 选择</span>
              <span class="tip-item"><kbd>Enter</kbd> 跳转</span>
              <span class="tip-item"><kbd>Esc</kbd> 关闭</span>
            </div>
          </div>
        </Transition>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped lang="css">
/* 遮罩 */
.search-overlay {
  position: fixed;
  inset: 0;
  z-index: 2000;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding-top: 16vh;
  background:
    radial-gradient(ellipse at 20% 20%, rgba(64, 120, 255, 0.14) 0%, transparent 55%),
    radial-gradient(ellipse at 80% 80%, rgba(140, 60, 255, 0.10) 0%, transparent 55%),
    rgba(8, 8, 20, 0.42);
  backdrop-filter: blur(12px) saturate(140%);
  /* -webkit-backdrop-filter: blur(12px) saturate(140%); */
}

/* 弹框主体 */
.search-modal {
   width: 50%;
  max-width: 860px;
  min-width: 420px;
  border-radius: 24px;
  overflow: hidden;
   /* 提高不透明度，确保亮色模式下背景足够白 */
  background:
    linear-gradient(
      140deg,
      rgba(255, 255, 255, 0.72) 0%,
      rgba(230, 238, 255, 0.55) 35%,
      rgba(220, 232, 255, 0.60) 65%,
      rgba(255, 255, 255, 0.68) 100%
    );
  background-size: 200% 200%;
  backdrop-filter: blur(48px) saturate(220%) brightness(1.06);
  /* -webkit-backdrop-filter: blur(48px) saturate(220%) brightness(1.06); */
  border: 1px solid rgba(255, 255, 255, 0.55);
  box-shadow:
    inset 0 1.5px 0 rgba(255, 255, 255, 0.90),
    inset 0 -1px 0 rgba(180, 200, 255, 0.15),
    inset 1px 0 0 rgba(255, 255, 255, 0.30),
    inset -1px 0 0 rgba(255, 255, 255, 0.15),
    0 0 0 1px rgba(255, 255, 255, 0.25),
    0 0 24px rgba(100, 140, 255, 0.18),
    0 0 60px rgba(120, 80, 255, 0.10),
    0 8px 24px rgba(0, 0, 0, 0.07),
    0 24px 64px rgba(0, 0, 0, 0.10);
  animation: liquid-shimmer 10s ease infinite;
}
/* 顶部玻璃高光线：模拟光线打在玻璃边缘 */
.search-modal::before {
  content: '';
  position: absolute;
  top: 0;
  left: 12%;
  right: 12%;
  height: 1px;
  background: linear-gradient(
    90deg,
    transparent 0%,
    rgba(255, 255, 255, 0.95) 30%,
    rgba(200, 220, 255, 0.80) 60%,
    rgba(255, 255, 255, 0.60) 80%,
    transparent 100%
  );
  z-index: 10;
  pointer-events: none;
}

/* 上半区玻璃光泽层 */
.search-modal::after {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 45%;
  background: linear-gradient(
    180deg,
    rgba(255, 255, 255, 0.14) 0%,
    rgba(220, 230, 255, 0.05) 70%,
    transparent 100%
  );
  border-radius: 24px 24px 0 0;
  z-index: 1;
  pointer-events: none;
}

/* 液态流动背景动画 */
@keyframes liquid-shimmer {
  0%   { background-position: 0% 0%; }
  25%  { background-position: 100% 0%; }
  50%  { background-position: 100% 100%; }
  75%  { background-position: 0% 100%; }
  100% { background-position: 0% 0%; }
}
/* 输入区 */
.search-input-wrap {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 20px;
  height: 60px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
}

.search-icon {
  font-size: 18px;
  color: var(--color-primary);
  flex-shrink: 0;
}

.search-input {
  flex: 1;
  border: none;
  outline: none;
  background: transparent;
  font-size: 16px;
  font-weight: 500;
  color: #323232;
  caret-color: var(--color-primary);
  letter-spacing: 0.01em;
}

.search-input::placeholder {
  color: rgba(70, 70, 70, 0.45);
  font-weight: 400;
}

.esc-hint {
  flex-shrink: 0;
  padding: 3px 8px;
  border: 1px solid rgba(255, 255, 255, 0.18);
  border-radius: 6px;
  font-size: 11px;
  font-family: inherit;
  font-weight: 600;
  color: var(--color-primary);
  background: rgba(255, 255, 255, 0.08);
  cursor: pointer;
  user-select: none;
  transition: background 0.15s, color 0.15s;
}

.esc-hint:hover {
  background: rgba(255, 255, 255, 0.14);
  color: #F1F5F9;
}

/* 搜索结果列表 */
.search-results {
  max-height: 340px;
  overflow-y: auto;
  padding: 6px 0;
}

.search-results::-webkit-scrollbar { width: 3px; }
.search-results::-webkit-scrollbar-track { background: transparent; }
.search-results::-webkit-scrollbar-thumb {
  background: rgba(139, 92, 246, 0.35);
  border-radius: 4px;
}

.result-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 11px 20px;
  cursor: pointer;
  color: rgba(79, 79, 79);
  transition: background 0.15s ease, color 0.15s ease;
}

.result-item:hover,
.result-item.is-active {
  background: linear-gradient(
    90deg,
    rgba(var(--el-color-primary-rgb), 0.18) 0%,
    rgba(var(--el-color-primary-rgb), 0.1) 100%
  );
  color: var(--color-primary);
}

.result-icon {
  font-size: 13px;
  flex-shrink: 0;
  opacity: 0.4;
  transition: opacity 0.15s;
}

.result-item:hover .result-icon,
.result-item.is-active .result-icon {
  opacity: 0.8;
}

.result-label {
  flex: 1;
  font-size: 14px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 高亮匹配字符 */
.result-label :deep(mark) {
  background: transparent;
  color: var(--color-primary-light, #A78BFA);
  font-weight: 700;
}

.result-enter {
  font-size: 11px;
  opacity: 0;
  flex-shrink: 0;
  transition: opacity 0.15s;
}

.result-item:hover .result-enter,
.result-item.is-active .result-enter {
  opacity: 0.5;
}

/* 无结果 */
.search-empty {
  padding: 36px 20px;
  text-align: center;
  font-size: 14px;
  color: rgba(148, 163, 184, 0.55);
}

.search-empty strong {
  color: rgba(167, 139, 250, 0.85);
  font-weight: 600;
}

/* 底部快捷键提示 */
.search-tips {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 9px 20px;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
  background: rgba(255, 255, 255, 0.02);
}

.tip-item {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  color: rgba(70, 70, 70, 0.8);
}

kbd {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 1px 5px;
  border-radius: 4px;
  border: 1px solid rgba(255, 255, 255, 0.12);
  background: rgba(255, 255, 255, 0.05);
  font-size: 10px;
  font-family: inherit;
  color: var(--color-primary);
  line-height: 1.6;
}

/* 遮罩动画 */
.search-overlay-enter-active,
.search-overlay-leave-active {
  transition: opacity 0.2s ease;
}
.search-overlay-enter-from,
.search-overlay-leave-to {
  opacity: 0;
}

/* 弹框动画 */
.search-modal-enter-active {
  transition:
    opacity 0.22s ease,
    transform 0.22s cubic-bezier(0.34, 1.56, 0.64, 1);
}
.search-modal-leave-active {
  transition:
    opacity 0.15s ease,
    transform 0.15s ease;
}
.search-modal-enter-from {
  opacity: 0;
  transform: scale(0.88) translateY(-12px);
}
.search-modal-leave-to {
  opacity: 0;
  transform: scale(0.94) translateY(-6px);
}
</style>
