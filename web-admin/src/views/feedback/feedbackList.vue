<script setup lang="ts">
import { Search } from '@element-plus/icons-vue'
import { onMounted, reactive, ref } from 'vue'
import {
  getFeedbackList,
  type FeedbackItem,
  type FeedbackType,
} from '@/api/modules/feedback'
import { formatDateTime } from '@/utils/date'

defineOptions({ name: 'feedbackList' })

const TYPE_OPTIONS: Array<{ label: string; value: FeedbackType }> = [
  { label: '产品建议', value: 'product' },
  { label: '账号问题', value: 'account' },
  { label: '活动问题', value: 'activity' },
]

const typeMeta: Record<FeedbackType, { text: string; type: 'success' | 'warning' | 'info' }> = {
  product: { text: '产品建议', type: 'success' },
  account: { text: '账号问题', type: 'warning' },
  activity: { text: '活动问题', type: 'info' },
}

const loading = ref(false)
const tableData = ref<FeedbackItem[]>([])
const total = ref(0)
const queryForm = reactive({
  page: 1,
  pageSize: 10,
  keyword: '',
  type: '' as FeedbackType | '',
})

function getTypeMeta(type: FeedbackType) {
  return typeMeta[type] ?? { text: type, type: 'info' as const }
}

async function getList() {
  loading.value = true
  try {
    const res = await getFeedbackList({
      page: queryForm.page,
      page_size: queryForm.pageSize,
      keyword: queryForm.keyword || undefined,
      type: queryForm.type || undefined,
    })
    tableData.value = res.data.list
    total.value = res.data.total
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  queryForm.page = 1
  getList()
}

function handleSizeChange(size: number) {
  queryForm.pageSize = size
  queryForm.page = 1
  getList()
}

function handleCurrentChange(page: number) {
  queryForm.page = page
  getList()
}

onMounted(() => {
  getList()
})
</script>

<template>
  <div>
    <el-card shadow="never">
      <el-form :model="queryForm" inline>
        <el-form-item label="关键词">
          <el-input
            v-model="queryForm.keyword"
            clearable
            placeholder="内容 / 手机号 / 昵称"
            @clear="handleSearch"
            @keyup.enter="handleSearch"
          />
        </el-form-item>
        <el-form-item label="问题类型">
          <el-select
            v-model="queryForm.type"
            clearable
            placeholder="全部类型"
            style="width: 140px"
            @change="handleSearch"
            @clear="handleSearch"
          >
            <el-option
              v-for="item in TYPE_OPTIONS"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :icon="Search" @click="handleSearch">搜索</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card class="m-t-10">
      <el-table
        v-loading="loading"
        :data="tableData"
        stripe
        border
        style="height: calc(100vh - 310px);"
      >
        <el-table-column prop="id" label="ID" width="90" fixed="left" />
        <el-table-column label="问题类型" width="110">
          <template #default="{ row }">
            <el-tag :type="getTypeMeta(row.type).type">
              {{ getTypeMeta(row.type).text }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="content" label="反馈内容" min-width="260" show-overflow-tooltip />
        <el-table-column label="截图" min-width="160">
          <template #default="{ row }">
            <div v-if="row.imageUrls?.length" class="image-list">
              <el-image
                v-for="url in row.imageUrls"
                :key="url"
                :src="url"
                :preview-src-list="row.imageUrls"
                fit="cover"
                preview-teleported
                class="feedback-image"
              />
            </div>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="用户" min-width="170">
          <template #default="{ row }">
            <div class="user-cell">
              <span>{{ row.userNickname || '-' }}</span>
              <span class="muted">{{ row.userPhone || '-' }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="userId" label="用户 ID" width="100" />
        <el-table-column prop="devicePlatform" label="平台" width="100">
          <template #default="{ row }">{{ row.devicePlatform || '-' }}</template>
        </el-table-column>
        <el-table-column prop="deviceVersion" label="系统版本" min-width="120">
          <template #default="{ row }">{{ row.deviceVersion || '-' }}</template>
        </el-table-column>
        <el-table-column prop="appVersion" label="App 版本" width="110">
          <template #default="{ row }">{{ row.appVersion || '-' }}</template>
        </el-table-column>
        <el-table-column label="提交时间" min-width="170">
          <template #default="{ row }">{{ formatDateTime(row.createdAt) }}</template>
        </el-table-column>
      </el-table>

      <el-pagination
        class="m-t-10"
        background
        layout="total, sizes, prev, pager, next, jumper"
        :current-page="queryForm.page"
        :page-size="queryForm.pageSize"
        :page-sizes="[10, 20, 50]"
        :total="total"
        @size-change="handleSizeChange"
        @current-change="handleCurrentChange"
      />
    </el-card>
  </div>
</template>

<style scoped>
.image-list {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.feedback-image {
  width: 42px;
  height: 42px;
  border-radius: 6px;
  cursor: pointer;
}

.user-cell {
  display: flex;
  flex-direction: column;
  line-height: 20px;
}

.muted {
  color: #909399;
  font-size: 12px;
}
</style>
