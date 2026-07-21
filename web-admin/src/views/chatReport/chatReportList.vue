<script setup lang="ts">
import { Search } from '@element-plus/icons-vue'
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  getChatReportDetail,
  getChatReportList,
  resolveChatReport,
  type ChatReportItem,
  type ChatReportReason,
  type ChatReportStatus,
} from '@/api/modules/chatReport'
import { formatDateTime } from '@/utils/date'

defineOptions({ name: 'chatReportList' })

const STATUS_OPTIONS: Array<{ label: string; value: ChatReportStatus }> = [
  { label: '待处理', value: 'pending' },
  { label: '举报成立', value: 'resolved' },
  { label: '未发现违规', value: 'dismissed' },
]

const statusMeta: Record<
  ChatReportStatus,
  { text: string; type: 'warning' | 'success' | 'info' }
> = {
  pending: { text: '待处理', type: 'warning' },
  resolved: { text: '举报成立', type: 'success' },
  dismissed: { text: '未发现违规', type: 'info' },
}

const reasonText: Record<ChatReportReason, string> = {
  spam: '垃圾广告',
  harassment: '骚扰辱骂',
  pornography: '色情低俗',
  fraud: '诈骗行为',
  illegal: '违法违规',
  other: '其他问题',
}

const loading = ref(false)
const detailLoading = ref(false)
const resolving = ref(false)
const tableData = ref<ChatReportItem[]>([])
const total = ref(0)
const detailVisible = ref(false)
const detail = ref<ChatReportItem | null>(null)
const queryForm = reactive({
  page: 1,
  pageSize: 10,
  keyword: '',
  status: 'pending' as ChatReportStatus | '',
})
const resolveForm = reactive({
  status: 'resolved' as Exclude<ChatReportStatus, 'pending'>,
  result: '',
  restrictionUntil: null as Date | null,
})

const restrictionShortcuts = [
  {
    text: '24 小时',
    value: () => new Date(Date.now() + 24 * 60 * 60 * 1000),
  },
  {
    text: '3 天',
    value: () => new Date(Date.now() + 3 * 24 * 60 * 60 * 1000),
  },
  {
    text: '7 天',
    value: () => new Date(Date.now() + 7 * 24 * 60 * 60 * 1000),
  },
]

function defaultRestrictionUntil() {
  return new Date(Date.now() + 24 * 60 * 60 * 1000)
}

function disablePastDate(value: Date) {
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  return value.getTime() < today.getTime()
}

function getReasonText(reason: string) {
  return reasonText[reason as ChatReportReason] ?? reason
}

function getStatusMeta(status: string) {
  return (
    statusMeta[status as ChatReportStatus] ?? {
      text: status,
      type: 'info' as const,
    }
  )
}

