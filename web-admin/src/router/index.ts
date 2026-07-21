import { createRouter, createWebHistory, type RouteRecordRaw } from 'vue-router'
import { getToken } from '@/utils/auth'
import { userRoutes } from '@/router/user'
import { activityRoutes } from '@/router/activity'
import { feedbackRoutes } from '@/router/feedback'
import { chatReportRoutes } from '@/router/chatReport'
declare module 'vue-router' {
  interface RouteMeta {
    title?: string
    icon?: string
    hidden?: boolean
  }
}

const routes: RouteRecordRaw[] = [
  {
    path: '/',
    component: () => import('@/layout/index.vue'),
    children: [
      {
        path: '',
        name: 'dashboard',
        component: () => import('@/views/home/index.vue'),
        meta: { title: '首页', icon: 'House' },
      },
      ...userRoutes,
      ...activityRoutes,
      ...feedbackRoutes,
      ...chatReportRoutes,
    ],
  },
  {
    path: '/login',
    name: 'login',
    component: () => import('@/views/login/index.vue'),
    meta: { title: '登录', hidden: true },
  },
  {
    path: '/404',
    name: '404',
    component: () => import('@/views/error/404.vue'),
    meta: { title: '页面未找到', hidden: true },
  },
  {
    path: '/500',
    name: '500',
    component: () => import('@/views/error/500.vue'),
    meta: { title: '服务器错误', hidden: true },
  },
  // 未匹配到的地址统一进入 404。
  {
    path: '/:pathMatch(.*)*',
    redirect: '/404',
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

const WHITE_LIST = ['/login', '/404', '/500']

router.beforeEach((to) => {
  const token = getToken()

  // 白名单页面无需登录。
  if (WHITE_LIST.includes(to.path)) {
    // 已登录用户访问登录页时回到首页。
    if (token && to.path === '/login') {
      return '/'
    }

    return true
  }

  // 非白名单页面必须有登录 token。
  if (token) {
    return true
  }

  // 未登录时跳转登录页，并保留原目标地址。
  return { path: '/login', query: { redirect: to.fullPath } }
})

export default router
