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
              <th>ACTIONS</th>
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
              <td>
                <div class="action-buttons">
                  <button 
                    class="action-btn setup-btn with-text" 
                    @click="openSetupModal(agent.id)"
                    title="Setup Agent">
                    <i class="fas fa-cog"></i>
                    <span>Setup</span>
                  </button>
                  <button 
                    class="action-btn delete-btn with-text" 
                    @click="deleteAgent(agent.id)"
                    title="Delete Agent">
                    <i class="fas fa-trash"></i>
                    <span>Delete</span>
                  </button>
                </div>
              </td>
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

    <!-- Setup Modal -->
    <div v-if="showSetupModal" class="modal-overlay" @click="closeSetupModal">
      <div class="modal setup-modal" @click.stop>
        <div class="modal-header">
          <h3>Setup Agent: {{ currentSetupAgentId }}</h3>
          <button class="btn-close" @click="closeSetupModal">
            <i class="fas fa-times"></i>
          </button>
        </div>
        
        <div class="modal-body">
          <div class="tab-content">
            <!-- Linux Tab -->
            <div class="setup-section">
              <h4><i class="fab fa-linux"></i> Agent Command Setup</h4>
              
              <div class="step">
                <h5>1. Download Agent Binary</h5>
                <div class="code-block">
                  <pre><code>wget http://your-server:8080/downloads/agent-linux
chmod +x agent-linux</code></pre>
                  <button class="copy-btn" @click="copyCommand('download-linux')">
                    <i class="fas fa-copy"></i>
                  </button>
                </div>
              </div>

              <div class="step">
                <h5>2. Setup Binary Location</h5>
                <div class="code-block">
                  <pre><code>mkdir -p bin
