<template>
  <div class="l-upload" :class="[round ? 'round' : '', size]">
    <!-- el-upload 组件 -->
    <el-upload class="avatar-uploader" :class="size" :list-type="listType" :headers="headers" :data="data" :action="action"
      :show-file-list="multiple" :on-success="handleAvatarSuccess" :before-upload="beforeAvatarUpload"
      :on-progress="handleAvatarProgress" v-loading="uploadLoading" :multiple="multiple"
      :on-preview="handlePictureCardPreview" :on-remove="handlePictureCardRemove" :limit="limit" ref="uploadRef"
      v-model:file-list="fileList" :on-exceed="handleExceed">
      <!-- 单图模式 -->
      <template v-if="!multiple && listType === 'picture-card'">
        <div class="avatar" v-if="modelValue" @click.stop>
          <el-image class="image" :src="modelValue as string" fit="scale-down" ref="imageRef" />
          <label class="el-upload-list-success" v-if="modelValue">
            <el-icon class="el-upload-successIcon">
              <Check />
            </el-icon>
          </label>
          <!-- 鼠标 hover 是图片上方遮罩层 -->
          <div class="el-upload-actions">
            <!-- 图片预览ICON -->
            <div class="item" @click="handlePreview">
              <el-icon size="18" color="#fff">
                <ZoomIn />
              </el-icon>
            </div>
            <!-- 图片删除ICON -->
            <div class="item" @click="handleRemove">
              <el-icon size="18" color="#fff">
                <Delete />
              </el-icon>
            </div>
          </div>
        </div>
        <el-icon size="25" v-else class="avatar-uploader-icon">
          <Plus />
        </el-icon>
      </template>
      <!-- 多图模式 -->
      <template v-if="multiple && listType === 'picture-card'">
        <el-icon size="25" class="avatar-uploader-icon">
          <Plus />
        </el-icon>
      </template>
      <!-- 文件格式 -->
      <template v-if="!multiple && listType === 'text'">
        <el-button type="primary" v-if="!modelValue && !uploadLoading">选择文件</el-button>
        <!-- 上传进度条 -->
        <div v-if="uploadLoading" class="upload-progress">
          <div class="progress-info">
            <span class="progress-text">上传中...</span>
            <span class="progress-percent">{{ uploadProgress }}%</span>
          </div>
          <el-progress :percentage="uploadProgress" :show-text="false" :stroke-width="6" />
        </div>
        <!-- 单文件模式 -->
        <div v-if="modelValue && modelValue !== '' && !uploadLoading" class="file-info">
          <div class="file-path">
            <span class="file-name">{{ modelValue }}</span>
          </div>
          <div class="file-status" @mouseenter="showDelete = true" @mouseleave="showDelete = false" @click.stop>
            <el-icon v-if="!showDelete" class="success-icon" color="#67C23A" :size="20">
              <SuccessFilled />
            </el-icon>
            <el-icon v-else class="delete-icon" @click.stop="handleRemoveFile" color="#F56C6C" :size="20">
              <Delete />
            </el-icon>
          </div>
        </div>
        <!-- 多文件模式 -->
        <div v-if="multiple && Array.isArray(modelValue) && modelValue.length > 0" class="file-list-info">
          <div v-for="(file, index) in modelValue" :key="index" class="file-item">
            <div class="file-path">
              <span class="file-name">{{ file }}</span>
            </div>
            <div class="file-status" @mouseenter="hoverIndex = index" @mouseleave="hoverIndex = -1" @click.stop>
              <el-icon v-if="hoverIndex !== index" class="success-icon" color="#67C23A" :size="18">
                <SuccessFilled />
              </el-icon>
              <el-icon v-else class="delete-icon" @click.stop="handleRemoveFileAt(index)" color="#F56C6C" :size="18">
                <Delete />
              </el-icon>
            </div>
          </div>
        </div>
      </template>
    </el-upload>
    <!-- 实现图片大图预览 -->
    <el-image-viewer v-if="isImageView" :url-list="previewUrls" :initial-index="previewUrlIndex" @close="imageClose" />
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, toRefs, onBeforeUnmount, watch } from "vue";
import type { PropType } from "vue";
import { Plus, ZoomIn, Delete, Check, SuccessFilled } from "@element-plus/icons-vue"; // 引入 Element 中的图标
import {
  ElImageViewer,
  ElMessage,
} from "element-plus";
import type { UploadProps, UploadUserFile } from "element-plus"; // 类型导入
import { useSleepUserStore } from '@/stores/user'
const userStore = useSleepUserStore()
const props = defineProps({
  // 父组件 v-model 绑定的值
  modelValue: {
    type: [String, Array] as PropType<string | string[]>, // 支持字符串或数组类型
    default: () => [],
  },
  // 图片是否为圆形
  round: {
    type: Boolean,
    default: true,
  },
  // 是否多图
  multiple: {
    type: Boolean,
    default: false,
  },
  limit: {
    type: Number,
    default: 9,
  },
  // 组件展示大小. small 中 | mini 小 | default 大
  size: {
    type: String,
    default: 'default'
  },
  // 上传 文件大小限制
  fileSize: {
    type: Number,
    default: 400,
  },
  // 上传组件的样式风格
  listType: {
    type: String,
    default: 'picture-card'
  }
});
// 添加 upload 组件的引用
const headers = {
  Authorization: `Bearer ${userStore.token || ""}`
};
const uploadRef = ref();
const action = ref(import.meta.env.VITE_AP_BASE_FILE_URL);
const data = ref({
  is_source: 'mindcare',
});
const state = reactive({
  uploadLoading: false, // 图片文件大时 呈现出一个上传中的loading状态
  isImageView: false, // 图片预览时的状态
  previewUrls: [] as any[], // 图片预览列表（可以是多张）
});
const previewUrlIndex = ref<number>(0)
const { uploadLoading, isImageView, previewUrls } = toRefs(state);

