<script setup lang="ts">
import { Search } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { computed, onMounted, reactive, ref } from 'vue'
import {
  banUser,
  getUserList,
  unbanUser,
  type BanType,
  type UserItem,
} from '@/api/modules/user'
import UserEditForm from '@/components/user/userEditForm.vue'

/**
 * APP 用户封禁（后台操作）相关逻辑备注：
 * - 对象：APP 端 users，不是后台管理员
 * - 限时：时间区间选择 → 换算天数展示 + duration_hours 提交
 * - 永久：不传 duration_hours
 * - 解封：单独调 unban 接口
 */
const MS_HOUR = 60 * 60 * 1000
const MS_DAY = 24 * MS_HOUR

/** 默认封禁区间：当前时间起 7 天 */
function defaultBanRange(): [Date, Date] {
  const start = new Date()
  const end = new Date(start.getTime() + 7 * MS_DAY)
  return [start, end]
}

/** 区间跨度毫秒；结束不晚于开始时返回 0 */
function calcRangeMs(range: [Date, Date] | null): number {
  if (!range || range.length !== 2) {
    return 0
  }
  const ms = range[1].getTime() - range[0].getTime()
  return ms > 0 ? ms : 0
}

/** 区间天数（向上取整，不足 1 天按 1 天） */
function calcRangeDays(range: [Date, Date] | null): number {
  const ms = calcRangeMs(range)
  if (ms <= 0) {
    return 0
  }
  return Math.max(1, Math.ceil(ms / MS_DAY))
}

/**
 * 提交给接口的封禁小时数：
 * 优先用「结束时间 − 当前时间」，使解封时间尽量对齐所选结束时间；
 * 若结束时间已过，则回退为区间跨度。
 */
function calcBanDurationHours(range: [Date, Date] | null): number {
  if (!range || range.length !== 2) {
    return 0
  }
  const end = range[1].getTime()
  const fromNow = end - Date.now()
  if (fromNow > 0) {
    return Math.max(1, Math.ceil(fromNow / MS_HOUR))
  }
  const span = calcRangeMs(range)
  if (span <= 0) {
    return 0
  }
  return Math.max(1, Math.ceil(span / MS_HOUR))
}

defineOptions({ name: 'userList' })

const loading = ref(false)
const tableData = ref<UserItem[]>([])
const total = ref(0)
const queryForm = reactive({
  page: 1,
  pageSize: 30,
  keyword: '',
  status: undefined as number | undefined,
})

const sourceMap: Record<string, string> = {
  aliyun_mobile: '本机号码',
  mobile_code: '验证码登录',
  password: '密码注册',
}

const genderMap: Record<string, string> = {
  male: '男',
  female: '女',
}

function isBanned(row: UserItem) {
  return row.status !== 1
}

function getStatusType(row: UserItem) {
  if (!isBanned(row)) {
    return 'success'
  }
  return row.banned_until ? 'warning' : 'danger'
}

function getStatusText(row: UserItem) {
  if (!isBanned(row)) {
    return '正常'
  }
  return row.banned_until ? '限时封禁' : '永久封禁'
}

function getRegisterSourceText(source: string) {
  return sourceMap[source] ?? (source || '-')
}

function getGenderText(gender: string) {
  return genderMap[gender] ?? (gender || '-')
}

