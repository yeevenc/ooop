<script setup lang="ts">
import { computed } from 'vue'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import type { MenuNode } from '@/layout/menu.ts'

const props = defineProps<{
  node: MenuNode
}>()

// 路由里使用的是图标组件名字符串，这里统一映射成真正的 Element Plus 图标组件
const iconComponent = computed(() => {
  const iconName = props.node.icon

  if (!iconName) {
    return ElementPlusIconsVue.Files
  }

  return (ElementPlusIconsVue as Record<string, unknown>)[iconName] || ElementPlusIconsVue.Files
})
</script>

<template>
  <el-menu-item v-if="!node.children?.length" :index="node.fullPath">
      <el-icon>
        <component :is="iconComponent" />
      </el-icon>
    <template #title>{{ node.title }}</template>
  </el-menu-item>

  <el-sub-menu v-else :index="node.id">
    <template #title>
        <el-icon>
          <component :is="iconComponent" />
        </el-icon>
      <span>{{ node.title }}</span>
    </template>

    <SidebarMenuNode
      v-for="child in node.children"
      :key="child.id"
      :node="child"
    />
  </el-sub-menu>
</template>