const showDelete = ref(false); // 控制单文件删除按钮显示
const hoverIndex = ref(-1); // 控制多文件删除按钮显示
const uploadProgress = ref(0); // 上传进度

const emit = defineEmits(["update:modelValue"]);
const fileList = ref<UploadUserFile[]>([]);
// Add a watcher to sync fileList with modelValue
watch(() => props.modelValue, (newVal) => {
  if (Array.isArray(newVal)) {
    fileList.value = newVal.map((item: any) => ({
      url: item.image || item,
      name: item.image,
      status: "success",
    }));
  } else if (typeof newVal === "string" && newVal) {
    fileList.value = [{ url: newVal, name: newVal, status: "success" }];
  } else {
    fileList.value = [];
  }
}, { immediate: true });
// 预览图片
const handlePictureCardPreview: UploadProps["onPreview"] = (uploadFile) => {
  if (props.multiple) {
    const urls = Array.isArray(props.modelValue) ? props.modelValue : [];
    const index = urls.indexOf(uploadFile.url || "");
    previewUrlIndex.value = index;
    // 多图模式：显示所有已上传图片
    state.previewUrls = Array.isArray(props.modelValue)
      ? props.modelValue
      : props.modelValue?.split(",").filter(Boolean) || [];
  } else {
    // 单图模式：只显示当前图片
    state.previewUrls = uploadFile.url ? [uploadFile.url] : [];
  }
  state.isImageView = true;
};
// 单图预览
const handlePreview = () => {
  if (props.modelValue) {
    state.previewUrls = [props.modelValue];
    state.isImageView = true;
  }
};
// 关闭图片预览
function imageClose() {
  state.isImageView = false;
  state.previewUrls = []; // 每次关闭图片预览时清空一下
  // 清空上传组件的文件列表
  if (uploadRef.value) {
  }
}

