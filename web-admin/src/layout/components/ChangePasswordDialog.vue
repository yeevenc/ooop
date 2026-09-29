<script setup lang="ts">
import { reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { changePassword } from '@/api/modules/auth'
import { useOooPUserStore } from '@/stores/user'

defineProps<{
  visible: boolean
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
}>()

const userStore = useOooPUserStore()
const submitting = ref(false)
const form = reactive({
  oldPassword: '',
  newPassword: '',
  confirmPassword: '',
})

function resetForm() {
  form.oldPassword = ''
  form.newPassword = ''
  form.confirmPassword = ''
}

function handleClose() {
  emit('update:visible', false)
  resetForm()
}

async function handleSubmit() {
  if (submitting.value) {
    return
  }
  if (!form.oldPassword) {
    ElMessage.warning('请输入原密码')
    return
  }
  if (form.newPassword.length < 8) {
    ElMessage.warning('新密码长度不能少于 8 位')
    return
  }
  if (form.newPassword === form.oldPassword) {
    ElMessage.warning('新密码不能与原密码相同')
    return
  }
  if (form.newPassword !== form.confirmPassword) {
    ElMessage.warning('两次输入的新密码不一致')
    return
  }

  submitting.value = true
  try {
    await changePassword({
      old_password: form.oldPassword,
      new_password: form.newPassword,
    })
    ElMessage.success('密码已修改，请重新登录')
    handleClose()
    userStore.logout()
  } catch {
    // 错误信息已由请求拦截器统一提示
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <el-dialog
    :model-value="visible"
    title="修改密码"
    width="420px"
    append-to-body
    destroy-on-close
    @update:model-value="handleClose"
  >
    <el-form :model="form" label-width="96px" @submit.prevent="handleSubmit">
      <el-form-item label="原密码" required>
        <el-input
          v-model="form.oldPassword"
          type="password"
          show-password
          autocomplete="current-password"
          placeholder="请输入原密码"
        />
      </el-form-item>
      <el-form-item label="新密码" required>
        <el-input
          v-model="form.newPassword"
          type="password"
          show-password
          maxlength="64"
          autocomplete="new-password"
          placeholder="至少 8 位"
        />
      </el-form-item>
      <el-form-item label="确认新密码" required>
        <el-input
          v-model="form.confirmPassword"
          type="password"
          show-password
          maxlength="64"
          autocomplete="new-password"
          placeholder="请再次输入新密码"
          @keyup.enter="handleSubmit"
        />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="handleClose">取消</el-button>
      <el-button type="primary" :loading="submitting" @click="handleSubmit">保存</el-button>
    </template>
  </el-dialog>
</template>
