import { createRouter, createWebHistory } from 'vue-router'
import Gates from '../pages/Gates.vue'
import GateDetail from '../pages/GateDetail.vue'
import GateSettings from '../pages/GateSettings.vue'
import RequestDetail from '../pages/RequestDetail.vue'
import NotFound from '../pages/NotFound.vue'

declare module 'vue-router' {
  interface RouteMeta {
    title?: string
  }
}

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      redirect: '/gates',
    },
    {
      path: '/gates',
      name: 'gates',
      component: Gates,
      meta: { title: 'Gates' },
    },
    {
      path: '/gates/:id',
      name: 'gate-detail',
      component: GateDetail,
      meta: { title: 'Gate Detail' },
    },
    {
      path: '/gates/:id/settings',
      name: 'gate-settings',
      component: GateSettings,
      meta: { title: 'Gate Settings' },
    },
    {
      path: '/gates/:id/requests/:rid',
      name: 'request-detail',
      component: RequestDetail,
      meta: { title: 'Request Detail' },
    },
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: NotFound,
      meta: { title: 'Not Found' },
    },
  ],
})

// Keep the browser tab title in sync with the active route.
const BASE_TITLE = 'mroki hub'
router.afterEach((to) => {
  const title = to.meta.title
  document.title = title ? `${title} · ${BASE_TITLE}` : BASE_TITLE
})

export default router
