import { createRouter, createWebHistory } from 'vue-router'
import HomeView from '../views/HomeView.vue'
import UploadView from '../views/UploadView.vue'
import LoginView from '../views/LoginView.vue'
import { useAuthStore } from '@/stores/auth'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    {
      path: '/',
      name: 'home',
      component: HomeView
    },
    {
      path: '/login',
      name: 'login',
      component: LoginView
    },
    {
      path: '/upload',
      name: 'upload',
      component: UploadView,
      meta: { requiresAuth: true } // 标记这个路由需要认证
    }
  ]
})

// 全局路由守卫
router.beforeEach((to, from, next) => {
  // This is a workaround to ensure the store is initialized before the guard is checked,
  // especially on a hard refresh.
  const authStore = useAuthStore()

  if (to.meta.requiresAuth && !authStore.isLoggedIn) {
    // 如果目标路由需要认证且用户未登录，则重定向到登录页
    next({ name: 'login' })
  } else {
    // 否则，正常放行
    next()
  }
})

export default router
