<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search } from '@element-plus/icons-vue'
import { getCategoryList, deleteCategory, type AdminCategory } from '@/api/modules/activity'
import CategoryForm from '@/components/activity/categoryForm.vue'

defineOptions({ name: 'activityCategory' })

const loading = ref(false)
const list = ref<AdminCategory[]>([])
const formVisible = ref(false)
const editing = ref<AdminCategory | null>(null)

// 分类数据量小，按名称/标识做本地过滤，输入即时生效。
const keyword = ref('')
const filteredList = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  if (!kw) {
    return list.value
  }
  return list.value.filter(
    item => item.label.toLowerCase().includes(kw) || item.id.toLowerCase().includes(kw),
  )
})

async function getList() {
  loading.value = true
  try {
    const res = await getCategoryList()
    list.value = res.data
  } finally {
    loading.value = false
  }
}

function handleCreate() {
  editing.value = null
  formVisible.value = true
}

function handleEdit(row: AdminCategory) {
  editing.value = row
  formVisible.value = true
}

async function handleDelete(row: AdminCategory) {
  try {
    await ElMessageBox.confirm(`确认删除分类「${row.label}」?`, '提示', { type: 'warning' })
  } catch {
    return
  }
  try {
    await deleteCategory(row.id)
    ElMessage.success('已删除')
    getList()
  } catch {
    // 错误信息已由请求拦截器统一提示
  }
}

onMounted(getList)
</script>

<template>
  <div>
    <el-card>
      <div class="toolbar">
        <el-input
          v-model="keyword"
          clearable
          placeholder="搜索分类名称"
          :prefix-icon="Search"
          style="width: 220px; margin-right: 12px"
        />
        <el-button type="primary" @click="handleCreate">新增分类</el-button>
      </div>
      <el-table v-loading="loading" :data="filteredList" stripe border>
        <el-table-column prop="id" label="标识" min-width="140" />
        <el-table-column prop="label" label="名称" min-width="140" />
        <el-table-column label="图标" width="90">
          <template #default="{ row }">
            <el-image
              v-if="row.icon"
              :src="row.icon"
              fit="cover"
              :preview-src-list="[row.icon]"
              preview-teleported
              style="width: 40px; height: 40px; border-radius: 8px; cursor: pointer"
            />
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column prop="sort" label="排序" width="100" />
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'">
              {{ row.status === 1 ? '启用' : '停用' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="160" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="handleEdit(row)">编辑</el-button>
            <el-button type="danger" link @click="handleDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <CategoryForm v-model:visible="formVisible" :category="editing" @success="getList" />
  </div>
</template>

<style scoped>
.toolbar {
  margin-bottom: 12px;
}
</style>
