<template>
  <div class="dashboard">
    <!-- Header -->
    <header class="dashboard-header">
      <div class="dashboard-title-wrapper">
        <button class="mobile-menu-toggle" @click="toggleSidebar">
          <i class="fas fa-bars"></i>
        </button>
        <h1 class="dashboard-title">Welcome back, Admin</h1>
      </div>
      
      <div class="header-actions">
        <button class="theme-toggle" @click="toggleTheme">
          <i class="fas" :class="isDark ? 'fa-sun' : 'fa-moon'"></i>
          <span>Theme</span>
        </button>
        
        <div class="user-menu">
          <button class="user-button" @click="toggleUserMenu">
            <div class="user-avatar">A</div>
            <span>Admin User</span>
            <i class="fas fa-chevron-down"></i>
          </button>
          <div v-show="showUserMenu" class="dropdown-menu">
            <a href="#" class="dropdown-item" @click="showToast('Profile page would open here', 'info')">
              <i class="fas fa-user-circle"></i>
              <span>My Profile</span>
            </a>
            <a href="#" class="dropdown-item" @click="showToast('Settings page would open here', 'info')">
              <i class="fas fa-cog"></i>
              <span>Settings</span>
            </a>
            <a href="#" class="dropdown-item" @click="logout">
              <i class="fas fa-sign-out-alt"></i>
              <span>Logout</span>
            </a>
          </div>
        </div>
      </div>
    </header>

    <!-- Main Content -->
    <div class="dashboard-content">
      <!-- Sidebar -->
      <nav class="sidebar" :class="{ 'open': sidebarOpen }">
        <ul class="sidebar-menu">
          <li class="sidebar-item">
            <a href="#" class="sidebar-link" :class="{ 'active': activeTab === 'agents' }" @click="switchTab('agents')">
              <i class="fas fa-server"></i>
              <span>Agents</span>
            </a>
          </li>
          <li class="sidebar-item">
            <a href="#" class="sidebar-link" :class="{ 'active': activeTab === 'clients' }" @click="switchTab('clients')">
              <i class="fas fa-users"></i>
              <span>Clients</span>
            </a>
          </li>
          <li class="sidebar-item">
            <a href="#" class="sidebar-link" :class="{ 'active': activeTab === 'logs' }" @click="switchTab('logs')">
              <i class="fas fa-list-alt"></i>
              <span>Connection Logs</span>
            </a>
          </li>
          <li class="sidebar-item">
            <a href="#" class="sidebar-link" :class="{ 'active': activeTab === 'database' }" @click="switchTab('database')">
              <i class="fas fa-database"></i>
              <span>Database Queries</span>
            </a>
          </li>
          <li class="sidebar-item">
            <a href="#" class="sidebar-link" :class="{ 'active': activeTab === 'ssh' }" @click="switchTab('ssh')">
              <i class="fas fa-terminal"></i>
              <span>SSH Commands</span>
            </a>
          </li>
        </ul>
      </nav>

      <!-- Main Content Area -->
      <main class="main-content">
        <!-- Statistics Cards -->
        <div class="stats-grid">
          <div class="stat-card">
            <div class="stat-header">
              <span class="stat-title">Connected Agents</span>
              <div class="stat-icon blue">
                <i class="fas fa-server"></i>
              </div>
            </div>
            <div class="stat-value">{{ stats.connectedAgents }}</div>
            <div class="stat-change positive">
             
            </div>
          </div>
          
          <div class="stat-card">
            <div class="stat-header">
              <span class="stat-title">Active Clients</span>
              <div class="stat-icon green">
                <i class="fas fa-users"></i>
              </div>
            </div>
            <div class="stat-value">{{ stats.activeClients }}</div>
            <div class="stat-change positive">
              
            </div>
          </div>
          
          <div class="stat-card">
            <div class="stat-header">
              <span class="stat-title">Total Logs</span>
              <div class="stat-icon purple">
                <i class="fas fa-list-alt"></i>
              </div>
            </div>
            <div class="stat-value">{{ stats.totalConnections }}</div>
            <div class="stat-change positive">
          
            </div>
          </div>
        </div>

        <!-- Tab Content -->
        <div class="tab-content">
          <AgentsTable v-if="activeTab === 'agents'" @open-add-agent-modal="openAddAgentModal" />
          <ClientsTable v-if="activeTab === 'clients'" />
          <LogsTable v-if="activeTab === 'logs'" />
          <QueriesTable v-if="activeTab === 'database'" />
          <SSHLogsTable v-if="activeTab === 'ssh'" />
        </div>
      </main>
    </div>

    <!-- Add Agent Modal -->
    <div v-if="showAddAgentModal" class="modal-overlay" @click="closeAddAgentModal">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h3><i class="fas fa-plus-circle"></i> Add New Agent</h3>
          <button class="modal-close" @click="closeAddAgentModal">
            <i class="fas fa-times"></i>
          </button>
        </div>
        
        <form @submit.prevent="submitAgent" class="agent-form">
          <div class="form-group">
            <label for="agentId">Agent ID</label>
            <input 
              type="text" 
              id="agentId" 
              v-model="newAgent.agentId" 
              placeholder="Enter agent ID (e.g., agent-server-01)"
              required
              class="form-input"
            />
            <small class="form-help">Unique identifier for the agent</small>
          </div>
          
          <div class="form-group">
            <label for="agentToken">Agent Token</label>
            <input 
              type="password" 
              id="agentToken" 
              v-model="newAgent.token" 
              placeholder="Enter agent authentication token"
              required
              class="form-input"
            />
            <small class="form-help">Authentication token for agent connection</small>
          </div>
          
          <div class="form-actions">
            <button type="button" @click="closeAddAgentModal" class="btn btn-secondary">
              <i class="fas fa-times"></i>
              Cancel
            </button>
            <button type="submit" class="btn btn-primary" :disabled="addingAgent">
              <i class="fas" :class="addingAgent ? 'fa-spinner fa-spin' : 'fa-plus'"></i>
              {{ addingAgent ? 'Adding...' : 'Add Agent' }}
            </button>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<script>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import AgentsTable from '../components/AgentsTable.vue'
