<script setup lang="ts">
import { Search } from '@element-plus/icons-vue'
import { onMounted, reactive, ref } from 'vue'
import { getUserList, type UserItem } from '@/api/modules/user'
import UserEditForm from '@/components/user/userEditForm.vue'

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

function getStatusType(status: number) {
  return status === 1 ? 'success' : 'danger'
}

function getStatusText(status: number) {
  return status === 1 ? '正常' : '禁用'
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
            <el-option label="禁用" :value="0" />
          </el-select>
        </el-form-item>

        <el-form-item>
          <el-button type="primary" :icon="Search" @click="handleSearch">搜索</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card  class="m-t-10">
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
        <el-table-column prop="device_no" label="设备号" min-width="160" show-overflow-tooltip>
          <template #default="{ row }">
            {{ row.device_no || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="push_platform" label="推送平台" min-width="120">
          <template #default="{ row }">
            {{ row.push_platform || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="registration_id" label="Registration ID" min-width="220" show-overflow-tooltip>
          <template #default="{ row }">
            {{ row.registration_id || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)">{{ getStatusText(row.status) }}</el-tag>
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
        <el-table-column prop="created_at" label="注册时间" min-width="180">
        </el-table-column>
        <el-table-column label="操作" width="90" fixed="right">
          <template #default="{ row }">
            <el-button type="primary" link @click="handleEdit(row)">编辑</el-button>
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
  </div>
</template>
