import type { RouteRecordRaw } from 'vue-router'

export const chatReportRoutes: RouteRecordRaw[] = [
  {
    path: 'chat-reports',
    name: 'chatReportList',
    component: () => import('@/views/chatReport/chatReportList.vue'),
    meta: { title: '聊天举报', icon: 'Warning' },
  },
]
