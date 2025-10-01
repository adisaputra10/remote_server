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
                    class="action-btn access-btn with-text" 
                    @click="openAccessModal(agent.id)"
                    title="Access Agent">
                    <i class="fas fa-terminal"></i>
                    <span>Access</span>
                  </button>
                  <button 
                    class="action-btn ssh-web-btn with-text" 
                    @click="openWebSSH(agent.id)"
                    title="Web SSH Terminal">
                    <i class="fas fa-globe"></i>
                    <span>Web SSH</span>
                  </button>
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
                <div v-if="isServerConfigured" class="code-block">
                  <pre><code>bin/agent -a {{ currentSetupAgentId }} -t {{ currentSetupAgentToken }} -r ws://{{ serverSettings.serverIP }}:{{ serverSettings.serverPort }}/ws/agent</code></pre>
                  <button class="copy-btn" @click="copyCommand('run-linux')">
                    <i class="fas fa-copy"></i>
                  </button>
                </div>
                <div v-else class="config-warning">
                  <i class="fas fa-exclamation-triangle"></i>
                  <span>Please configure server IP in Settings first before running agent command</span>
                </div>
              </div>
              
              <div class="step">
                <h5>4. Agent Authentication</h5>
                <div class="token-info">
                  <div class="token-row">
                    <span class="token-label">Agent ID:</span>
                    <code class="token-value">{{ currentSetupAgentId }}</code>
                    <button class="copy-small-btn" @click="copyToClipboard(currentSetupAgentId)">
                      <i class="fas fa-copy"></i>
                    </button>
                  </div>
                  <div class="token-row">
                    <span class="token-label">Token:</span>
                    <code class="token-value">{{ currentSetupAgentToken || 'No token available' }}</code>
                    <button class="copy-small-btn" @click="copyToClipboard(currentSetupAgentToken)" v-if="currentSetupAgentToken">
                      <i class="fas fa-copy"></i>
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div class="setup-notes">
            <h4><i class="fas fa-info-circle"></i> Important Notes</h4>
            <div v-if="!isServerConfigured" class="config-warning">
              <i class="fas fa-exclamation-triangle"></i>
              <span><strong>Server not configured:</strong> Please go to Settings to configure server IP address before using agent commands.</span>
            </div>
            <ul v-if="isServerConfigured">
              <li>Agent ID <code>{{ currentSetupAgentId }}</code> and Token <code>{{ currentSetupAgentToken }}</code> are required for authentication</li>
              <li>Full command: <code>bin/agent -a {{ currentSetupAgentId }} -t {{ currentSetupAgentToken }} -r ws://{{ serverSettings.serverIP }}:{{ serverSettings.serverPort }}/ws/agent</code></li>
              <li>Token validates agent identity against server database</li>
              <li>Ensure the agent binary is located in <code>bin/</code> directory</li>
              <li>Make sure the agent can reach the WebSocket server at <code>ws://{{ serverSettings.serverIP }}:{{ serverSettings.serverPort }}/ws/agent</code></li>
              <li>Agent will be rejected if token is invalid or missing</li>
              <li>Check firewall settings if connection fails</li>
            </ul>
            <ul v-else>
              <li>Go to <strong>Settings</strong> page to configure server IP address</li>
              <li>Server IP is required for agent connection commands</li>
              <li>Agent commands will be available after server configuration</li>
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

    <!-- Access Modal -->
    <div v-if="showAccessModal" class="modal-overlay" @click="closeAccessModal">
      <div class="modal access-modal" @click.stop>
        <div class="modal-header">
          <h3>Access Options for Agent: {{ currentAccessAgentId }}</h3>
          <button class="btn-close" @click="closeAccessModal">
            <i class="fas fa-times"></i>
          </button>
        </div>
        
        <div class="modal-body">
          <div class="access-table-container">
            <table class="access-table">
              <thead>
                <tr>
                  <th>Service Type</th>
                  <th>Command</th>
                  <th>Action</th>
                </tr>
              </thead>
              <tbody>
                <!-- SSH Access -->
                <tr>
                  <td>
                    <div class="service-info">
                      <i class="fas fa-terminal"></i>
                      <span>SSH Access</span>
                    </div>
                  </td>
                  <td>
                    <div class="command-container">
                      <code class="command-text">{{ generateSSHCommand() }}</code>
                    </div>
                  </td>
                  <td>
                    <button class="btn btn-copy" @click="copyToClipboard(generateSSHCommand())" title="Copy to clipboard">
                      <i class="fas fa-copy"></i>
                    </button>
                  </td>
                </tr>

                <!-- MySQL Tunnel -->
                <tr>
                  <td>
                    <div class="service-info">
                      <i class="fas fa-database"></i>
                      <span>MySQL Tunnel</span>
                    </div>
                  </td>
                  <td>
                    <div class="command-container">
                      <code class="command-text">{{ generateMySQLCommand() }}</code>
                    </div>
                  </td>
                  <td>
                    <button class="btn btn-copy" @click="copyToClipboard(generateMySQLCommand())" title="Copy to clipboard">
                      <i class="fas fa-copy"></i>
                    </button>
                  </td>
                </tr>

                <!-- PostgreSQL Tunnel -->
                <tr>
                  <td>
                    <div class="service-info">
                      <i class="fas fa-database"></i>
                      <span>PostgreSQL Tunnel</span>
                    </div>
                  </td>
                  <td>
                    <div class="command-container">
                      <code class="command-text">{{ generatePostgreSQLCommand() }}</code>
                    </div>
                  </td>
                  <td>
                    <button class="btn btn-copy" @click="copyToClipboard(generatePostgreSQLCommand())" title="Copy to clipboard">
                      <i class="fas fa-copy"></i>
                    </button>
                  </td>
                </tr>

                <!-- MongoDB Tunnel -->
                <tr>
                  <td>
                    <div class="service-info">
                      <i class="fas fa-leaf"></i>
                      <span>MongoDB Tunnel</span>
                    </div>
                  </td>
                  <td>
                    <div class="command-container">
                      <code class="command-text">{{ generateMongoDBCommand() }}</code>
                    </div>
                  </td>
                  <td>
                    <button class="btn btn-copy" @click="copyToClipboard(generateMongoDBCommand())" title="Copy to clipboard">
                      <i class="fas fa-copy"></i>
                    </button>
                  </td>
                </tr>

                <!-- Custom Tunnel -->
                <tr>
                  <td>
                    <div class="service-info">
                      <i class="fas fa-cogs"></i>
                      <span>Custom Tunnel</span>
                    </div>
                  </td>
                  <td>
                    <div class="command-container">
                      <div class="custom-inputs">
                        <input 
                          type="text" 
                          v-model="customLocalPort" 
                          placeholder="Local Port (e.g., 9999)" 
                          class="form-input"
                        />
                        <input 
                          type="text" 
                          v-model="customTargetHost" 
                          placeholder="Target Host:Port (e.g., localhost:80)" 
                          class="form-input"
                        />
                      </div>
                      <code class="command-text">{{ generateCustomCommand() }}</code>
                    </div>
                  </td>
                  <td>
                    <button class="btn btn-copy" @click="copyToClipboard(generateCustomCommand())" title="Copy to clipboard">
                      <i class="fas fa-copy"></i>
                    </button>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>

          <div class="access-note">
            <i class="fas fa-info-circle"></i>
            <p><strong>Note:</strong> Make sure to use your user token (-T parameter) when executing these commands. 
            Default tokens: <code>admin_token_2025_secure</code> for admin, <code>user_token_2025_access</code> for user.</p>
          </div>
        </div>
        
        <div class="modal-footer">
          <button class="btn btn-secondary" @click="closeAccessModal">
            Close
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { apiService } from '../config/api.js'
import Pagination from './Pagination.vue'

