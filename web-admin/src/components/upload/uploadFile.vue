<template>
    <div class="upload-component">
        <!-- 上传按钮区域 -->
        <div class="upload-area" v-if="!fileList.length || !showFileList">
            <el-upload ref="uploadRef" :action="uploadUrl" :data="data" :accept="accept" :multiple="multiple"
                :before-upload="beforeUpload" :on-success="handleSuccess" :on-error="handleError"
                :on-remove="handleRemove" :on-preview="handlePreview" :file-list="fileList" :limit="limit"
                :on-exceed="handleExceed" :disabled="disabled" :drag="drag" :list-type="listType">
                <div v-if="drag" class="upload-drag-area">
                    <el-icon class="upload-icon">
                        <Plus />
                    </el-icon>
                    <div class="el-upload__text">
                        将文件拖到此处，或<em>点击上传</em>
                    </div>
                </div>
                <div v-else class="upload-button">
                    <el-button plain :size="buttonSize" :type="buttonType" :disabled="disabled">
                        <el-icon>
                            <Upload />
                        </el-icon>{{ buttonLabel }}
                    </el-button>
                </div>
                <template #tip>
                    <div class="el-upload__tip" v-if="tip">
                        {{ tip }}
                    </div>
                </template>
            </el-upload>
        </div>

        <!-- 文件列表 -->
        <div v-if="showFileList && fileList.length" class="file-list-container">
            <div v-for="(file, index) in fileList" :key="file.uid || index" class="file-item">
                <!-- 视频文件 -->
                <div v-if="isVideoFile(file)" class="file-preview">
                    <video fit="contain" :src="file.url || file.response?.data" controls class="video"
                        style="width:100%;">
                        您的浏览器不支持视频播放
                    </video>

                    <div class="file-info">
                        <span class="file-name">{{ file.name }}</span>
                        <span class="file-size">{{ formatFileSize(file.size) }}</span>
                    </div>
                    <!-- 视频地址 -->
                    <div class="video-url-container">
                        <el-input :value="file.url || file.response?.data" readonly placeholder="视频地址"
                            class="video-url-input">
                        </el-input>
                        <el-button type="warning" @click="copyVideoUrl(file.url || file.response?.data)"
                            class="copy-btn">
                            复制地址
                        </el-button>
                    </div>
                    <div class="file-actions">
                        <el-button size="small" @click="handleDownload(file)">下载</el-button>
                        <el-button size="small" type="danger" @click="handleRemove(file)">删除</el-button>
                    </div>
                </div>

                <!-- 音频文件 -->
                <div v-else-if="isAudioFile(file)" class="file-preview">
                    <audio :src="file.url || file.response?.data" controls width="100%">
                        您的浏览器不支持音频播放
                    </audio>
                    <div class="file-info">
                        <span class="file-name">{{ file.name }}</span>
                        <span class="file-size">{{ formatFileSize(file.size) }}</span>
                    </div>
                    <div class="file-actions">
                        <el-button size="small" @click="handleDownload(file)">下载</el-button>
                        <el-button size="small" type="danger" @click="handleRemove(file)">删除</el-button>
                    </div>
                </div>

                <!-- 图片文件 -->
                <div v-else-if="isImageFile(file)" class="file-preview">
                    <el-image :src="file.url || file.response?.data"
                        :preview-src-list="[file.url || file.response?.data]" :preview-teleported="true" fit="cover"
                        class="image-preview" />
                    <div class="file-info">
                        <span class="file-name">{{ file.name }}</span>
                        <span class="file-size">{{ formatFileSize(file.size) }}</span>
                    </div>
                    <div class="file-actions">
                        <el-button size="small" @click="handleDownload(file)">下载</el-button>
                        <el-button size="small" type="danger" @click="handleRemove(file)">删除</el-button>
                    </div>
                </div>

                <!-- 其他文件 -->
                <div v-else class="file-preview">
                    <div class="file-icon">
                        <el-icon>
                            <Document />
                        </el-icon>
                    </div>
                    <div class="file-info">
                        <span class="file-name">{{ file.name }}</span>
                        <span class="file-size">{{ formatFileSize(file.size) }}</span>
                    </div>
                    <div class="file-actions">
                        <el-button size="small" @click="handleDownload(file)">下载</el-button>
                        <el-button size="small" type="danger" @click="handleRemove(file)">删除</el-button>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import {
    Upload,
    Plus,
    Document,
} from '@element-plus/icons-vue'
import {
    ElMessage,
    ElMessageBox,
   type UploadProps
} from 'element-plus'
import { getUploadImageAction, resolveUploadURL } from '@/utils/upload'