mv agent-linux bin/agent
chmod +x bin/agent</code></pre>
                  <button class="copy-btn" @click="copyCommand('setup-binary')">
                    <i class="fas fa-copy"></i>
                  </button>
                </div>
              </div>

              <div class="step">
                <h5>3. Run Agent Command</h5>
                <div class="code-block">
                  <pre><code>bin/agent -a {{ currentSetupAgentId }} -r ws://{{ serverSettings.serverIP }}:{{ serverSettings.serverPort }}/ws/agent</code></pre>
                  <button class="copy-btn" @click="copyCommand('run-linux')">
                    <i class="fas fa-copy"></i>
                  </button>
                </div>
              </div>
            </div>
          </div>

          <div class="setup-notes">
            <h4><i class="fas fa-info-circle"></i> Important Notes</h4>
            <ul>
              <li>Use command: <code>bin/agent -a {{ currentSetupAgentId }} -r ws://{{ serverSettings.serverIP }}:{{ serverSettings.serverPort }}/ws/agent</code></li>
              <li>Agent ID <code>{{ currentSetupAgentId }}</code> is automatically set from database</li>
              <li>Ensure the agent binary is located in <code>bin/</code> directory</li>
              <li>Make sure the agent can reach the WebSocket server at <code>ws://{{ serverSettings.serverIP }}:{{ serverSettings.serverPort }}/ws/agent</code></li>
              <li>Check firewall settings if connection fails</li>
              <li>Agent will appear as "connected" in this dashboard once running</li>
            </ul>
          </div>
        </div>
        
        <div class="modal-footer">
          <button class="btn btn-secondary" @click="closeSetupModal">
            Close
          </button>
        </div>
      </div>
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

    // Setup Modal Data
    const showSetupModal = ref(false)
    const currentSetupAgentId = ref('')
    
    // Server Settings Data  
    const serverSettings = ref({
      serverIP: '192.168.1.115',
      serverPort: 8080
    })

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

    // Setup Modal Functions
    const openSetupModal = (agentId) => {
      console.log('Opening Setup Modal for agent:', agentId)
      currentSetupAgentId.value = agentId
      showSetupModal.value = true
    }

    const closeSetupModal = () => {
      console.log('Closing Setup Modal...')
      showSetupModal.value = false
    }

    // Copy to clipboard functions
    const copyToClipboard = async (text) => {
      try {
        await navigator.clipboard.writeText(text)
        // Show toast notification
        const toast = document.createElement('div')
        toast.textContent = 'Copied to clipboard!'
        toast.style.cssText = `
          position: fixed;
          top: 20px;
          right: 20px;
          background: var(--color-success);
          color: white;
          padding: 0.75rem 1rem;
          border-radius: var(--radius-md);
          box-shadow: var(--shadow-lg);
          z-index: 10000;
          font-size: 0.875rem;
          font-weight: 500;
        `
        document.body.appendChild(toast)
        setTimeout(() => {
          document.body.removeChild(toast)
        }, 2000)
      } catch (err) {
        console.error('Failed to copy:', err)
        alert('Failed to copy to clipboard')
      }
    }

    // Copy command helper
    const copyCommand = (commandType) => {
      const commands = {
        'download-linux': 'wget http://your-server:8080/downloads/agent-linux\nchmod +x agent-linux',
        'setup-binary': 'mkdir -p bin\nmv agent-linux bin/agent\nchmod +x bin/agent',
        'run-linux': `bin/agent -a ${currentSetupAgentId.value} -r ws://${serverSettings.value.serverIP}:${serverSettings.value.serverPort}/ws/agent`
      }
      
      const command = commands[commandType]
      if (command) {
        copyToClipboard(command)
      }
    }

    // Delete Agent Function
    const deleteAgent = async (agentId) => {
      if (!confirm(`Are you sure you want to delete agent "${agentId}"?\n\nThis action cannot be undone.`)) {
        return
      }

      try {
        console.log('=== DELETING AGENT ===')
        console.log('Agent ID:', agentId)
        console.log('API URL:', `${import.meta.env.VITE_API_BASE_URL}/api/agents/${agentId}`)
        
        // Call API to delete agent
        const response = await apiService.deleteAgent(agentId)
        console.log('Delete response:', response)
        
        console.log(`Agent "${agentId}" deleted successfully from database`)
        alert(`Agent "${agentId}" has been deleted successfully!`)
        
        // Wait a moment then refresh agents list to ensure database sync
        console.log('Refreshing agents list after delete...')
        setTimeout(async () => {
          await fetchAgents()
          console.log('Agents list refreshed after delete')
        }, 500)
        
      } catch (error) {
        console.error('=== DELETE AGENT ERROR ===')
        console.error('Full error object:', error)
        console.error('Error response:', error.response)
        console.error('Error message:', error.message)
        console.error('Error status:', error.response?.status)
        console.error('Error data:', error.response?.data)
        
        let errorMessage = 'Failed to delete agent'
        if (error.response?.data?.error) {
          errorMessage = error.response.data.error
        } else if (error.response?.data?.message) {
          errorMessage = error.response.data.message
        } else if (error.message) {
          errorMessage = error.message
        }
        alert(`Failed to delete agent "${agentId}": ${errorMessage}`)
      }
    }

    // Load server settings for dynamic IP
    const loadServerSettings = async () => {
      try {
        console.log('=== LOADING SERVER SETTINGS ===')
        const response = await apiService.getSettings()
        console.log('Settings response:', response)
        
        if (response && response.success && response.data) {
          const settingsData = response.data
          console.log('Settings data:', settingsData)
          
          const newSettings = {
            serverIP: settingsData.server_ip || '192.168.1.115',
            serverPort: parseInt(settingsData.server_port) || 8080
          }
          
          console.log('New server settings:', newSettings)
          serverSettings.value = newSettings
          console.log('Updated serverSettings.value:', serverSettings.value)
        } else {
          console.warn('No settings data found, using defaults')
          // Use default values if no data returned
          serverSettings.value = {
            serverIP: '192.168.1.115',
            serverPort: 8080
          }
        }
      } catch (error) {
        console.error('Failed to load server settings:', error)
        // Keep default values
        serverSettings.value = {
          serverIP: '192.168.1.115',
          serverPort: 8080
        }
      }
    }

    onMounted(() => {
      fetchAgents()
      loadServerSettings()
      
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
      showSetupModal,
      currentSetupAgentId,
      serverSettings,
      refreshData,
      viewDetails,
      handlePageChange,
      openAddAgentModal,
      openSetupModal,
      closeSetupModal,
      copyToClipboard,
      copyCommand,
      deleteAgent,
      loadServerSettings
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

.action-buttons {
  display: flex;
  gap: 0.5rem;
  align-items: center;
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
  margin-right: 0;
}

.action-btn.with-text {
  width: auto;
  padding: 0.5rem 0.75rem;
  gap: 0.5rem;
  font-size: 0.875rem;
  font-weight: 500;
}

.action-btn.with-text span {
  font-size: 0.8rem;
}

.action-btn:hover {
  background: var(--primary-color);
  color: white;
  border-color: var(--primary-color);
}

.action-btn.setup-btn:hover {
  background: var(--info-color);
  color: white;
  border-color: var(--info-color);
}

.action-btn.delete-btn:hover {
  background: var(--danger-color);
  color: white;
  border-color: var(--danger-color);
}

.action-btn.danger:hover {
  background: var(--danger-color);
  color: white;
  border-color: var(--danger-color);
}

/* Setup Modal Styles */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.setup-modal {
  background: var(--surface-color);
  border-radius: var(--radius-lg);
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.15);
  width: 90%;
  max-width: 900px;
  max-height: 90vh;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.5rem;
  border-bottom: 1px solid var(--border-color);
  background: var(--surface-alt);
}

.modal-header h3 {
  margin: 0;
  color: var(--text-primary);
  font-size: 1.25rem;
  font-weight: 600;
}

.btn-close {
  background: none;
  border: none;
  font-size: 1.25rem;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 0.25rem;
  border-radius: var(--radius-sm);
  transition: all 0.2s ease;
}

.btn-close:hover {
  background: var(--surface-color);
  color: var(--text-primary);
}

.modal-body {
  padding: 1.5rem;
  overflow-y: auto;
  flex: 1;
}

.setup-tabs {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 2rem;
  border-bottom: 1px solid var(--border-color);
}

.tab-btn {
  padding: 0.75rem 1.5rem;
  border: none;
  background: none;
  color: var(--text-secondary);
  cursor: pointer;
  border-bottom: 2px solid transparent;
  transition: all 0.2s ease;
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.tab-btn:hover {
  color: var(--text-primary);
  background: var(--surface-alt);
}

.tab-btn.active {
  color: var(--color-primary);
  border-bottom-color: var(--color-primary);
}

.setup-section h4 {
  color: var(--text-primary);
  margin-bottom: 1.5rem;
  font-size: 1.25rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.step {
  margin-bottom: 2rem;
}

.step h5 {
  color: var(--text-primary);
  margin-bottom: 0.75rem;
  font-size: 1rem;
  font-weight: 600;
}

.step p {
  color: var(--text-secondary);
  margin-bottom: 0.75rem;
  line-height: 1.6;
}

.code-block {
  position: relative;
  background: var(--background-color);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  overflow: hidden;
}

.code-block pre {
  margin: 0;
  padding: 1rem;
  overflow-x: auto;
  font-family: 'Fira Code', 'Consolas', 'Monaco', monospace;
  font-size: 0.875rem;
  line-height: 1.5;
  color: var(--text-primary);
}

.code-block code {
  font-family: inherit;
}

.copy-btn {
  position: absolute;
  top: 0.5rem;
  right: 0.5rem;
  background: var(--surface-color);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  padding: 0.5rem;
  cursor: pointer;
  color: var(--text-secondary);
  transition: all 0.2s ease;
  font-size: 0.875rem;
}

.copy-btn:hover {
  background: var(--color-primary);
  color: white;
  border-color: var(--color-primary);
}

.setup-notes {
  background: var(--surface-alt);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  padding: 1.5rem;
  margin-top: 2rem;
}

.setup-notes h4 {
  color: var(--text-primary);
  margin-bottom: 1rem;
  font-size: 1rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.setup-notes ul {
  margin: 0;
  padding-left: 1.5rem;
  color: var(--text-secondary);
  line-height: 1.6;
}

.setup-notes li {
  margin-bottom: 0.5rem;
}

.setup-notes code {
  background: var(--background-color);
  padding: 0.2rem 0.4rem;
  border-radius: var(--radius-sm);
  font-family: 'Fira Code', 'Consolas', 'Monaco', monospace;
  font-size: 0.875rem;
  color: var(--color-primary);
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  padding: 1.5rem;
  border-top: 1px solid var(--border-color);
  background: var(--surface-alt);
}

.modal-footer .btn {
  padding: 0.75rem 1.5rem;
  border: none;
  border-radius: var(--radius-md);
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.modal-footer .btn-secondary {
  background: var(--surface-color);
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
}

.modal-footer .btn-secondary:hover {
  background: var(--border-color);
  color: var(--text-primary);
}
</style>
