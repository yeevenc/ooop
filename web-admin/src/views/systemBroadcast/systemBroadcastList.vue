<script setup lang="ts">
import { Bell, Search } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { onMounted, onUnmounted, reactive, ref } from 'vue'
import {
  createSystemBroadcast,
  getSystemBroadcastList,
  type SystemBroadcastItem,
  type SystemBroadcastStatus,
} from '@/api/modules/systemBroadcast'
import { formatDateTime } from '@/utils/date'

defineOptions({ name: 'systemBroadcastList' })

const STATUS_OPTIONS: Array<{ label: string; value: SystemBroadcastStatus }> = [
  { label: '排队中', value: 'pending' },
  { label: '发送中', value: 'running' },
  { label: '已完成', value: 'completed' },
  { label: '失败', value: 'failed' },
]

const statusMeta: Record<
  SystemBroadcastStatus,
  { text: string; type: 'warning' | 'success' | 'danger' | 'info' }
> = {
  pending: { text: '排队中', type: 'warning' },
  running: { text: '发送中', type: 'info' },
  completed: { text: '已完成', type: 'success' },
  failed: { text: '失败', type: 'danger' },
}

const loading = ref(false)
const submitting = ref(false)
const createVisible = ref(false)
const tableData = ref<SystemBroadcastItem[]>([])
const total = ref(0)
const queryForm = reactive({
  page: 1,
  pageSize: 10,
  status: '' as SystemBroadcastStatus | '',
})
const createForm = reactive({
  title: '',
  content: '',
})

let pollTimer: number | undefined

function getStatusMeta(status: string) {
  return (
    statusMeta[status as SystemBroadcastStatus] ?? {
      text: status,
      type: 'info' as const,
    }
  )
}

function hasActiveJob() {
  return tableData.value.some((item) => item.status === 'pending' || item.status === 'running')
}

function stopPolling() {
  if (pollTimer) {
    window.clearInterval(pollTimer)
    pollTimer = undefined
  }
}

function startPolling() {
  stopPolling()
  pollTimer = window.setInterval(() => {
    if (hasActiveJob()) {
      getList(true)
    }
  }, 3000)
}

async function getList(silent = false) {
  if (!silent) {
    loading.value = true
  }
  try {
    const res = await getSystemBroadcastList({
      page: queryForm.page,
      page_size: queryForm.pageSize,
      status: queryForm.status || undefined,
    })
    tableData.value = res.data.list
    total.value = res.data.total
  } finally {
    if (!silent) {
      loading.value = false
    }
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

function resetCreateForm() {
  createForm.title = ''
  createForm.content = ''
}

function handleOpenCreate() {
  resetCreateForm()
  createVisible.value = true
}

function handleCreateClose() {
  createVisible.value = false
  resetCreateForm()
}

async function handleCreateSubmit() {
  const title = createForm.title.trim()
  const content = createForm.content.trim()
  if (!title) {
    ElMessage.warning('请填写通知标题')
    return
  }
  if (!content) {
    ElMessage.warning('请填写通知内容')
    return
  }

  try {
    await ElMessageBox.confirm(
      '将向当前所有正常用户发送系统通知，并写入站内消息。封禁用户不会收到；关闭了通知权限的用户只会看到站内消息。',
      '确认发送',
      {
        type: 'warning',
        confirmButtonText: '确认发送',
        cancelButtonText: '取消',
      },
    )
  } catch {
    return
  }

  submitting.value = true
  try {
    await createSystemBroadcast({ title, content })
    ElMessage.success('已加入发送队列')
    createVisible.value = false
    resetCreateForm()
    queryForm.page = 1
    await getList()
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  getList().then(startPolling)
})

onUnmounted(() => {
  stopPolling()
})
</script>

<template>
  <div>
    <el-card shadow="never">
      <el-form :model="queryForm" inline>
        <el-form-item label="发送状态">
          <el-select
            v-model="queryForm.status"
            clearable
            placeholder="全部状态"
            style="width: 140px"
            @change="handleSearch"
            @clear="handleSearch"
          >
            <el-option
              v-for="item in STATUS_OPTIONS"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :icon="Search" @click="handleSearch">搜索</el-button>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" :icon="Bell" @click="handleOpenCreate">发送通知</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card class="m-t-10">
      <el-table
        v-loading="loading"
        :data="tableData"
        stripe
        border
        style="height: calc(100vh - 310px)"
      >
        <el-table-column prop="id" label="ID" width="80" fixed="left" />
        <el-table-column prop="title" label="标题" min-width="160" show-overflow-tooltip />
        <el-table-column prop="content" label="内容" min-width="260" show-overflow-tooltip />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusMeta(row.status).type">
              {{ getStatusMeta(row.status).text }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="目标人数" width="100">
          <template #default="{ row }">{{ row.targetCount }}</template>
        </el-table-column>
        <el-table-column label="站内信" width="110">
          <template #default="{ row }">{{ row.messageCount }}/{{ row.targetCount }}</template>
        </el-table-column>
        <el-table-column label="Push 结果" min-width="220">
          <template #default="{ row }">
            成功 {{ row.pushSuccess }} / 跳过 {{ row.pushSkipped }} / 失败 {{ row.pushFailed }}
          </template>
        </el-table-column>
        <el-table-column label="失败原因" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">{{ row.errorMessage || '-' }}</template>
        </el-table-column>
        <el-table-column label="创建时间" min-width="170">
          <template #default="{ row }">{{ formatDateTime(row.createdAt) }}</template>
        </el-table-column>
        <el-table-column label="完成时间" min-width="170">
          <template #default="{ row }">{{ formatDateTime(row.finishedAt) }}</template>
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

    <el-dialog
      v-model="createVisible"
      title="发送系统通知"
      width="560px"
      @close="handleCreateClose"
    >
      <el-form label-width="80px">
        <el-form-item label="标题" required>
          <el-input
            v-model="createForm.title"
            maxlength="80"
            show-word-limit
            placeholder="通知标题，展示在系统通知栏和站内消息"
          />
        </el-form-item>
        <el-form-item label="内容" required>
          <el-input
            v-model="createForm.content"
            type="textarea"
            :rows="5"
            maxlength="500"
            show-word-limit
            placeholder="通知正文"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="handleCreateClose">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="handleCreateSubmit">
          发送
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>
