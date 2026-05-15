import { createRouter, createWebHistory } from 'vue-router'

import { useAuthStore } from '@/stores/auth'
import { useConfigStore } from '@/stores/config'
import type { Role } from '@/lib/types'

declare module 'vue-router' {
  interface RouteMeta {
    // Require a non-anonymous session.
    requiresAuth?: boolean
    // Require the user's role to be one of these. Implies requiresAuth.
    requiresRole?: Role[]
  }
}

const router = createRouter({
  history: createWebHistory(),
  routes: [
    // The homepage is the document with path "" — DocView resolves that case.
    { path: '/', name: 'home', component: () => import('@/views/DocView.vue'), props: () => ({ path: '' }) },
    {
      path: '/login',
      name: 'login',
      component: () => import('@/views/Login.vue'),
    },
    {
      // `:path(.*)*` catches arbitrarily-deep slash-separated paths and binds
      // them as a string array param; the route component receives a single
      // joined string via the props mapper.
      path: '/doc/:path(.*)*',
      name: 'doc-view',
      component: () => import('@/views/DocView.vue'),
      props: (route) => ({ path: joinPath(route.params.path) }),
    },
    {
      path: '/edit/:path(.*)*',
      name: 'doc-edit',
      component: () => import('@/views/DocEdit.vue'),
      meta: { requiresRole: ['admin', 'editor'] },
      props: (route) => ({ path: joinPath(route.params.path), mode: 'edit' as const }),
    },
    {
      path: '/new/:path(.*)*',
      name: 'doc-new',
      component: () => import('@/views/DocEdit.vue'),
      meta: { requiresRole: ['admin', 'editor'] },
      props: (route) => ({ path: joinPath(route.params.path), mode: 'new' as const }),
    },
    {
      path: '/admin',
      redirect: '/admin/users',
    },
    {
      path: '/admin/users',
      name: 'admin-users',
      component: () => import('@/views/admin/Users.vue'),
      meta: { requiresRole: ['admin'] },
    },
    {
      path: '/admin/access-rules',
      name: 'admin-access-rules',
      component: () => import('@/views/admin/AccessRules.vue'),
      meta: { requiresRole: ['admin'] },
    },
    {
      path: '/admin/settings',
      name: 'admin-settings',
      component: () => import('@/views/admin/Settings.vue'),
      meta: { requiresRole: ['admin'] },
    },
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: () => import('@/views/NotFound.vue'),
    },
  ],
})

router.beforeEach((to) => {
  const auth = useAuthStore()
  const config = useConfigStore()

  // Wiki-wide lockdown: when wiki_config.require_login is on, every route
  // except /login itself demands an authenticated session. Mirrors the
  // backend's CanAccess early-return so the UX matches what the API would
  // enforce anyway. wiki_config is eager-loaded in main.ts.
  const lockedDown = config.config?.require_login === true
  if (lockedDown && !auth.isAuthenticated && to.name !== 'login') {
    return { name: 'login', query: { redirect: to.fullPath } }
  }

  const needsAuth = to.meta.requiresAuth || (to.meta.requiresRole?.length ?? 0) > 0
  if (needsAuth && !auth.isAuthenticated) {
    return { name: 'login', query: { redirect: to.fullPath } }
  }
  if (to.meta.requiresRole && auth.role && !to.meta.requiresRole.includes(auth.role)) {
    return { name: 'not-found' }
  }
})

function joinPath(p: unknown): string {
  if (Array.isArray(p)) return p.join('/')
  if (typeof p === 'string') return p
  return ''
}

export default router
