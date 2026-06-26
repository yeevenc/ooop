<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import {
  createCategory,
  updateCategory,
  type AdminCategory,
  type CategoryPayload,
} from '@/api/modules/activity'
import UploadImage from '@/components/upload/uploadImage.vue'

const props = defineProps<{
  visible: boolean
  category: AdminCategory | null
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'success'): void
}>()

const submitting = ref(false)
const isEdit = ref(false)
const form = reactive({
  id: '',
  label: '',
  icon: '',
  sort: 0,
  status: 1,
})

watch(
  () => props.visible,
  (visible) => {
    if (!visible) {
      return
    }
    if (props.category) {
      isEdit.value = true
      form.id = props.category.id
      form.label = props.category.label
      form.icon = props.category.icon
      form.sort = props.category.sort
      form.status = props.category.status
    } else {
      isEdit.value = false
      form.id = ''
      form.label = ''
      form.icon = ''
      form.sort = 0
      form.status = 1
    }
  },
)

function handleClose() {
  emit('update:visible', false)
}

async function handleSubmit() {
  if (!isEdit.value && !form.id.trim()) {
    ElMessage.warning('请填写分类标识')
    return
  }
  if (!form.label.trim()) {
    ElMessage.warning('请填写分类名称')
    return
  }

  submitting.value = true
  try {
    const payload: CategoryPayload = {
      label: form.label.trim(),
      icon: form.icon,
      sort: form.sort,
      status: form.status,
    }
    if (isEdit.value) {
      await updateCategory(form.id, payload)
    } else {
      await createCategory({ ...payload, id: form.id.trim() })
    }
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
    :title="isEdit ? '编辑分类' : '新增分类'"
    width="420px"
    @update:model-value="handleClose"
  >
    <el-form :model="form" label-width="72px">
      <el-form-item label="标识">
        <el-input v-model="form.id" :disabled="isEdit" placeholder="英文标识，如 outdoor" />
      </el-form-item>
      <el-form-item label="名称">
        <el-input v-model="form.label" maxlength="32" placeholder="分类名称" />
      </el-form-item>
      <el-form-item label="图标">
        <UploadImage v-model="form.icon" size="small" />
      </el-form-item>
      <el-form-item label="排序">
        <el-input-number v-model="form.sort" :min="0" :step="10" />
      </el-form-item>
      <el-form-item label="状态">
        <el-switch
          v-model="form.status"
          :active-value="1"
          :inactive-value="0"
          active-text="启用"
          inactive-text="停用"
        />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="handleClose">取消</el-button>
      <el-button type="primary" :loading="submitting" @click="handleSubmit">保存</el-button>
    </template>
  </el-dialog>
</template>