// 定义props
const props = withDefaults(defineProps<{
    modelValue?: string | string[]
    multiple?: boolean
    limit?: number
    maxSize?: number // 最大文件大小(MB)
    disabled?: boolean
    drag?: boolean
    listType?: 'text' | 'picture' | 'picture-card'
    buttonType?: 'primary' | 'success' | 'warning' | 'danger' | 'info'
    buttonSize?: 'large' | 'default' | 'small'
    buttonLabel?: string
    tip?: string
    showFileList?: boolean
}>(), {
    modelValue: () => '',
    multiple: false,
    limit: 1,
    maxSize: 10, // 默认10MB
    disabled: false,
    drag: false,
    listType: 'text',
    buttonType: 'primary',
    buttonSize: 'default',
    buttonLabel: '上传文件',
    tip: '',
    showFileList: true,
})

// 定义emit
const emit = defineEmits<{
    'update:modelValue': [value: string | string[]]
    'success': [file: any, fileList: any[]]
    'error': [error: any]
    'remove': [file: any]
    'exceed': [files: FileList, fileList: any[]]
}>()
// 上传文件接口地址
const uploadUrl = getUploadImageAction()
// 内部文件列表
const fileList = ref<any[]>([])

// 引用上传组件
const uploadRef = ref()
//默认的文件类型
const accept = ref('video/*,.mp4,.avi,.mov,.wmv,.flv,.webm,.mkv,.pag')

// 额外参数
const data = ref({
    is_source: import.meta.env.VITE_DATA,
});
// 文件类型判断
const isVideoFile = (file: any) => /\.(mp4|avi|mov|wmv|flv|webm|mkv)$/i.test(file.url || file.name || '')
const isAudioFile = (file: any) => /\.(mp3|wav|ogg|flac|aac)$/i.test(file.name || file.url || '')
const isImageFile = (file: any) => /\.(jpg|jpeg|png|gif|bmp|webm|svg)$/i.test(file.name || file.url || '')

// 格式化文件大小
const formatFileSize = (size: number) => {
    if (!size) {
        return
    }
    if (size < 1024) return size + ' B'
    else if (size < 1024 * 1024) return (size / 1024).toFixed(2) + ' KB'
    else return (size / 1024 / 1024).toFixed(2) + ' MB'
}

// 上传前验证
const beforeUpload: UploadProps['beforeUpload'] = (file: File) => {
    // 验证文件大小
    const maxSize = props.maxSize * 1024 * 1024 // MB转字节
    if (file.size > maxSize) {
        ElMessage.error(`文件大小不能超过 ${props.maxSize}MB!`)
        return false
    }
    // 验证文件类型
    const fileName = file.name.toLowerCase()
    const validExtensions = ['.mp4', '.avi', '.mov', '.wmv', '.flv', '.webm', '.mkv', '.pag']
    const hasValidExtension = validExtensions.some(ext => fileName.endsWith(ext))

    if (!hasValidExtension) {
        ElMessage.error('仅支持上传视频文件 (MP4, AVI, MOV, WMV, FLV, WebM, MKV)!')
        return false
    }
    return true
}

// 上传成功处理
const handleSuccess = (response: any, file: any, uploadFileList: any[]) => {
    // 更新内部文件列表
    const newFileList = uploadFileList.map(item => ({
        ...item,
        url: resolveUploadURL(item.response?.data) || item.url,
        response: {
            ...item.response,
            data: resolveUploadURL(item.response?.data) || item.url
        }
    }))
    fileList.value = newFileList

    // 更新v-model绑定的值
    if (props.multiple) {
        const urls = newFileList.map(item => resolveUploadURL(item.response?.data) || item.url).filter(Boolean)
        emit('update:modelValue', urls)
    } else {
        const url = resolveUploadURL(response?.data) || resolveUploadURL(file?.response?.data) || file?.url
        if (url) {
            emit('update:modelValue', url)
        }
    }
    emit('success', file, newFileList)
    ElMessage.success('上传成功!')
}

