<script setup lang="ts">
import { Search } from '@element-plus/icons-vue'
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  getActivityList,
  getCategoryList,
  deleteActivity,
  reviewActivity,
  setActivityStatus,
  type AdminActivity,
  type AdminCategory,
  type PushNotificationResult,
} from '@/api/modules/activity'
import { formatDateTime } from '@/utils/date'
import ActivityEditForm from '@/components/activity/activityEditForm.vue'

defineOptions({ name: 'activityList' })

const STATUS_OPTIONS = [
  { label: '待审核', value: 'pending' },
  { label: '进行中', value: 'ongoing' },
  { label: '已拒绝', value: 'rejected' },
  { label: '已下架', value: 'taken_down' },
  { label: '已取消', value: 'cancelled' },
]

const statusMeta: Record<string, { text: string; type: 'warning' | 'success' | 'danger' | 'info' }> = {
  pending: { text: '待审核', type: 'warning' },
  ongoing: { text: '进行中', type: 'success' },
  rejected: { text: '已拒绝', type: 'danger' },
  taken_down: { text: '已下架', type: 'info' },
  cancelled: { text: '已取消', type: 'info' },
}

const loading = ref(false)
const tableData = ref<AdminActivity[]>([])
const total = ref(0)
const categories = ref<AdminCategory[]>([])
const queryForm = reactive({
  page: 1,
  pageSize: 10,
  keyword: '',
  status: '',
  category_id: '',
})
const editVisible = ref(false)
const editId = ref<string | null>(null)

function getStatusMeta(status: string) {
  return statusMeta[status] ?? { text: status, type: 'info' as const }
}

async function getList() {
  loading.value = true
  try {
    const res = await getActivityList({
      page: queryForm.page,
      page_size: queryForm.pageSize,
      keyword: queryForm.keyword || undefined,
      status: queryForm.status || undefined,
      category_id: queryForm.category_id || undefined,
    })
    tableData.value = res.data.list
    total.value = res.data.total
  } finally {
    loading.value = false
  }
}

async function loadCategories() {
  const res = await getCategoryList()
  categories.value = res.data
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

// 统一二次确认 + 执行 + 刷新；取消时静默返回。
async function withConfirm(message: string, run: () => Promise<void>) {
  try {
    await ElMessageBox.confirm(message, '提示', { type: 'warning' })
  } catch {
    return
  }
  try {
    await run()
    getList()
  } catch {
    // 错误信息已由请求拦截器统一提示
  }
}

function showReviewNotificationResult(result: PushNotificationResult | undefined, actionText: string) {
  if (!result) {
    ElMessage.warning(`已${actionText}，但未返回 Push 结果`)
    return
  }

  if (result.success) {
    ElMessage.success(`已${actionText}，Push 已发送`)
    return
  }

  const details = [
    `已${actionText}，但 Push 发送失败`,
    `目标别名：${result.alias || '-'}`,
    `触发状态：${result.triggered ? '已触发' : '未触发'}`,
    `错误信息：${result.message || '-'}`,
  ]

  if (result.response) {
    details.push(`极光返回：${result.response}`)
  }

  ElMessageBox.alert(details.join('\n'), 'Push 发送结果', {
    type: 'warning',
    confirmButtonText: '知道了',
  })
}

function handleReview(row: AdminActivity, action: 'approve' | 'reject') {
  const text = action === 'approve' ? '通过' : '拒绝'
  withConfirm(`确认${text}活动「${row.title}」?`, async () => {
    const response = await reviewActivity(row.id, action)
    showReviewNotificationResult(response.data.notification, text)
  })
}

function handleStatus(row: AdminActivity, status: 'taken_down' | 'ongoing') {
  const text = status === 'taken_down' ? '下架' : '上架'
  withConfirm(`确认${text}活动「${row.title}」?`, async () => {
    await setActivityStatus(row.id, status)
    ElMessage.success(`已${text}`)
  })
}

function handleEdit(row: AdminActivity) {
  editId.value = row.id
  editVisible.value = true
}

function handleDelete(row: AdminActivity) {
  withConfirm(`确认删除活动「${row.title}」?删除后不可恢复。`, async () => {
    await deleteActivity(row.id)
    ElMessage.success('已删除')
  })
}

onMounted(() => {
  getList()
  loadCategories()
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
            placeholder="活动标题"
            @clear="handleSearch"
            @keyup.enter="handleSearch"
          />
        </el-form-item>
        <el-form-item label="状态">
          <el-select
            v-model="queryForm.status"
            clearable
            placeholder="全部状态"
            style="width: 140px"
            @change="handleSearch"
            @clear="handleSearch"
          >
            <el-option
              v-for="opt in STATUS_OPTIONS"
              :key="opt.value"
              :label="opt.label"
              :value="opt.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item label="分类">
          <el-select
            v-model="queryForm.category_id"
            clearable
            placeholder="全部分类"
            style="width: 140px"
            @change="handleSearch"
            @clear="handleSearch"
          >
            <el-option v-for="c in categories" :key="c.id" :label="c.label" :value="c.id" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :icon="Search" @click="handleSearch">搜索</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card class="m-t-10">
      <el-table v-loading="loading" :data="tableData" stripe border>
        <el-table-column prop="id" label="ID" width="80" fixed="left" />
        <el-table-column label="封面" width="80">
          <template #default="{ row }">
            <el-image
              v-if="row.imageUrl"
              :src="row.imageUrl"
              fit="cover"
              :preview-src-list="[row.imageUrl]"
              preview-teleported
              style="width: 44px; height: 44px; border-radius: 6px; cursor: pointer"
            />
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column prop="title" label="标题" min-width="180" show-overflow-tooltip />
        <el-table-column prop="categoryLabel" label="分类" width="100">
          <template #default="{ row }">{{ row.categoryLabel || '-' }}</template>
        </el-table-column>
        <el-table-column prop="city" label="城市" width="100">
          <template #default="{ row }">{{ row.city || '-' }}</template>
        </el-table-column>
        <el-table-column label="发起人" min-width="120">
          <template #default="{ row }">{{ row.organizer?.name || '-' }}</template>
        </el-table-column>
        <el-table-column label="人数" width="90">
          <template #default="{ row }">{{ row.currentCount }}/{{ row.totalCount }}</template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusMeta(row.status).type">{{ getStatusMeta(row.status).text }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" min-width="170">
          <template #default="{ row }">{{ formatDateTime(row.createdAt) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="240" fixed="right">
          <template #default="{ row }">
            <template v-if="row.status === 'pending'">
              <el-button type="success" link @click="handleReview(row, 'approve')">通过</el-button>
              <el-button type="danger" link @click="handleReview(row, 'reject')">拒绝</el-button>
            </template>
            <el-button
              v-if="row.status === 'ongoing'"
              type="warning"
              link
              @click="handleStatus(row, 'taken_down')"
            >
              下架
            </el-button>
            <el-button
              v-if="row.status === 'taken_down'"
              type="success"
              link
              @click="handleStatus(row, 'ongoing')"
            >
              上架
            </el-button>
            <el-button type="primary" link @click="handleEdit(row)">编辑</el-button>
            <el-button type="danger" link @click="handleDelete(row)">删除</el-button>
          </template>
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

    <ActivityEditForm v-model:visible="editVisible" :activity-id="editId" @success="getList" />
  </div>
</template>
