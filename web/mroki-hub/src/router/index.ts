import { createRouter, createWebHistory } from 'vue-router'
import Gates from '../pages/Gates.vue'
import GateDetail from '../pages/GateDetail.vue'
import RequestDetail from '../pages/RequestDetail.vue'
import NotFound from '../pages/NotFound.vue'

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
    },
    {
      path: '/gates/:id',
      name: 'gate-detail',
      component: GateDetail,
    },
    {
      path: '/gates/:id/requests/:rid',
      name: 'request-detail',
      component: RequestDetail,
    },
    {
      path: '/:pathMatch(.*)*',
      name: 'not-found',
      component: NotFound,
    },
  ],
})

export default router
