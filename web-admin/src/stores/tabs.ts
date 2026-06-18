import { ref, computed } from 'vue'
import type { RouteLocationNormalizedLoaded } from 'vue-router'
// Direct import for programmatic navigation after tab removal.
// No circular dep: router does not import this store.
import router from '@/router'

export interface TabItem {
  name: string
  title: string
  path: string
  closable: boolean
}

const DASHBOARD_TAB: TabItem = {
  name: 'home',
  title: '首页',
  path: '/',
  closable: false,
}

const tabs = ref<TabItem[]>([DASHBOARD_TAB])
const activeTabName = ref<string>(DASHBOARD_TAB.name)
const cachedViews = computed(() => tabs.value.map(t => t.name))

export function useTabs() {

  function addTab(route: RouteLocationNormalizedLoaded): void {
    const name = String(route.name ?? '')
    if (!name || route.meta?.hidden) return
    const existing = tabs.value.find(t => t.name === name)
    if (existing) {
      existing.path = route.fullPath
    } else {
      tabs.value.push({
        name,
        title: String(route.meta?.title ?? name),
        path: route.fullPath,
        closable: name !== DASHBOARD_TAB.name,
      })
    }
    activeTabName.value = name
  }

  function removeTab(name: string): void {
    const idx = tabs.value.findIndex(t => t.name === name)
    if (idx === -1) return
    if (!tabs.value[idx].closable) return
    tabs.value.splice(idx, 1)
    if (activeTabName.value === name) {
      const nextTab = tabs.value[idx] ?? tabs.value[idx - 1]
      if (nextTab) {
        activeTabName.value = nextTab.name
        router.push(nextTab.path)
      }
    }
  }

  function closeOtherTabs(name: string): void {
    tabs.value = tabs.value.filter(t => !t.closable || t.name === name)
    if (!tabs.value.find(t => t.name === activeTabName.value)) {
      const target = tabs.value.find(t => t.name === name) ?? tabs.value[0]
      activeTabName.value = target.name
      router.push(target.path)
    }
  }

  function closeAllTabs(): void {
    tabs.value = tabs.value.filter(t => !t.closable)
    activeTabName.value = DASHBOARD_TAB.name
    router.push(DASHBOARD_TAB.path)
  }

  // Called by TagBar on click for immediate visual feedback before route watcher fires
  function setActive(name: string): void {
    activeTabName.value = name
  }

  return { tabs, activeTabName, cachedViews, addTab, removeTab, closeOtherTabs, closeAllTabs, setActive }
}
