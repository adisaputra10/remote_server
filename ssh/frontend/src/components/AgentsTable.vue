<template>
  <div class="agents-table-container">
    <div class="table-header">
      <h2 class="table-title">Connected Agents</h2>
      <div class="header-actions">
        <button class="btn btn-success" @click="openAddAgentModal">
          <i class="fas fa-plus"></i>
          Add Agent
        </button>
        <button class="btn btn-primary" @click="refreshData">
          <i class="fas fa-sync-alt"></i>
          Refresh
        </button>
      </div>
    </div>
    
    <div v-if="loading" class="loading-state">
      <i class="fas fa-spinner fa-spin"></i>
      Loading agents...
    </div>
    
    <div v-else-if="error" class="error-state">
      <i class="fas fa-exclamation-triangle"></i>
      {{ error }}
    </div>
    
    <div v-else-if="allAgents.length === 0" class="empty-state">
      <i class="fas fa-server"></i>
      No agents data from API
      <br>
      <small>Check relay server connection and agents status</small>
    </div>
    
    <div v-else class="table-wrapper">
      <div class="table-container">
        <table class="table">
          <thead>
            <tr>
              <th>AGENT ID</th>
              <th>STATUS</th>
              <th>CONNECTED AT</th>
              <th>LAST PING</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="agent in paginatedAgents" :key="agent.id">
              <td>{{ agent.id }}</td>
              <td>
                <span :class="['badge', 'badge-' + agent.status]">
                  {{ agent.statusText }}
                </span>
              </td>
              <td>{{ agent.connectedAt }}</td>
              <td>{{ agent.lastPing }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      
      <Pagination
        :current-page="currentPage"
        :total-items="allAgents.length"
        :items-per-page="itemsPerPage"
        @page-changed="handlePageChange"
      />
    </div>
  </div>
</template>

<script>
import { ref, onMounted, computed } from 'vue'
import { apiService } from '../config/api.js'
import Pagination from './Pagination.vue'

export default {
  name: 'AgentsTable',
  components: {
    Pagination
  },
  emits: ['open-add-agent-modal'],
  setup(props, { emit }) {
    const allAgents = ref([])
    const loading = ref(false)
    const error = ref(null)
    const currentPage = ref(1)
    const itemsPerPage = ref(20)

    const paginatedAgents = computed(() => {
      const start = (currentPage.value - 1) * itemsPerPage.value
      const end = start + itemsPerPage.value
      return allAgents.value.slice(start, end)
    })

    const fetchAgents = async () => {
      try {
        loading.value = true
        error.value = null
        
        console.log('=== AgentsTable Debug Info ===')
        console.log('API Base URL:', import.meta.env.VITE_API_BASE_URL)
        console.log('Full API URL:', import.meta.env.VITE_API_BASE_URL + '/api/agents')
        console.log('Auth token:', localStorage.getItem('auth_token') ? 'Present' : 'Missing')
        
        const response = await apiService.getAgents()
        console.log('Agents response:', response.data)
        console.log('Response status:', response.status)
        console.log('Number of agents:', response.data ? response.data.length : 0)
        
        if (!response.data || response.data.length === 0) {
          console.warn('No agents data received from API')
          allAgents.value = []
          return
        }
        
        // Transform data to match the expected format
        allAgents.value = (response.data || []).map((agent, index) => {
          console.log('Raw agent data:', agent)
          
          let statusClass = 'warning'
          let statusText = 'unknown'
          
          // Check various status fields
          const status = agent.status || agent.state || agent.connection_status || 'unknown'
          const isConnected = agent.connected === true || 
                             agent.is_connected === true || 
                             agent.active === true ||
                             status.toLowerCase() === 'connected' ||
                             status.toLowerCase() === 'active' ||
                             status.toLowerCase() === 'online'
          
          if (isConnected) {
            statusClass = 'success'
            statusText = 'connected'
          } else {
            statusClass = 'danger'
            statusText = 'disconnected'
          }
          
          const transformedAgent = {
            id: agent.id || agent.agent_id || agent.name || `agent-${index + 1}`,
            status: statusClass,
            statusText: statusText,
            connectedAt: agent.connected_at || agent.connected_since || agent.last_seen || agent.created_at || 'Unknown',
            lastPing: agent.last_ping || agent.last_seen || agent.updated_at || agent.last_activity || 'Unknown'
          }
          
          console.log('Transformed agent:', transformedAgent)
          return transformedAgent
        })
        
        console.log('All transformed agents:', allAgents.value)
        
      } catch (err) {
        console.error('Error fetching agents:', err)
        console.error('API URL:', import.meta.env.VITE_API_BASE_URL + '/api/agents')
        console.error('Error details:', {
          status: err.response?.status,
          statusText: err.response?.statusText,
          data: err.response?.data,
          message: err.message
        })
        
        if (err.response?.status === 401) {
          error.value = 'Authentication required. Please login to access agents data.'
        } else if (err.response?.status === 404) {
          error.value = 'API endpoint not found. Please check relay server configuration.'
        } else if (err.code === 'ECONNREFUSED' || err.code === 'NETWORK_ERROR') {
          error.value = 'Cannot connect to relay server. Please check if server is running.'
        } else {
          error.value = `Failed to load agents data: ${err.message}`
        }
        allAgents.value = []
      } finally {
        loading.value = false
      }
    }

    const refreshData = () => {
      console.log('Refreshing agents data...')
      fetchAgents()
    }

    const viewDetails = (agentId) => {
      console.log('Viewing details for agent:', agentId)
      alert(`Agent Details: ${agentId}`)
    }

    const handlePageChange = (page) => {
      currentPage.value = page
    }

    const openAddAgentModal = () => {
      console.log('Opening Add Agent Modal...')
      emit('open-add-agent-modal')
    }

    onMounted(() => {
      fetchAgents()
      
      // Auto-refresh every 30 seconds
      setInterval(fetchAgents, 30000)
    })

    return {
      allAgents,
      paginatedAgents,
      loading,
      error,
      currentPage,
      itemsPerPage,
      refreshData,
      viewDetails,
      handlePageChange,
      openAddAgentModal
    }
  }
}
</script>

<style scoped>
.agents-table-container {
  padding: 1.5rem;
  height: 100%;
  display: flex;
  flex-direction: column;
}

.table-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
}

.header-actions {
  display: flex;
  gap: 0.75rem;
  align-items: center;
}

.table-title {
  font-size: 1.25rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.loading-state, .error-state, .empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  height: 300px;
  color: var(--text-secondary);
  font-size: 16px;
  flex: 1;
}

.error-state {
  color: var(--color-danger);
}

.loading-state i {
  font-size: 20px;
}

.table-wrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.table-container {
  flex: 1;
  overflow: auto;
}

.action-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  background: var(--surface-color);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.3s ease;
  margin-right: 0.5rem;
}

.action-btn:hover {
  background: var(--primary-color);
  color: white;
  border-color: var(--primary-color);
}
</style>
