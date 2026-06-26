import type { RouteRecordRaw } from 'vue-router'

export const feedbackRoutes: RouteRecordRaw[] = [
  {
    path: 'feedback',
    name: 'feedbackList',
    component: () => import('@/views/feedback/feedbackList.vue'),
    meta: { title: '意见反馈', icon: 'ChatDotRound' },
  },
]
