import type { RouteRecordRaw } from 'vue-router'

export const systemBroadcastRoutes: RouteRecordRaw[] = [
  {
    path: 'system-broadcasts',
    name: 'systemBroadcastList',
    component: () => import('@/views/systemBroadcast/systemBroadcastList.vue'),
    meta: { title: '系统通知', icon: 'Bell' },
  },
]