import ClientsTable from '../components/ClientsTable.vue'
import LogsTable from '../components/LogsTable.vue'
import QueriesTable from '../components/QueriesTable.vue'
import SSHLogsTable from '../components/SSHLogsTable.vue'
import { apiService } from '../config/api.js'

export default {
  name: 'Dashboard',
  components: {
    AgentsTable,
    ClientsTable,
    LogsTable,
    QueriesTable,
    SSHLogsTable
  },
  setup() {
    const router = useRouter()
    const activeTab = ref('agents')
    const sidebarOpen = ref(false)
    const showUserMenu = ref(false)
    const isDark = ref(false)
    
    const stats = ref({
      connectedAgents: 0,
      activeClients: 0,
      totalConnections: 0,
      dataTransferred: '0GB'
    })

    // Add Agent Modal State
    const showAddAgentModal = ref(false)
    const addingAgent = ref(false)
    const newAgent = ref({
      agentId: '',
      token: ''
    })

    const fetchDashboardStats = async () => {
      try {
        console.log('=== FETCHING DASHBOARD STATS FROM API ===')
        
        // Fetch Agents data
        console.log('Fetching agents...')
        const agentsResponse = await apiService.getAgents()
        const connectedAgents = (agentsResponse.data || []).filter(agent => 
          agent.status === 'connected' || agent.active === true
        ).length
        console.log('Connected Agents:', connectedAgents)
        
        // Fetch Clients data
        console.log('Fetching clients...')
        const clientsResponse = await apiService.getClients()
        const activeClients = (clientsResponse.data || []).filter(client => 
          client.status === 'connected' || client.active === true
        ).length
        console.log('Active Clients:', activeClients)
        
        // Fetch Connection Logs data for total count
        console.log('Fetching connection logs...')
        const logsResponse = await apiService.getConnectionLogs()
        const totalConnections = (logsResponse.data || []).length
        console.log('Total Logs:', totalConnections)
        
        // Update stats
        stats.value = {
          connectedAgents,
          activeClients,
          totalConnections,
          dataTransferred: '0GB' // We can calculate this later if needed
        }
        
        console.log('=== FINAL DASHBOARD STATS ===')
        console.log('Stats updated:', stats.value)
        console.log('✅ ALL DATA FROM API - NO DUMMY DATA')
        
      } catch (error) {
        console.error('=== DASHBOARD STATS API ERROR ===')
        console.error('Error fetching dashboard stats:', error)
        
        // Keep stats at 0 if API fails - no dummy fallback
        stats.value = {
          connectedAgents: 0,
          activeClients: 0,
          totalConnections: 0,
          dataTransferred: '0GB'
        }
        console.log('Using zero values due to API error - NO DUMMY DATA')
      }
    }

    const toggleSidebar = () => {
      sidebarOpen.value = !sidebarOpen.value
    }

    const toggleUserMenu = () => {
      showUserMenu.value = !showUserMenu.value
    }

    const toggleTheme = () => {
      isDark.value = !isDark.value
      const html = document.documentElement
      if (isDark.value) {
        html.setAttribute('data-theme', 'dark')
        localStorage.setItem('theme', 'dark')
      } else {
        html.removeAttribute('data-theme')
        localStorage.setItem('theme', 'light')
      }
    }

    const switchTab = (tab) => {
      activeTab.value = tab
      showUserMenu.value = false
    }

    // Add Agent Modal Functions
    const openAddAgentModal = () => {
      console.log('Opening Add Agent Modal...')
      showAddAgentModal.value = true
      // Reset form
      newAgent.value = {
        agentId: '',
        token: ''
      }
    }

    const closeAddAgentModal = () => {
      console.log('Closing Add Agent Modal...')
      showAddAgentModal.value = false
      newAgent.value = {
        agentId: '',
        token: ''
      }
    }

    const submitAgent = async () => {
      try {
        addingAgent.value = true
        console.log('=== ADDING NEW AGENT ===')
        console.log('Agent ID:', newAgent.value.agentId)
        console.log('Token:', newAgent.value.token ? '[HIDDEN]' : '[EMPTY]')

        // Validate form
        if (!newAgent.value.agentId.trim()) {
          alert('Please enter Agent ID')
          return
        }
        if (!newAgent.value.token.trim()) {
          alert('Please enter Agent Token')
          return
        }

        // TODO: Call API to add agent
        // For now, just simulate success
        console.log('Simulating agent addition...')
        await new Promise(resolve => setTimeout(resolve, 2000))
        
        console.log('✅ Agent added successfully!')
        alert(`Agent "${newAgent.value.agentId}" added successfully!`)
        
        // Close modal and refresh stats
        closeAddAgentModal()
        fetchDashboardStats()
        
      } catch (error) {
        console.error('=== ADD AGENT ERROR ===')
        console.error('Error adding agent:', error)
        alert(`Failed to add agent: ${error.message}`)
      } finally {
        addingAgent.value = false
      }
    }

    const showToast = (message, type) => {
      console.log(`Toast: ${message} (${type})`)
    }

    const logout = async () => {
      try {
        await apiService.logout()
      } catch (error) {
        console.error('Logout error:', error)
        // Continue with logout even if API call fails
      } finally {
        localStorage.removeItem('auth_token')
        localStorage.removeItem('user_name')
        router.push('/login')
      }
    }

    onMounted(() => {
      const savedTheme = localStorage.getItem('theme')
      if (savedTheme === 'dark') {
        isDark.value = true
        document.documentElement.setAttribute('data-theme', 'dark')
      }
      
      // Fetch initial dashboard stats from API
      console.log('=== DASHBOARD MOUNTED - FETCHING STATS FROM API ===')
      fetchDashboardStats()
      
      // Auto-refresh dashboard stats every 10 seconds
      setInterval(() => {
        console.log('=== AUTO-REFRESH DASHBOARD STATS ===')
        fetchDashboardStats()
      }, 10000)
    })

    return {
      activeTab,
      sidebarOpen,
      showUserMenu,
      isDark,
      stats,
      fetchDashboardStats,
      // Add Agent Modal
      showAddAgentModal,
      addingAgent,
      newAgent,
      openAddAgentModal,
      closeAddAgentModal,
      submitAgent,
      // Other functions
      toggleSidebar,
      toggleUserMenu,
      toggleTheme,
      switchTab,
      showToast,
      logout
    }
  }
}
</script>

