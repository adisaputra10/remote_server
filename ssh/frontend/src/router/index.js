import { createRouter, createWebHistory } from 'vue-router'
import LoginNew from '../views/LoginNew.vue'
import Dashboard from '../views/Dashboard.vue'

const routes = [
  {
    path: '/',
    redirect: (to) => {
      // Check if user is authenticated
      const isAuthenticated = localStorage.getItem('auth_token')
      return isAuthenticated ? '/dashboard' : '/login'
    }
  },
  {
    path: '/login',
    name: 'Login',
    component: LoginNew
  },
  {
    path: '/dashboard',
    name: 'Dashboard',
    component: Dashboard,
    meta: { requiresAuth: true }
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to, from, next) => {
  const isAuthenticated = localStorage.getItem('auth_token')
  
  if (to.matched.some(record => record.meta.requiresAuth)) {
    if (!isAuthenticated) {
      next('/login')
    } else {
      next()
    }
  } else {
    next()
  }
})

export default router