async function getList() {
  loading.value = true
  try {
    const response = await getUserList({
      page: queryForm.page,
      page_size: queryForm.pageSize,
      keyword: queryForm.keyword,
      status: queryForm.status,
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

const editVisible = ref(false)
const editUserId = ref<number | null>(null)

function handleEdit(row: UserItem) {
  editUserId.value = row.id
  editVisible.value = true
}

// —— 封禁弹窗（永久 / 限时区间）——
const banVisible = ref(false)
const banSubmitting = ref(false)
const banTarget = ref<UserItem | null>(null)
const banForm = reactive({
  type: 'permanent' as BanType,
  /** 限时封禁时间区间 [开始, 结束] */
  range: null as [Date, Date] | null,
  /** 封禁原因备注 */
  reason: '',
})

const banRangeDays = computed(() => calcRangeDays(banForm.range))
const banDurationHours = computed(() => calcBanDurationHours(banForm.range))

function handleOpenBan(row: UserItem) {
  banTarget.value = row
  banForm.type = 'permanent'
  banForm.range = defaultBanRange()
  banForm.reason = ''
  banVisible.value = true
}

function handleBanClose() {
  banVisible.value = false
  banTarget.value = null
}

async function handleBanSubmit() {
  if (!banTarget.value) {
    return
  }
  if (banForm.type === 'temporary') {
    if (!banForm.range || banForm.range.length !== 2) {
      ElMessage.warning('请选择封禁时间区间')
      return
    }
    if (calcRangeMs(banForm.range) <= 0) {
      ElMessage.warning('结束时间必须晚于开始时间')
      return
    }
    if (banDurationHours.value <= 0) {
      ElMessage.warning('结束时间必须晚于当前时间')
      return
    }
  }

  banSubmitting.value = true
  try {
    await banUser(banTarget.value.id, {
      type: banForm.type,
      // 按结束时间相对当前换算小时数（接口：now + hours）
      duration_hours: banForm.type === 'temporary' ? banDurationHours.value : undefined,
      reason: banForm.reason.trim() || undefined,
    })
    ElMessage.success(
      banForm.type === 'temporary'
        ? `封禁成功，区间 ${banRangeDays.value} 天`
        : '封禁成功',
    )
    handleBanClose()
    await getList()
  } catch {
    // 错误由请求拦截器统一提示
  } finally {
    banSubmitting.value = false
  }
}

async function handleUnban(row: UserItem) {
  try {
    await ElMessageBox.confirm(
      `确认解封用户「${row.nickname || row.phone || row.id}」？`,
      '解封确认',
      { type: 'warning', confirmButtonText: '确认解封', cancelButtonText: '取消' },
    )
  } catch {
    return
  }

  loading.value = true
  try {
    await unbanUser(row.id)
    ElMessage.success('解封成功')
    await getList()
  } catch {
    // 错误由请求拦截器统一提示
  } finally {
    loading.value = false
  }
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
            placeholder="手机号 / 用户名 / 昵称"
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
            <el-option label="正常" :value="1" />
            <el-option label="封禁" :value="0" />
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
        style="height: calc(100vh - 320px);"
        stripe
        border
        :data="tableData"
      >
        <el-table-column prop="id" label="ID" width="90" fixed="left" />
        <el-table-column prop="phone" label="手机号" min-width="140" />
        <el-table-column prop="username" label="用户名" min-width="140">
          <template #default="{ row }">
            {{ row.username || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="nickname" label="昵称" min-width="140">
          <template #default="{ row }">
            {{ row.nickname || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="avatar" label="头像" min-width="90">
          <template #default="{ row }">
            <el-image
              v-if="row.avatar"
              :src="row.avatar"
              fit="cover"
              :preview-src-list="[row.avatar]"
              preview-teleported
              style="width: 36px; height: 36px; border-radius: 50%; cursor: pointer"
            />
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column prop="gender" label="性别" width="90">
          <template #default="{ row }">
            {{ getGenderText(row.gender) }}
          </template>
        </el-table-column>
        <el-table-column prop="region" label="地区" min-width="120">
          <template #default="{ row }">
            {{ row.region || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="bio" label="个性签名" min-width="180" show-overflow-tooltip>
          <template #default="{ row }">
            {{ row.bio || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="platform" label="平台" width="100">
          <template #default="{ row }">
            {{ row.platform || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="110">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row)">{{ getStatusText(row) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="banned_until" label="解封时间" min-width="170">
          <template #default="{ row }">
            <template v-if="isBanned(row)">
              {{ row.banned_until || '永久' }}
            </template>
            <template v-else>-</template>
          </template>
        </el-table-column>
        <el-table-column prop="ban_reason" label="封禁备注" min-width="140" show-overflow-tooltip>
          <template #default="{ row }">
            {{ row.ban_reason || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="register_source" label="注册来源" min-width="130">
          <template #default="{ row }">
            {{ getRegisterSourceText(row.register_source) }}
          </template>
        </el-table-column>
        <el-table-column prop="last_login_at" label="最近登录" min-width="180">
          <template #default="{ row }">
            {{ row.last_login_at || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="注册时间" min-width="180" />
        <el-table-column label="操作" width="160" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="handleEdit(row)">编辑</el-button>
            <el-button
              v-if="!isBanned(row)"
              type="danger"
              link
              @click="handleOpenBan(row)"
            >
              封禁
            </el-button>
            <el-button v-else type="success" link @click="handleUnban(row)">解封</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        class="m-t-10"
        background
        layout="total, sizes, prev, pager, next, jumper"
        :current-page="queryForm.page"
        :page-size="queryForm.pageSize"
        :page-sizes="[30, 50, 100]"
        :total="total"
        @size-change="handleSizeChange"
        @current-change="handleCurrentChange"
      />
    </el-card>

    <UserEditForm v-model:visible="editVisible" :user-id="editUserId" @success="getList" />

    <el-dialog
      :model-value="banVisible"
      title="封禁用户"
      width="480px"
      @update:model-value="(v: boolean) => !v && handleBanClose()"
    >
      <el-form label-width="96px">
        <el-form-item label="用户">
          {{ banTarget?.nickname || banTarget?.phone || banTarget?.id || '-' }}
        </el-form-item>
        <el-form-item label="封禁类型" required>
          <el-radio-group v-model="banForm.type">
            <el-radio value="permanent">永久封禁</el-radio>
            <el-radio value="temporary">限时封禁</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="banForm.type === 'temporary'" label="封禁区间" required>
          <el-date-picker
            v-model="banForm.range"
            type="datetimerange"
            range-separator="至"
            start-placeholder="开始时间"
            end-placeholder="结束时间"
            format="YYYY-MM-DD HH:mm"
            :default-time="[new Date(2000, 0, 1, 0, 0, 0), new Date(2000, 0, 1, 23, 59, 59)]"
            style="width: 100%"
          />
          <div v-if="banRangeDays > 0" class="ban-range-tip">
            区间共 <strong>{{ banRangeDays }}</strong> 天；
            将封禁至所选结束时间（约 {{ banDurationHours }} 小时）
          </div>
        </el-form-item>
        <el-form-item label="备注">
          <el-input
            v-model="banForm.reason"
            type="textarea"
            :rows="3"
            maxlength="200"
            show-word-limit
            placeholder="选填，封禁原因备注，将展示给 APP 用户"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="handleBanClose">取消</el-button>
        <el-button type="danger" :loading="banSubmitting" @click="handleBanSubmit">
          确认封禁
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.ban-range-tip {
  margin-top: 8px;
  color: var(--el-text-color-secondary);
  font-size: 13px;
  line-height: 1.5;
}

.ban-range-tip strong {
  color: var(--el-color-danger);
}
</style>