<style scoped>
.dashboard {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
  background-color: var(--background-color);
}

.dashboard-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem 2rem;
  background: var(--surface-color);
  border-bottom: 1px solid var(--border-color);
  box-shadow: var(--shadow-sm);
  position: sticky;
  top: 0;
  z-index: 100;
}

.dashboard-title-wrapper {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.mobile-menu-toggle {
  display: none;
  background: none;
  border: none;
  font-size: 1.25rem;
  color: var(--text-primary);
  cursor: pointer;
  padding: 0.5rem;
  border-radius: var(--radius-md);
  transition: all 0.3s ease;
}

.mobile-menu-toggle:hover {
  background: var(--surface-alt);
}

.dashboard-title {
  font-size: 1.5rem;
  font-weight: 700;
  background: var(--primary-gradient);
  background-clip: text;
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  margin: 0;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 1rem;
}

.theme-toggle {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  background: none;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  color: var(--text-primary);
  cursor: pointer;
  transition: all 0.3s ease;
}

.theme-toggle:hover {
  background: var(--surface-alt);
  border-color: var(--primary-color);
}

.user-menu {
  position: relative;
}

.user-button {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.5rem 1rem;
  background: none;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  color: var(--text-primary);
  cursor: pointer;
  transition: all 0.3s ease;
}

.user-button:hover {
  background: var(--surface-alt);
  border-color: var(--primary-color);
}

.user-avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: var(--primary-gradient);
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-weight: 600;
  font-size: 0.875rem;
}

.dropdown-menu {
  position: absolute;
  top: 100%;
  right: 0;
  margin-top: 0.5rem;
  background: var(--surface-color);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  min-width: 200px;
  overflow: hidden;
  z-index: 1000;
}