// 删除图片 单图
function handleRemove() {
  emit("update:modelValue", "");
}
const handlePictureCardRemove: UploadProps["onRemove"] = (
  _uploadFile,
  uploadFiles
) => {
  if (props.multiple) {
    const urls = (uploadFiles as any[]).map(file => file.response?.data || file.url).filter(Boolean);
    emit("update:modelValue", urls); // 更新父组件的值
  } else {
    emit("update:modelValue", "");
  }
};
// 删除文件（单文件模式）
function handleRemoveFile() {
  showDelete.value = false;
  uploadProgress.value = 0;
  emit("update:modelValue", "");
}

// 删除指定索引的文件（多文件模式）
function handleRemoveFileAt(index: number) {
  hoverIndex.value = -1;
  if (Array.isArray(props.modelValue)) {
    const newFiles = [...props.modelValue];
    newFiles.splice(index, 1);
    emit("update:modelValue", newFiles);
  }
}
// 图片上传成功
const handleAvatarSuccess: UploadProps['onSuccess'] = (response, _uploadFile, uploadFiles) => {
  if (props.multiple) {
    // 检查是否所有文件都上传完成
    const isAllUploaded = uploadFiles.every((file: any) => file.status === 'success')
    // 将新上传的图片地址添加到列表
    if (isAllUploaded) {
      // 获取所有已上传文件的URL
      const urls = uploadFiles
        .map((file: any) => file.response?.data || file.url)
        .filter(Boolean)

      // 更新状态
      state.uploadLoading = false
      uploadProgress.value = 0 // 重置进度
      emit("update:modelValue", urls)
      state.previewUrls = urls
    }
  } else {
    const url = response.data
    emit("update:modelValue", url)
    state.previewUrls = [url]
    state.uploadLoading = false
    uploadProgress.value = 0 // 重置进度
  }
}

// 图片上传之前
const beforeAvatarUpload: UploadProps['beforeUpload'] = (rawFile) => {
  if (rawFile.size / 1024 > props.fileSize) {
    state.uploadLoading = false
    ElMessage.error(`图片大小不得超过${props.fileSize}KB!`)
    return 
  }
  // 文件验证通过,立即显示进度条
  state.uploadLoading = true;
  uploadProgress.value = 0;
  return true
}

// 图片上传时
function handleAvatarProgress(event: any) {
  state.uploadLoading = true;
  // 计算上传进度
  if (event.percent) {
    uploadProgress.value = Math.floor(event.percent);
  }
}
// 图片上传校验
const handleExceed: UploadProps["onExceed"] = (_files, _uploadFiles) => {
  ElMessage.warning(`最多上传 ${props.limit}张`);
};
onBeforeUnmount(() => {
  uploadRef.value.clearFiles();
});
</script>
<style scoped lang="scss">
.l-upload {
  :deep(.el-upload--picture-card) {
    background: #fff;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
    /* 添加阴影 */
  }

  &.mini {
    :deep(.el-upload--picture-card) {
      --el-upload-size: 80px;
      width: var(--el-upload-size);
      height: var(--el-upload-size);
    }

    :deep(.el-upload-list--picture-card .el-upload-list__item) {
      --el-upload-size: 80px;
      width: var(--el-upload-size);
      height: var(--el-upload-size);
    }
  }

  &.small {
    :deep(.el-upload--picture-card) {
      --el-upload-size: 100px;
      width: var(--el-upload-size);
      height: var(--el-upload-size);
    }

    :deep(.el-upload-list--picture-card .el-upload-list__item) {
      --el-upload-size: 100px;
      width: var(--el-upload-size);
      height: var(--el-upload-size);
    }
  }

  &.default {
    :deep(.el-upload--picture-card) {
      --el-upload-size: 148px;
      width: var(--el-upload-size);
      height: var(--el-upload-size);
    }

    :deep(.el-upload-list--picture-card .el-upload-list__item) {
      --el-upload-size: 148px;
      width: var(--el-upload-size);
      height: var(--el-upload-size);
    }
  }

  :deep(.el-upload) {
    border: 1px solid #dcdfe6;
  }

  // 去掉动画效果
  :deep(.el-list-enter-active),
  :deep(.el-list-leave-active) {
    transition: all 0s;
  }

  :deep(.el-upload:hover) {
    border-color: var(--el-color-primary);
  }

  .avatar-uploader {
    .miniSize {
      width: 50px;
      height: 50px;
    }

    .avatar {
      width: 100%;
      height: 100%;
      cursor: auto;
      position: relative;

      .image {
        width: 100%;
        height: 100%;
      }

      .el-upload-list-success {
        position: absolute;
        right: -15px;
        top: -6px;
        width: 40px;
        height: 24px;
        background: var(--el-color-success);
        text-align: center;
        transform: rotate(45deg);

        .el-upload-successIcon {
          transform: rotate(-45deg);
          color: #fff;
          font-size: 12px;
          margin-top: 11px;
        }
      }

      .el-upload-actions {
        top: 0;
        left: 0;
        cursor: auto;
        width: 100%;
        height: 100%;
        transition: 0.5s;
        position: absolute;
        background: rgba(0, 0, 0, 0.8);
        opacity: 0;
        display: flex;
        align-items: center;
        justify-content: center;

        .item {
          width: 40px;
          height: 40px;
          display: flex;
          cursor: pointer;
          align-items: center;
          justify-content: center;
        }
      }

      &:hover {
        .el-upload-actions {
          opacity: 1;
          display: flex;
          align-items: center;
          justify-content: center;
        }
      }
    }
  }
}

