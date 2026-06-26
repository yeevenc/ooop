<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { getUserDetail, updateUser, type UpdateUserPayload } from '@/api/modules/user'
import UploadImage from '@/components/upload/uploadImage.vue'

const props = defineProps<{
  visible: boolean
  userId: number | null
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'success'): void
}>()

const loading = ref(false)
const submitting = ref(false)
const form = reactive({
  nickname: '',
  gender: '',
  avatar: '',
  bio: '',
})

// 打开弹窗时拉取详情并直接赋值，不做额外处理。
watch(
  () => props.visible,
  async (visible) => {
    if (!visible || !props.userId) {
      return
    }
    loading.value = true
    try {
      const res = await getUserDetail(props.userId)
      const data = res.data
      form.nickname = data.nickname
      form.gender = data.gender
      form.avatar = data.avatar
      form.bio = data.bio
    } catch {
      // 错误信息已由请求拦截器统一提示
    } finally {
      loading.value = false
    }
  },
)

function handleClose() {
  emit('update:visible', false)
}

async function handleSubmit() {
  if (!props.userId) {
    return
  }
  if (!form.nickname.trim()) {
    ElMessage.warning('昵称不能为空')
    return
  }

  submitting.value = true
  try {
    const payload: UpdateUserPayload = {
      nickname: form.nickname.trim(),
      gender: form.gender,
      avatar: form.avatar,
      bio: form.bio.trim(),
    }
    await updateUser(props.userId, payload)
    ElMessage.success('保存成功')
    emit('success')
    emit('update:visible', false)
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
    title="编辑用户资料"
    width="50%"
    @update:model-value="handleClose"
  >
    <el-form v-loading="loading" :model="form" position="left" label-width="auto">
      <el-form-item label="头像">
        <UploadImage v-model="form.avatar" size="small" />
      </el-form-item>
      <el-form-item label="昵称">
        <el-input
          v-model="form.nickname"
          maxlength="32"
          show-word-limit
          placeholder="请输入昵称"
        />
      </el-form-item>
      <el-form-item label="性别">
        <el-select v-model="form.gender" clearable placeholder="未设置" style="width: 100%">
          <el-option label="男" value="男" />
          <el-option label="女" value="女" />
        </el-select>
      </el-form-item>
      <el-form-item label="个性签名">
        <el-input
          v-model="form.bio"
          type="textarea"
          :rows="3"
          maxlength="200"
          show-word-limit
          placeholder="请输入个性签名"
        />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="handleClose">取消</el-button>
      <el-button type="primary" :loading="submitting" @click="handleSubmit">保存</el-button>
    </template>
  </el-dialog>
</template>