.dropdown-item {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 1rem;
  color: var(--text-primary);
  text-decoration: none;
  transition: all 0.3s ease;
}

.dropdown-item:hover {
  background: var(--surface-alt);
}

.dashboard-content {
  display: flex;
  flex: 1;
  overflow: hidden;
}

.sidebar {
  width: 250px;
  background: var(--surface-color);
  border-right: 1px solid var(--border-color);
  padding: 1.5rem 0;
  transition: all 0.3s ease;
}

.sidebar-menu {
  list-style: none;
  padding: 0;
  margin: 0;
}

.sidebar-item {
  margin-bottom: 0.25rem;
}

.sidebar-link {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.75rem 1.5rem;
  color: var(--text-secondary);
  text-decoration: none;
  transition: all 0.3s ease;
  position: relative;
}

.sidebar-link::before {
  content: '';
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 4px;
  background: var(--primary-color);
  opacity: 0;
  transition: all 0.3s ease;
}

.sidebar-link:hover,
.sidebar-link.active {
  color: var(--primary-color);
  background: rgba(102, 126, 234, 0.1);
}

.sidebar-link.active::before {
  opacity: 1;
}

.sidebar-link i {
  width: 20px;
  text-align: center;
  font-size: 1.125rem;
}

.main-content {
  flex: 1;
  padding: 2rem;
  overflow-y: auto;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
  gap: 1.5rem;
  margin-bottom: 2rem;
}

.stat-card {
  background: var(--surface-color);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  padding: 1.5rem;
  box-shadow: var(--shadow);
  transition: all 0.3s ease;
}

.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-lg);
}

.stat-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1rem;
}

.stat-title {
  font-size: 0.875rem;
  color: var(--text-secondary);
  font-weight: 500;
}

.stat-icon {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-lg);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1.25rem;
  color: white;
}

.stat-icon.blue {
  background: var(--info-color);
}

.stat-icon.green {
  background: var(--success-color);
}

.stat-icon.purple {
  background: var(--purple-color);
}

.stat-icon.orange {
  background: var(--warning-color);
}

.stat-value {
  font-size: 2rem;
  font-weight: 700;
  color: var(--text-primary);
  margin-bottom: 0.5rem;
}

.stat-change {
  display: flex;
  align-items: center;
  gap: 0.25rem;
  font-size: 0.875rem;
}

.stat-change.positive {
  color: var(--success-color);
}

.tab-content {
  background: var(--surface-color);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  overflow: hidden;
  box-shadow: var(--shadow);
}

/* Add Agent Modal Styles */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.7);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 1rem;
}

.modal-content {
  background: var(--surface-color);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-color);
  width: 100%;
  max-width: 500px;
  max-height: 90vh;
  overflow-y: auto;
  box-shadow: 0 10px 25px rgba(0, 0, 0, 0.15);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.5rem;
  border-bottom: 1px solid var(--border-color);
}

.modal-header h3 {
  margin: 0;
  color: var(--text-primary);
  font-size: 1.25rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.modal-close {
  background: none;
  border: none;
  font-size: 1.25rem;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 0.5rem;
  border-radius: var(--radius-md);
  transition: all 0.3s ease;
}

.modal-close:hover {
  background: var(--surface-alt);
  color: var(--text-primary);
}

.agent-form {
  padding: 1.5rem;
}

.form-group {
  margin-bottom: 1.5rem;
}

.form-group label {
  display: block;
  margin-bottom: 0.5rem;
  color: var(--text-primary);
  font-weight: 500;
}

.form-input {
  width: 100%;
  padding: 0.75rem;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  background: var(--surface-alt);
  color: var(--text-primary);
  font-size: 0.9rem;
  transition: all 0.3s ease;
  box-sizing: border-box;
}

.form-input:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.form-help {
  display: block;
  margin-top: 0.25rem;
  font-size: 0.8rem;
  color: var(--text-secondary);
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  margin-top: 2rem;
  padding-top: 1rem;
  border-top: 1px solid var(--border-color);
}

@media (max-width: 768px) {
  .mobile-menu-toggle {
    display: block;
  }

  .dashboard-header {
    padding: 1rem;
  }

  .sidebar {
    position: fixed;
    left: -250px;
    top: 0;
    height: 100vh;
    z-index: 1000;
    box-shadow: var(--shadow-lg);
  }

  .sidebar.open {
    left: 0;
  }

  .main-content {
    padding: 1rem;
  }

  .stats-grid {
    grid-template-columns: 1fr;
  }

  .header-actions {
    gap: 0.5rem;
  }

  .theme-toggle span,
  .user-button span {
    display: none;
  }
}

@media (max-width: 480px) {
  .dashboard-title {
    font-size: 1.25rem;
  }

  .stat-card {
    padding: 1rem;
  }

  .stat-value {
    font-size: 1.5rem;
  }
}
</style>
