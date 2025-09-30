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

  addClient(clientData) {
    return api.post('/api/clients', clientData)
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
  },

  // User Management
  getUsers() {
    return api.get('/api/users')
  },

  addUser(userData) {
    return api.post('/api/users', userData)
  },

  updateUser(userId, userData) {
    return api.put(`/api/users/${userId}`, userData)
  },

  deleteUser(userId) {
    return api.delete(`/api/users/${userId}`)
  },

  updateUserRole(userId, role) {
    return api.patch(`/api/users/${userId}/role`, { role })
  },

  // Project Management
  getProjects() {
    return api.get('/api/projects')
  },

  getUserProjects() {
    return api.get('/api/user/projects')
  },

  getUserProjectAgents(projectId) {
    return api.get(`/api/user/projects/${projectId}/agents`)
  },

  getProject(projectId) {
    return api.get(`/api/projects/${projectId}`)
  },

  addProject(projectData) {
    return api.post('/api/projects', projectData)
  },

  updateProject(projectId, projectData) {
    return api.put(`/api/projects/${projectId}`, projectData)
  },

  deleteProject(projectId) {
    return api.delete(`/api/projects/${projectId}`)
  },

  // Project Users
  getProjectUsers(projectId) {
    return api.get(`/api/projects/${projectId}/users`)
  },

  addUserToProject(projectId, userData) {
    return api.post(`/api/projects/${projectId}/users`, userData)
  },

  updateProjectUserRole(projectId, userId, role) {
    return api.patch(`/api/projects/${projectId}/users/${userId}`, { role })
  },

  removeUserFromProject(projectId, userId) {
    return api.delete(`/api/projects/${projectId}/users/${userId}`)
  },

  // Project Agents
  getProjectAgents(projectId) {
    return api.get(`/api/projects/${projectId}/agents`)
  },

  addAgentToProject(projectId, agentData) {
    return api.post(`/api/projects/${projectId}/agents`, agentData)
  },

  updateProjectAgent(projectId, agentId, agentData) {
    return api.patch(`/api/projects/${projectId}/agents/${agentId}`, agentData)
  },

  removeAgentFromProject(projectId, agentId) {
    return api.delete(`/api/projects/${projectId}/agents/${agentId}`)
  },

  // User Projects (for getting projects a user has access to)
  getUserProjects() {
    return api.get('/api/user/projects')
  },

  // SSH Management
  getSSHManagement() {
    return api.get('/api/ssh-management')
  },

  updateSSHManagement(agentId, sshManagement) {
    return api.put('/api/ssh-management', {
      agent_id: agentId,
      ssh_management: sshManagement
    })
  },

  // SSH Connections
  getSSHConnections() {
    return api.get('/api/ssh-connections')
  },

  addSSHConnection(sshData) {
    return api.post('/api/ssh-connections', sshData)
  },

  updateSSHConnection(id, sshData) {
    return api.put(`/api/ssh-connections/${id}`, sshData)
  },

  deleteSSHConnection(id) {
    return api.delete(`/api/ssh-connections/${id}`)
  }
}

export default apiService