// 上传错误处理
const handleError = (error: any) => {
    ElMessage.error('上传失败!')
    emit('error', error)
}

// 文件移除处理
const handleRemove = async (file: any) => {
    try {
        await ElMessageBox.confirm('确定要删除这个文件吗？', '删除确认', {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning'
        })

        const index = fileList.value.findIndex(item => item.uid === file.uid)
        if (index > -1) {
            fileList.value.splice(index, 1)

            // 更新v-model绑定的值
            if (props.multiple) {
                const urls = fileList.value.map(item => item.response?.url || item.url).filter(Boolean)
                emit('update:modelValue', urls)
            } else {
                emit('update:modelValue', '')
            }
        }

        emit('remove', file)
    } catch {
        // 用户取消删除
    }
}

// 文件预览处理
const handlePreview = (file: any) => {
    if (isImageFile(file)) {
        // 图片预览已通过el-image组件实现
    } else if (isVideoFile(file) || isAudioFile(file)) {
        // 视频和音频文件会在列表中直接显示
    } else {
        // 其他文件类型可考虑下载或使用外部预览服务
        handleDownload(file)
    }
}

// 文件下载处理
const handleDownload = (file: any) => {
    const url = file.url || file.response?.data
    if (url) {
        const a = document.createElement('a')
        a.href = url
        a.download = file.name || 'download'
        a.click()
    } else {
        ElMessage.warning('文件地址不存在')
    }
}

// 文件超出限制处理
const handleExceed = (files: FileList, fileList: any[]) => {
    ElMessage.warning(`最多只能上传 ${props.limit} 个文件!`)
    emit('exceed', files, fileList)
}

// 监听v-model值变化，更新内部文件列表
watch(
    () => props.modelValue,
    (newVal) => {
        if (Array.isArray(newVal)) {
            // 多文件模式
            fileList.value = newVal.map((url, index) => ({
                uid: index,
                name: `file_${index}`,
                url,
                status: 'success'
            }))
        } else if (newVal) {
            // 单文件模式
            fileList.value = [{
                uid: 0,
                name: 'file_0',
                url: newVal,
                status: 'success'
            }]
        } else {
            fileList.value = []
        }
    },
    { immediate: true }
)
// 复制视频地址
const copyVideoUrl = (url: string) => {
    navigator.clipboard.writeText(url);
    ElMessage.success("复制成功");
}
// 暴露方法给父组件
defineExpose({
    uploadRef,
    fileList,
    submit: () => uploadRef.value?.submit()
})
</script>

<style scoped>
.upload-component {
    width: 100%;
}

.upload-area {
    margin-bottom: 10px;
}

.upload-drag-area {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 40px 0;
    border: 1px dashed var(--el-border-color);
    border-radius: 6px;
    cursor: pointer;
    transition: var(--el-transition-duration-fast);
}

.upload-drag-area:hover {
    border-color: var(--el-color-primary);
}

.upload-icon {
    font-size: 48px;
    color: #8c939d;
    margin-bottom: 16px;
}

.upload-button {
    display: inline-block;
}

.file-list-container {
    display: flex;
    flex-wrap: wrap;
    gap: 15px;
}

.file-item {
    width: 100%;
    border: 1px solid #e4e7ed;
    border-radius: 4px;
    padding: 10px;
    margin-bottom: 10px;
}

.file-preview {
    display: flex;
    flex-direction: column;
    gap: 10px;
}

.image-preview {
    width: 100%;
    height: 200px;
    border-radius: 4px;
}

.file-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 60px;
    height: 60px;
    background-color: #f5f7fa;
    border: 1px solid #e4e7ed;
    border-radius: 4px;
}

.file-info {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 5px 0;
}

.file-name {
    flex: 1;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
}

.file-size {
    color: #909399;
    font-size: 12px;
    margin-left: 10px;
}

.file-actions {
    display: flex;
    justify-content: flex-end;
    gap: 5px;
    margin-top: 5px;
}

.video-url-container {
    display: flex;
    gap: 10px;
    align-items: center;
}

.video-url-input {
    flex: 1;
}

.copy-btn {
    flex-shrink: 0;
}

@media (min-width: 768px) {
    .file-item {
        width: calc(80% - 15px);
    }
}

@media (min-width: 1024px) {
    .file-item {
        width: calc(50% - 15px);
    }
}
</style>