async function getList() {
  loading.value = true
  try {
    const response = await getChatReportList({
      page: queryForm.page,
      page_size: queryForm.pageSize,
      keyword: queryForm.keyword || undefined,
      status: queryForm.status || undefined,
    })
    tableData.value = response.data.list
    total.value = response.data.total
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

async function openDetail(row: ChatReportItem) {
  detailVisible.value = true
  detailLoading.value = true
  detail.value = null
  resolveForm.status = 'resolved'
  resolveForm.result = ''
  resolveForm.restrictionUntil = defaultRestrictionUntil()
  try {
    const response = await getChatReportDetail(row.id)
    detail.value = response.data
  } finally {
    detailLoading.value = false
  }
}

async function submitResolve() {
  if (!detail.value || detail.value.status !== 'pending' || resolving.value) {
    return
  }
  const result = resolveForm.result.trim()
  if (!result) {
    ElMessage.warning('请填写明确的处理结果，用户将在站内消息中看到')
    return
  }
  if (
    resolveForm.status === 'resolved' &&
    (!resolveForm.restrictionUntil ||
      resolveForm.restrictionUntil.getTime() <= Date.now())
  ) {
    ElMessage.warning('请选择晚于当前时间的聊天限制解除时间')
    return
  }

  resolving.value = true
  try {
    const response = await resolveChatReport(detail.value.id, {
      status: resolveForm.status,
      result,
      restriction_until:
        resolveForm.status === 'resolved'
          ? resolveForm.restrictionUntil?.toISOString()
          : undefined,
    })
    detail.value = response.data
    ElMessage.success('举报已处理，结果已通过站内消息通知用户')
    getList()
  } finally {
    resolving.value = false
  }
}

function evidenceSenderName(senderId: string) {
  if (!detail.value) {
    return senderId
  }
  if (senderId === detail.value.reporter.id) {
    return detail.value.reporter.nickname || `用户 ${senderId}`
  }
  if (senderId === detail.value.reportedUser.id) {
    return detail.value.reportedUser.nickname || `用户 ${senderId}`
  }
  return `用户 ${senderId}`
}

onMounted(getList)
</script>

<template>
  <div>
    <el-card shadow="never">
      <el-form :model="queryForm" inline>
        <el-form-item label="关键词">
          <el-input
            v-model="queryForm.keyword"
            clearable
            placeholder="举报编号 / 用户 ID / 内容"
            @clear="handleSearch"
            @keyup.enter="handleSearch"
          />
        </el-form-item>
        <el-form-item label="处理状态">
          <el-select
            v-model="queryForm.status"
            clearable
            placeholder="全部状态"
            style="width: 150px"
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
          <el-button type="primary" :icon="Search" @click="handleSearch"
            >搜索</el-button
          >
        </el-form-item>
      </el-form>
    </el-card>

    <el-card class="m-t-10">
      <el-table v-loading="loading" :data="tableData" stripe border>
        <el-table-column prop="id" label="举报编号" width="100" fixed="left" />
        <el-table-column label="举报原因" width="120">
          <template #default="{ row }">{{
            getReasonText(row.reason)
          }}</template>
        </el-table-column>
        <el-table-column label="举报人" min-width="150">
          <template #default="{ row }">
            <div class="user-cell">
              <span>{{ row.reporter.nickname || '-' }}</span>
              <span class="muted">ID {{ row.reporter.id }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="被举报人" min-width="150">
          <template #default="{ row }">
            <div class="user-cell">
              <span>{{ row.reportedUser.nickname || '-' }}</span>
              <span class="muted">ID {{ row.reportedUser.id }}</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column
          prop="description"
          label="补充说明"
          min-width="220"
          show-overflow-tooltip
        >
          <template #default="{ row }">{{ row.description || '-' }}</template>
        </el-table-column>
        <el-table-column prop="evidenceCount" label="证据消息" width="100" />
        <el-table-column label="状态" width="110">
          <template #default="{ row }">
            <el-tag :type="getStatusMeta(row.status).type">
              {{ getStatusMeta(row.status).text }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="提交时间" min-width="170">
          <template #default="{ row }">{{
            formatDateTime(row.createdAt)
          }}</template>
        </el-table-column>
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-button
              :type="row.status === 'pending' ? 'danger' : 'primary'"
              link
              @click="openDetail(row)"
            >
              {{ row.status === 'pending' ? '处理' : '查看' }}
            </el-button>
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

    <el-dialog
      v-model="detailVisible"
      title="聊天举报详情"
      width="760px"
      destroy-on-close
    >
      <div v-loading="detailLoading">
        <template v-if="detail">
          <el-descriptions :column="2" border>
            <el-descriptions-item label="举报编号">{{
              detail.id
            }}</el-descriptions-item>
            <el-descriptions-item label="会话 ID">{{
              detail.conversationId
            }}</el-descriptions-item>
            <el-descriptions-item label="举报人">
              {{ detail.reporter.nickname || '-' }}（ID
              {{ detail.reporter.id }}）
            </el-descriptions-item>
            <el-descriptions-item label="被举报人">
              {{ detail.reportedUser.nickname || '-' }}（ID
              {{ detail.reportedUser.id }}）
            </el-descriptions-item>
            <el-descriptions-item label="举报原因">
              {{ getReasonText(detail.reason) }}
            </el-descriptions-item>
            <el-descriptions-item label="提交时间">{{
              formatDateTime(detail.createdAt)
            }}</el-descriptions-item>
            <el-descriptions-item label="补充说明" :span="2">
              {{ detail.description || '-' }}
            </el-descriptions-item>
          </el-descriptions>

          <h4 class="section-title">
            聊天证据快照（{{ detail.evidenceCount }} 条）
          </h4>
          <div class="evidence-list">
            <div
              v-for="item in detail.evidence || []"
              :key="item.id"
              class="evidence-item"
            >
              <div class="evidence-meta">
                <strong>{{ evidenceSenderName(item.senderId) }}</strong>
                <span>{{ formatDateTime(item.createdAt) }}</span>
              </div>
              <el-image
                v-if="item.type === 'image'"
                :src="item.content"
                :preview-src-list="[item.content]"
                preview-teleported
                fit="cover"
                class="evidence-image"
              />
              <div v-else class="evidence-content">{{ item.content }}</div>
            </div>
            <el-empty
              v-if="!detail.evidence?.length"
              description="暂无可用聊天证据"
              :image-size="70"
            />
          </div>

          <template v-if="detail.status === 'pending'">
            <h4 class="section-title">处理举报</h4>
            <el-form label-width="90px">
              <el-form-item label="处理结论">
                <el-radio-group v-model="resolveForm.status">
                  <el-radio value="resolved">举报成立</el-radio>
                  <el-radio value="dismissed">未发现违规</el-radio>
                </el-radio-group>
              </el-form-item>
              <el-form-item label="处理结果">
                <el-input
                  v-model="resolveForm.result"
                  type="textarea"
                  :rows="4"
                  maxlength="500"
                  show-word-limit
                  placeholder="请填写用户可理解的处理结果，该内容将通过站内消息发送给举报人"
                />
              </el-form-item>
              <el-form-item
                v-if="resolveForm.status === 'resolved'"
                label="限制至"
              >
                <div class="restriction-field">
                  <el-date-picker
                    v-model="resolveForm.restrictionUntil"
                    type="datetime"
                    format="YYYY-MM-DD HH:mm"
                    placeholder="选择聊天限制解除时间"
                    :disabled-date="disablePastDate"
                    :shortcuts="restrictionShortcuts"
                    style="width: 260px"
                  />
                  <span class="restriction-tip"
                    >默认限制 24 小时，可自由选择解除时间</span
                  >
                </div>
              </el-form-item>
            </el-form>
          </template>
          <el-descriptions v-else class="result-panel" :column="1" border>
            <el-descriptions-item label="处理状态">
              {{ getStatusMeta(detail.status).text }}
            </el-descriptions-item>
            <el-descriptions-item label="处理结果">{{
              detail.handleResult
            }}</el-descriptions-item>
            <el-descriptions-item label="处理时间">
              {{ detail.handledAt ? formatDateTime(detail.handledAt) : '-' }}
            </el-descriptions-item>
            <el-descriptions-item
              v-if="detail.status === 'resolved'"
              label="聊天限制至"
            >
              {{
                detail.restrictionUntil
                  ? formatDateTime(detail.restrictionUntil)
                  : '-'
              }}
            </el-descriptions-item>
          </el-descriptions>
        </template>
      </div>
      <template #footer>
        <el-button @click="detailVisible = false">关闭</el-button>
        <el-button
          v-if="detail?.status === 'pending'"
          type="danger"
          :loading="resolving"
          @click="submitResolve"
        >
          确认处理并通知用户
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.user-cell {
  display: flex;
  flex-direction: column;
  line-height: 20px;
}

.muted,
.evidence-meta span {
  color: #909399;
  font-size: 12px;
}

.section-title {
  margin: 20px 0 10px;
}

.evidence-list {
  max-height: 300px;
  overflow-y: auto;
  padding: 10px;
  border: 1px solid #ebeef5;
  border-radius: 8px;
  background: #f8f9fb;
}

.evidence-item + .evidence-item {
  margin-top: 10px;
}

.evidence-meta {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 5px;
}

.evidence-content {
  padding: 8px 10px;
  border-radius: 6px;
  background: #fff;
  line-height: 20px;
  white-space: pre-wrap;
  word-break: break-word;
}

.evidence-image {
  width: 100px;
  height: 100px;
  border-radius: 6px;
}

.result-panel {
  margin-top: 20px;
}

.restriction-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.restriction-tip {
  color: #909399;
  font-size: 12px;
}
</style>
