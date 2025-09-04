import axios from 'axios'
import { useAuthStore } from '../stores/auth'

// Create axios instance
const api = axios.create({
  baseURL: '/api',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json'
  }
})

// Request interceptor
api.interceptors.request.use(
  (config) => {
    // Add auth token if available
    const authStore = useAuthStore()
    if (authStore.token) {
      config.headers.Authorization = `Bearer ${authStore.token}`
    }
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// Response interceptor
api.interceptors.response.use(
  (response) => {
    return response
  },
  (error) => {
    // Handle common errors
    console.log('API Error:', error.response?.status, error.config?.url, error.response?.data)
    
    if (error.response?.status === 401) {
      // For debugging: don't auto logout, just log the error
      console.warn('Unauthorized access to:', error.config?.url)
      
      // Only auto-logout for login endpoint errors
      if (error.config?.url?.includes('/auth/login')) {
        const authStore = useAuthStore()
        authStore.clearAuth()
        
        if (window.location.pathname !== '/login') {
          window.location.href = '/login'
        }
      }
    }
    return Promise.reject(error)
  }
)

// API methods
export default {
  // Authentication
  login(credentials) {
    return api.post('/auth/login', credentials)
  },

  logout() {
    return api.post('/auth/logout')
  },

  // Stats
  getStats() {
    return api.get('/stats')
  },

  // Command Logs
  getCommandLogs(params = {}) {
    return api.get('/logs', { params })
  },

  // Access Logs  
  getAccessLogs(params = {}) {
    return api.get('/access-logs', { params })
  },

  // Sessions
  getSessions(params = {}) {
    return api.get('/sessions', { params })
  },

  terminateSession(sessionId) {
    return api.delete(`/sessions/${sessionId}`)
  },

  // Agents
  getAgents(params = {}) {
    return api.get('/agents', { params })
  },

  // User Management
  getUsers() {
    return api.get('/users')
  },

  createUser(userData) {
    return api.post('/users', userData)
  },

  updateUser(id, userData) {
    return api.put(`/users?id=${id}`, userData)
  },

  deleteUser(id) {
    return api.delete(`/users?id=${id}`)
  },

  // User-Agent Assignments
  getUserAssignments(params = {}) {
    return api.get('/user-assignments', { params })
  },

  createUserAssignment(userData) {
    return api.post('/user-assignments', userData)
  },

  deleteUserAssignment(userId, agentId) {
    return api.delete(`/user-assignments?user_id=${userId}&agent_id=${agentId}`)
  },

  // System
  getSystemInfo() {
    return api.get('/system')
  }
}
