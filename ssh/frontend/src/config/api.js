import axios from 'axios'

// Create axios instance with base URL from environment
const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json'
  }
})

// Add request interceptor to include auth token
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('auth_token')
    if (token) {
      config.headers['Authorization'] = `Basic ${token}`
    }
    
    // Debug logging for DELETE requests
    if (config.method === 'delete') {
      console.log('=== DELETE REQUEST DEBUG ===')
      console.log('URL:', config.url)
      console.log('Full URL:', config.baseURL + config.url)
      console.log('Method:', config.method)
      console.log('Headers:', config.headers)
      console.log('Auth token present:', !!token)
    }
    
    return config
  },
  (error) => {
    console.error('Request interceptor error:', error)
    return Promise.reject(error)
  }
)

// Add response interceptor to handle auth errors
api.interceptors.response.use(
  (response) => {
    return response
  },
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('auth_token')
      localStorage.removeItem('username')
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)

// API functions for relay server endpoints
export const apiService = {
  // Authentication
  login(credentials) {
    return api.post('/login', credentials)
  },

  logout() {
    return api.post('/logout')
  },

  // Agents
  getAgents() {
    return api.get('/api/agents')
  },

  addAgent(agentData) {
    return api.post('/api/agents', agentData)
  },

  deleteAgent(agentId) {
    return api.delete(`/api/agents/${agentId}`)
  },

  // Clients
  getClients() {
    return api.get('/api/clients')
  },

  deleteClient(clientId) {
    return api.delete(`/api/clients/${clientId}`)
  },

  // Connection Logs
  getConnectionLogs() {
    return api.get('/api/logs')
  },

  // Tunnel Logs (Database Queries)
  getTunnelLogs() {
    return api.get('/api/tunnel-logs')
  },

  // SSH Logs
  getSSHLogs() {
    return api.get('/api/ssh-logs')
  },

  // Log SSH Command
  logSSHCommand(data) {
    return api.post('/api/log-ssh', data)
  },

  // Log Database Query
  logQuery(data) {
    return api.post('/api/log-query', data)
  },

  // Health Check
  getHealth() {
    return api.get('/health')
  },

  // Server Settings
  getServerSettings() {
    return api.get('/api/settings')
  },

  updateServerSettings(settings) {
    return api.put('/api/settings', settings)
  },

  // Settings (new format for table-based settings)
  getSettings() {
    return api.get('/api/settings')
  },

  saveSettings(settings) {
    return api.put('/api/settings', settings)
  }
}

export default apiService