// 文件格式样式
.file-info {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background-color: #f5f7fa;
  transition: all 0.3s ease;

  &:hover {
    background-color: #ecf5ff;
    border-color: #409eff;
  }

  .file-path {
    flex: 1;
    display: flex;
    align-items: center;
    overflow: hidden;
    margin-right: 12px;

    .file-name {
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      color: #606266;
      font-size: 14px;
    }
  }

  .file-status {
    display: flex;
    align-items: center;
    cursor: pointer;
    transition: all 0.2s ease;

    .success-icon,
    .delete-icon {
      transition: all 0.2s ease;
    }

    .delete-icon:hover {
      transform: scale(1.2);
    }
  }
}

// 上传进度条样式
.upload-progress {
  margin-top: 12px;
  padding: 16px;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  background: linear-gradient(135deg, #f5f7fa 0%, #ecf5ff 100%);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);

  .progress-info {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 10px;

    .progress-text {
      font-size: 14px;
      color: #606266;
      font-weight: 500;
    }

    .progress-percent {
      font-size: 14px;
      color: #409eff;
      font-weight: 600;
    }
  }

  :deep(.el-progress) {
    .el-progress-bar__outer {
      background-color: #e4e7ed;
      border-radius: 3px;
    }

    .el-progress-bar__inner {
      background: linear-gradient(90deg, #409eff 0%, #66b1ff 100%);
      border-radius: 3px;
      transition: width 0.3s ease;
    }
  }
}

.file-list-info {
  margin-top: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;

  .file-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 10px 14px;
    border: 1px solid #e4e7ed;
    border-radius: 6px;
    background-color: #f5f7fa;
    transition: all 0.3s ease;

    &:hover {
      background-color: #ecf5ff;
      border-color: #409eff;
      transform: translateX(2px);
    }

    .file-path {
      flex: 1;
      display: flex;
      align-items: center;
      overflow: hidden;
      margin-right: 12px;

      .file-name {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        color: #606266;
        font-size: 14px;
      }
    }

    .file-status {
      display: flex;
      align-items: center;
      cursor: pointer;
      transition: all 0.2s ease;

      .success-icon,
      .delete-icon {
        transition: all 0.2s ease;
      }

      .delete-icon:hover {
        transform: scale(1.2);
      }
    }
  }
}

.round {
  :deep(.el-upload) {
    border-radius: 10px !important;
    overflow: hidden;
  }
}
</style>
