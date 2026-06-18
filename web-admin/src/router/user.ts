import type { RouteRecordRaw } from 'vue-router'

export const userRoutes: RouteRecordRaw[] = [
  {
    path: 'user',
    redirect: '/user/userList',
    meta: { title: '用户管理', icon: 'User' },
    children: [
      {
        path: 'userList',
        name: 'userList',
        component: () => import('@/views/user/userList.vue'),
        meta: { title: '用户列表' },
      },
    ],
  },
]