export default {
  name: 'AgentsTable',
  components: {
    Pagination
  },
  emits: ['open-add-agent-modal'],
  setup(props, { emit }) {
    const router = useRouter()
    const allAgents = ref([])
    const loading = ref(false)
    const error = ref(null)
    const currentPage = ref(1)
    const itemsPerPage = ref(20)

    // Setup Modal Data
    const showSetupModal = ref(false)
    const currentSetupAgentId = ref('')
    const currentSetupAgentToken = ref('')
    
    // Access Modal Data
    const showAccessModal = ref(false)
    const currentAccessAgentId = ref('')
    const customLocalPort = ref('9999')
    const customTargetHost = ref('localhost:80')
    
    // Server Settings Data (loaded from database via API)
    const serverSettings = ref({
      serverIP: '',
      serverPort: 8080
    })

    const paginatedAgents = computed(() => {
      const start = (currentPage.value - 1) * itemsPerPage.value
      const end = start + itemsPerPage.value
      return allAgents.value.slice(start, end)
    })

    const isServerConfigured = computed(() => {
      return serverSettings.value.serverIP && serverSettings.value.serverIP.trim() !== ''
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
            token: agent.token || '',
            status: statusClass,
            statusText: statusText,
            connectedAt: agent.connected_at || agent.connected_since || agent.last_seen || agent.created_at || 'Unknown',
            lastPing: agent.last_ping || agent.last_seen || agent.updated_at || agent.last_activity || 'Unknown'
          }
          
          console.log('Raw agent data:', agent)
          console.log('Agent token from raw:', agent.token)
          console.log('Transformed agent:', transformedAgent)
          console.log('Transformed agent token:', transformedAgent.token)
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
      console.log('All agents in allAgents.value:', allAgents.value)
      
      // Find agent data to get token
      const agent = allAgents.value.find(a => a.id === agentId)
      console.log('Found agent data:', agent)
      console.log('Agent ID match check:', allAgents.value.map(a => ({id: a.id, matches: a.id === agentId})))
      
      currentSetupAgentId.value = agentId
      currentSetupAgentToken.value = agent ? agent.token : ''
      showSetupModal.value = true
      
      console.log('Setup modal data:', {
        agentId: currentSetupAgentId.value,
        token: currentSetupAgentToken.value,
        agentFound: !!agent
      })
    }

    const closeSetupModal = () => {
      console.log('Closing Setup Modal...')
      showSetupModal.value = false
    }

    // Access Modal Functions
    const openAccessModal = (agentId) => {
      console.log('Opening Access Modal for agent:', agentId)
      currentAccessAgentId.value = agentId
      showAccessModal.value = true
    }

    const closeAccessModal = () => {
      console.log('Closing Access Modal...')
      showAccessModal.value = false
    }

    // Web SSH Functions
    const openWebSSH = (agentId) => {
      console.log('Opening Web SSH for agent:', agentId)
      // Open SSH web terminal in new tab dengan parameter agent ID
      const route = router.resolve({
        name: 'SSHWebTerminal',
        query: { agentId: agentId }
      })
      window.open(route.href, '_blank')
    }

    // Command Generation Functions
    const generateSSHCommand = () => {
      return `.\\bin\\universal-client.exe -T admin_token_2025_secure -u username -H target-server -a ${currentAccessAgentId.value}`
    }

    const generateMySQLCommand = () => {
      return `.\\bin\\universal-client.exe -T admin_token_2025_secure -L :3306 -t localhost:3306 -a ${currentAccessAgentId.value}`
    }

    const generatePostgreSQLCommand = () => {
      return `.\\bin\\universal-client.exe -T admin_token_2025_secure -L :5432 -t localhost:5432 -a ${currentAccessAgentId.value}`
    }

    const generateMongoDBCommand = () => {
      return `.\\bin\\universal-client.exe -T admin_token_2025_secure -L :27017 -t localhost:27017 -a ${currentAccessAgentId.value}`
    }

    const generateCustomCommand = () => {
      return `.\\bin\\universal-client.exe -T admin_token_2025_secure -L :${customLocalPort.value} -t ${customTargetHost.value} -a ${currentAccessAgentId.value}`
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
        'run-linux': `bin/agent -a ${currentSetupAgentId.value} -t ${currentSetupAgentToken.value} -r ws://${serverSettings.value.serverIP}:${serverSettings.value.serverPort}/ws/agent`
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
        console.log('Settings API response:', response)
        console.log('Response data:', response.data)
        
        // Backend returns { success: true, data: {...} }, axios wraps it in response.data
        if (response.data && response.data.success && response.data.data) {
          const settingsData = response.data.data
          console.log('Settings data from database:', settingsData)
          
          const newSettings = {
            serverIP: settingsData.server_ip || '',
            serverPort: parseInt(settingsData.server_port) || 8080
          }
          
          console.log('New server settings:', newSettings)
          serverSettings.value = newSettings
          console.log('Updated serverSettings.value:', serverSettings.value)
        } else {
          console.warn('No settings data found from API')
          throw new Error('No settings data returned from server')
        }
      } catch (error) {
        console.error('Failed to load server settings:', error)
        // Don't use hardcoded fallback, let user know they need to configure
        console.warn('Using empty server settings - user needs to configure in Settings')
        serverSettings.value = {
          serverIP: '',
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
      currentSetupAgentToken,
      showAccessModal,
      currentAccessAgentId,
      customLocalPort,
      customTargetHost,
      serverSettings,
      isServerConfigured,
      refreshData,
      viewDetails,
      handlePageChange,
      openAddAgentModal,
      openSetupModal,
      closeSetupModal,
      openAccessModal,
      closeAccessModal,
      openWebSSH,
      generateSSHCommand,
      generateMySQLCommand,
      generatePostgreSQLCommand,
      generateMongoDBCommand,
      generateCustomCommand,
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

.config-warning {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: var(--color-danger);
  font-style: italic;
  padding: 0.75rem;
  background: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
  border-radius: var(--radius-sm);
  margin-bottom: 1rem;
}

.config-warning strong {
  font-weight: 600;
}

/* Access Modal Styles */
.access-modal {
  background: var(--surface-color);
  border-radius: var(--radius-lg);
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.15);
  width: 95%;
  max-width: 1200px;
  max-height: 90vh;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.access-table-container {
  overflow-x: auto;
  margin: 1rem 0;
}

.access-table {
  width: 100%;
  border-collapse: collapse;
  background: var(--surface-color);
}

.access-table th,
.access-table td {
  padding: 1rem;
  text-align: left;
  border-bottom: 1px solid var(--border-color);
}

.access-table th {
  background: var(--surface-alt);
  font-weight: 600;
  color: var(--text-primary);
  font-size: 0.875rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.service-info {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  font-weight: 500;
}

.service-info i {
  font-size: 1.25rem;
  width: 1.5rem;
  text-align: center;
}

.fa-terminal { color: #10b981; }
.fa-database { color: #3b82f6; }
.fa-leaf { color: #059669; }
.fa-cogs { color: #f59e0b; }

.command-container {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.command-text {
  background: var(--color-dark);
  color: var(--color-light);
  padding: 0.75rem;
  border-radius: var(--radius-sm);
  font-family: 'Courier New', monospace;
  font-size: 0.875rem;
  word-break: break-all;
  display: block;
  white-space: pre-wrap;
}

.custom-inputs {
  display: flex;
  gap: 0.5rem;
  margin-bottom: 0.5rem;
}

.form-input {
  flex: 1;
  padding: 0.5rem;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  background: var(--surface-color);
  color: var(--text-primary);
  font-size: 0.875rem;
}

.form-input:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.15);
}

.btn-copy {
  background: var(--color-primary);
  color: white;
  border: none;
  padding: 0.5rem 0.75rem;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 2.5rem;
}

.btn-copy:hover {
  background: #2563eb;
  transform: translateY(-1px);
}

.access-note {
  display: flex;
  align-items: flex-start;
  gap: 0.75rem;
  padding: 1rem;
  background: rgba(59, 130, 246, 0.1);
  border: 1px solid rgba(59, 130, 246, 0.3);
  border-radius: var(--radius-sm);
  margin-top: 1rem;
}

.access-note i {
  color: var(--color-primary);
  margin-top: 0.125rem;
}

.access-note p {
  margin: 0;
  color: var(--text-secondary);
  font-size: 0.875rem;
  line-height: 1.5;
}

.access-note code {
  background: rgba(0, 0, 0, 0.1);
  padding: 0.125rem 0.25rem;
  border-radius: 0.25rem;
  font-family: 'Courier New', monospace;
  font-size: 0.8rem;
}

.access-btn {
  background: var(--color-success);
  color: white;
}

.access-btn:hover {
  background: #059669;
}

.ssh-web-btn {
  background: #3b82f6;
  color: white;
}

.ssh-web-btn:hover {
  background: #2563eb;
}
</style>
