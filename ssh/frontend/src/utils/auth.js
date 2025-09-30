// Authentication and Authorization utilities

import { ref, computed } from 'vue'

// Global reactive state for current user
const currentUser = ref(null)
const isAuthenticated = ref(false)

// Initialize auth state from localStorage
const initAuth = () => {
  const token = localStorage.getItem('auth_token')
  const username = localStorage.getItem('user_name')
  const role = localStorage.getItem('user_role')
  
  if (token && username) {
    isAuthenticated.value = true
    currentUser.value = {
      username,
      role: role || 'user', // default to user if no role stored
      token
    }
  }
}

// Set user after login
const setUser = (userData) => {
  currentUser.value = userData
  isAuthenticated.value = true
  
  // Store in localStorage
  localStorage.setItem('auth_token', userData.token)
  localStorage.setItem('user_name', userData.username)
  localStorage.setItem('user_role', userData.role)
}

// Clear user data on logout
const clearUser = () => {
  currentUser.value = null
  isAuthenticated.value = false
  
  // Clear localStorage
  localStorage.removeItem('auth_token')
  localStorage.removeItem('user_name')
  localStorage.removeItem('user_role')
  // Keep logging_out flag until router redirect is complete
}

// Computed properties for role checking
const isAdmin = computed(() => {
  return currentUser.value?.role === 'admin'
})

const isUser = computed(() => {
  return currentUser.value?.role === 'user'
})

// Permission checking functions
const hasPermission = (permission) => {
  if (!currentUser.value) return false
  
  const role = currentUser.value.role
  
  // Admin permissions
  if (role === 'admin') {
    return true // Admin has all permissions
  }
  
  // User permissions
  if (role === 'user') {
    const userPermissions = [
      'view_dashboard',
      'view_agents',
      'view_clients', 
      'view_logs',
      'view_ssh_commands',
      'view_projects', // Add projects permission for users
      'view_ssh' // Add SSH management permission for users
    ]
    return userPermissions.includes(permission)
  }
  
  return false
}

// Route/page permissions
const canAccessPage = (pageName) => {
  if (!isAuthenticated.value) return false
  
  const role = currentUser.value?.role
  
  // Admin can access everything
  if (role === 'admin') return true
  
  // User permissions for specific pages
  const userPages = [
    'dashboard',
    'agents',
    'clients', 
    'logs',
    'queries',
    'sshLogs',
    'projects', // Allow users to access projects page
    'sshManagement' // Allow users to access SSH Management page
  ]
  
  // Admin-only pages
  const adminOnlyPages = [
    'userManagement',
    'remoteSSH',
    'settings'
  ]
  
  if (role === 'user') {
    return userPages.includes(pageName) && !adminOnlyPages.includes(pageName)
  }
  
  return false
}

// Menu visibility
const getVisibleMenuItems = () => {
  if (!isAuthenticated.value) return []
  
  const role = currentUser.value?.role
  
  // For regular users, show Projects and SSH Management
  if (role === 'user') {
    return [
      { name: 'Projects', route: 'projects', permission: 'view_projects', icon: 'fa-project-diagram' },
      { name: 'SSH Management', route: 'sshManagement', permission: 'view_ssh', icon: 'fa-key' }
    ]
  }
  
  // For admin users, show all menus except Remote SSH Management
  const allMenuItems = [
    { name: 'Agents', route: 'agents', permission: 'view_agents', icon: 'fa-server' },
    { name: 'Projects', route: 'projects', permission: 'manage_projects', icon: 'fa-project-diagram' },
    { name: 'SSH Management', route: 'sshManagement', permission: 'manage_ssh', icon: 'fa-key' },
    { name: 'History Client', route: 'clients', permission: 'view_clients', icon: 'fa-users' },
    { name: 'Connection Logs', route: 'logs', permission: 'view_logs', icon: 'fa-list-alt' },
    { name: 'Database Queries', route: 'queries', permission: 'view_logs', icon: 'fa-database' },
    { name: 'SSH Commands', route: 'sshLogs', permission: 'view_logs', icon: 'fa-terminal' },
    { name: 'User Management', route: 'userManagement', permission: 'manage_users', icon: 'fa-users-cog' },
    { name: 'Settings', route: 'settings', permission: 'manage_settings', icon: 'fa-cog' }
  ]
  
  return allMenuItems.filter(item => hasPermission(item.permission))
}

// Initialize auth on module load
initAuth()

export {
  currentUser,
  isAuthenticated,
  isAdmin,
  isUser,
  initAuth,
  setUser,
  clearUser,
  hasPermission,
  canAccessPage,
  getVisibleMenuItems
}