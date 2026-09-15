<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { getActivityIntents, type ActivityIntent } from '@/api/modules/activityIntent'
import { getCategoryList, type AdminCategory } from '@/api/modules/activity'
import { formatDateTime } from '@/utils/date'

const city = ref('')
const page = ref(1)
const loading = ref(false)
const rows = ref<ActivityIntent[]>([])
const categories = ref<AdminCategory[]>([])
const categoryLabel = (ids: number[]) => ids.map(id => categories.value.find(item => item.id === id)?.label || String(id)).join('、')
async function load(nextPage = 1) {
  if (loading.value) return
  loading.value = true
  try {
    const result = await getActivityIntents({ city: city.value.trim() || undefined, page: nextPage })
    rows.value = result.data
    page.value = nextPage
  } finally { loading.value = false }
}
onMounted(async () => {
  const result = await getCategoryList()
  categories.value = result.data
  await load()
})
</script>

<template>
  <div class="intent-page">
    <el-card>
      <el-alert title="活动意向不等于报名。官方发布通过审核后，仅通知同城、同类别且同意接收通知的有效登记用户。" type="info" :closable="false" />
      <el-form inline class="filters" @submit.prevent="load()">
        <el-form-item label="城市"><el-input v-model="city" placeholder="如：杭州市" clearable /></el-form-item>
        <el-form-item><el-button type="primary" :loading="loading" @click="load()">查询</el-button></el-form-item>
      </el-form>
      <el-table :data="rows" v-loading="loading">
        <el-table-column prop="userId" label="用户 ID" width="100" />
        <el-table-column prop="city" label="城市" width="120" />
        <el-table-column prop="locationText" label="地图选点地址" min-width="200" />
        <el-table-column label="活动类别" min-width="140"><template #default="{ row }">{{ categoryLabel(row.categoryIds) }}</template></el-table-column>
        <el-table-column label="时段" min-width="150"><template #default="{ row }">{{ row.timeSlots.join('、') }}</template></el-table-column>
        <el-table-column prop="note" label="补充说明" min-width="180" />
        <el-table-column label="通知意愿" width="100"><template #default="{ row }">{{ row.notifyEnabled ? '接收' : '不接收' }}</template></el-table-column>
        <el-table-column label="更新时间" width="180"><template #default="{ row }">{{ formatDateTime(row.updatedAt) }}</template></el-table-column>
      </el-table>
      <div class="pagination">
        <el-button :disabled="loading || page === 1" @click="load(page - 1)">上一页</el-button>
        <span>第 {{ page }} 页</span>
        <el-button :disabled="loading || rows.length < 50" @click="load(page + 1)">下一页</el-button>
      </div>
    </el-card>
  </div>
</template>
<style scoped>
.intent-page { padding: 20px; }
.filters { margin-top: 20px; }
.pagination { display: flex; gap: 16px; align-items: center; justify-content: flex-end; margin-top: 20px; }
</style>
