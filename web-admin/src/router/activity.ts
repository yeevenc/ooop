import type { RouteRecordRaw } from 'vue-router'

export const activityRoutes: RouteRecordRaw[] = [
  {
    path: 'activity',
    redirect: '/activity/activityList',
    meta: { title: '活动管理', icon: 'Calendar' },
    children: [
      {
        path: 'activityList',
        name: 'activityList',
        component: () => import('@/views/activity/activityList.vue'),
        meta: { title: '活动列表' },
      },
      {
        path: 'activityCategory',
        name: 'activityCategory',
        component: () => import('@/views/activity/activityCategory.vue'),
        meta: { title: '活动分类' },
      },
    ],
  },
]
