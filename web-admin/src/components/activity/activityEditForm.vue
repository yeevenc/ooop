<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import {
  getActivityDetail,
  updateActivity,
  getCategoryList,
  type AdminCategory,
  type UpdateActivityPayload,
} from '@/api/modules/activity'

const props = defineProps<{
  visible: boolean
  activityId: string | null
}>()

const emit = defineEmits<{
  (e: 'update:visible', value: boolean): void
  (e: 'success'): void
}>()

const loading = ref(false)
const submitting = ref(false)
const categories = ref<AdminCategory[]>([])
const form = reactive<UpdateActivityPayload>({
  title: '',
  category_id: '',
  activity_time: '',
  location_text: '',
  city: '',
  total_count: 2,
  cost_type: '',
  fee_detail: '',
  gender_requirement: '',
  intro: '',
  notice: '',
})

// 打开时拉取详情 + 分类选项，字段直接赋值。
watch(
  () => props.visible,
  async (visible) => {
    if (!visible || !props.activityId) {
      return
    }
    loading.value = true
    try {
      const [detailRes, catRes] = await Promise.all([
        getActivityDetail(props.activityId),
        getCategoryList(),
      ])
      categories.value = catRes.data
      const d = detailRes.data
      form.title = d.title
      form.category_id = d.categoryId
      form.activity_time = d.activityTime
      form.location_text = d.locationText
      form.city = d.city
      form.total_count = d.totalCount
      form.cost_type = d.costType
      form.fee_detail = d.feeDetail
      form.gender_requirement = d.genderRequirement
      form.intro = d.intro
      form.notice = d.notice
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
  if (!props.activityId) {
    return
  }
  if (!form.title.trim()) {
    ElMessage.warning('请填写标题')
    return
  }
  if (!form.category_id) {
    ElMessage.warning('请选择分类')
    return
  }
  if (!form.intro.trim()) {
    ElMessage.warning('请填写简介')
    return
  }

  submitting.value = true
  try {
    await updateActivity(props.activityId, {
      ...form,
      title: form.title.trim(),
      intro: form.intro.trim(),
    })
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
    title="编辑活动"
    width="50%"
    @update:model-value="handleClose"
  >
    <el-form v-loading="loading" :model="form" label-width="90px">
      <el-form-item label="标题">
        <el-input v-model="form.title" maxlength="80" show-word-limit placeholder="活动标题" />
      </el-form-item>
      <el-form-item label="分类">
        <el-select v-model="form.category_id" placeholder="选择分类" style="width: 100%">
          <el-option v-for="c in categories" :key="c.id" :label="c.label" :value="c.id" />
        </el-select>
      </el-form-item>
      <el-form-item label="时间">
        <el-input v-model="form.activity_time" placeholder="如 19:00-21:00" />
      </el-form-item>
      <el-form-item label="城市">
        <el-input v-model="form.city" placeholder="城市" />
      </el-form-item>
      <el-form-item label="地点">
        <el-input v-model="form.location_text" placeholder="详细地点" />
      </el-form-item>
      <el-form-item label="人数">
        <el-input-number v-model="form.total_count" :min="2" />
      </el-form-item>
      <el-form-item label="费用方式">
        <el-input v-model="form.cost_type" placeholder="如 AA制 / 免费 / 我请客" />
      </el-form-item>
      <el-form-item label="费用说明">
        <el-input v-model="form.fee_detail" placeholder="选填" />
      </el-form-item>
      <el-form-item label="性别要求">
        <el-input v-model="form.gender_requirement" placeholder="选填" />
      </el-form-item>
      <el-form-item label="简介">
        <el-input
          v-model="form.intro"
          type="textarea"
          :rows="3"
          maxlength="1000"
          show-word-limit
        />
      </el-form-item>
      <el-form-item label="注意事项">
        <el-input
          v-model="form.notice"
          type="textarea"
          :rows="2"
          maxlength="500"
          show-word-limit
        />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="handleClose">取消</el-button>
      <el-button type="primary" :loading="submitting" @click="handleSubmit">保存</el-button>
    </template>
  </el-dialog>
</template>
