import { createRouter, createWebHistory } from 'vue-router'
import LoginNew from '../views/LoginNew.vue'
import Dashboard from '../views/Dashboard.vue'
import SSHWebTerminal from '../components/SSHWebTerminal.vue'

const routes = [
  {
    path: '/',
    redirect: (to) => {
      // Check if user is authenticated
      const isAuthenticated = localStorage.getItem('auth_token')
      const isLoggingOut = localStorage.getItem('logging_out')
      
      // Clear logging out flag if it exists
      if (isLoggingOut) {
        localStorage.removeItem('logging_out')
      }
      
      // Development bypass - only enable if explicitly needed
      // Comment out for better security testing
      /*
      if (!isAuthenticated && import.meta.env.DEV && !isLoggingOut) {
        const credentials = btoa('admin:admin123')
        localStorage.setItem('auth_token', credentials)
        localStorage.setItem('user_name', 'admin')
        localStorage.setItem('user_role', 'admin')
        return '/dashboard'
      }
      */
      
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
  },
  {
    path: '/ssh-terminal',
    name: 'SSHWebTerminal',
    component: SSHWebTerminal,
    meta: { requiresAuth: true }
  }
]

const router = createRouter({
  history: createWebHistory(),
  routes
})

router.beforeEach((to, from, next) => {
  const isAuthenticated = localStorage.getItem('auth_token')
  const isLoggingOut = localStorage.getItem('logging_out')
  
  // Clear logging out flag if it exists
  if (isLoggingOut) {
    localStorage.removeItem('logging_out')
  }
  
  // Development bypass - only auto login if specifically requested
  // Comment out the auto-login for better security testing
  /*
  if (!isAuthenticated && import.meta.env.DEV && !isLoggingOut) {
    const credentials = btoa('admin:admin123')
    localStorage.setItem('auth_token', credentials)
    localStorage.setItem('user_name', 'admin')
    localStorage.setItem('user_role', 'admin')
    isAuthenticated = credentials
  }
  */
  
  if (to.matched.some(record => record.meta.requiresAuth)) {
    if (!isAuthenticated) {
      console.log('Route requires auth but user not authenticated, redirecting to login')
      next('/login')
    } else {
      next()
    }
  } else {
    next()
  }
})

export default router