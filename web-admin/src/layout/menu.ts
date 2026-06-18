import type { RouteRecordRaw } from 'vue-router'

export interface MenuNode {
  id: string
  title: string
  icon: string
  fullPath: string
  children?: MenuNode[]
}

export interface FlatMenuItem {
  id: string
  title: string
  fullPath: string
}

export const resolveRoutePath = (basePath: string, routePath: string) => {
  if (!routePath) {
    return basePath || '/'
  }

  if (routePath.startsWith('/')) {
    return routePath
  }

  if (basePath === '/' || !basePath) {
    return `/${routePath}`
  }

  return `${basePath}/${routePath}`.replace(/\/+/g, '/')
}

// 统一从路由树构建菜单，Header 和 Sidebar 共用，避免重复维护
export const buildMenuTree = (routes: readonly RouteRecordRaw[], basePath = '/'): MenuNode[] => {
  return routes.flatMap((route) => {
    if (route.meta?.hidden) {
      return []
    }

    const fullPath = resolveRoutePath(basePath, String(route.path ?? ''))
    const childRoutes = (route.children ?? []) as readonly RouteRecordRaw[]
    const childNodes = childRoutes.length ? buildMenuTree(childRoutes, fullPath) : []

    if (!route.meta?.title) {
      return childNodes
    }

    return [
      {
        id: String(route.name ?? fullPath),
        title: String(route.meta.title),
        icon: String(route.meta.icon ?? ''),
        fullPath,
        children: childNodes.length ? childNodes : undefined,
      },
    ]
  })
}

export const flattenMenuTree = (menuTree: MenuNode[]): FlatMenuItem[] => {
  return menuTree.flatMap((item) => {
    const children = item.children ? flattenMenuTree(item.children) : []

    if (item.children?.length) {
      return children
    }

    return [
      {
        id: item.id,
        title: item.title,
        fullPath: item.fullPath,
      },
    ]
  })
}
