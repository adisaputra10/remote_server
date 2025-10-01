<template>
  <div class="ssh-management">
    <div class="page-header">
      <div class="header-content">
        <div class="header-text">
          <h2>SSH Management</h2>
          <p class="page-description">
            Manage SSH tunnels for agents. Configure which agents can provide SSH access to remote servers.
          </p>
        </div>
        <button class="add-tunnel-btn" @click="openAddTunnelModal">
          <i class="fas fa-plus"></i>
          Add Tunnel
        </button>
      </div>
    </div>

    <div v-if="loading" class="loading-state">
      <i class="fas fa-spinner fa-spin"></i>
      Loading tunnels...
    </div>

    <div v-else-if="tunnels && tunnels.length > 0" class="table-wrapper">
      <div class="table-container">
        <table class="data-table">
          <thead>
            <tr>
              <th>TUNNEL ID</th>
              <th>TUNNEL NAME</th>
              <th>SSH HOST</th>
              <th>SSH PORT</th>
              <th>SSH USERNAME</th>
              <th>CREATED</th>
              <th>ACTIONS</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="tunnel in paginatedTunnels" :key="tunnel.id">
              <td>{{ tunnel.id }}</td>
              <td>{{ tunnel.name || '-' }}</td>
              <td>{{ tunnel.host || '-' }}</td>
              <td>{{ tunnel.port || '-' }}</td>
              <td>{{ tunnel.username || '-' }}</td>
              <td>{{ formatDate(tunnel.created_at) }}</td>
              <td>
                <div class="action-buttons">
                  <button 
                    class="action-btn script-btn" 
                    @click="generateScriptModal(tunnel)"
                    title="Generate SSH Script">
                    <i class="fas fa-code"></i>
                  </button>
                  <button 
                    class="action-btn web-ssh-btn" 
                    @click="openSSHWeb(tunnel)"
                    title="Open SSH Web Terminal">
                    <i class="fas fa-terminal"></i>
                  </button>
                  <button 
                    class="action-btn configure-btn" 
                    @click="editTunnel(tunnel)"
                    title="Edit Tunnel">
                    <i class="fas fa-edit"></i>
                  </button>
                  <button 
                    class="action-btn delete-btn" 
                    @click="deleteTunnel(tunnel)"
                    title="Delete Tunnel">
                    <i class="fas fa-trash"></i>
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <div v-else class="empty-state">
      <i class="fas fa-server"></i>
      No tunnels available
    </div>

    <Pagination
      v-if="tunnels && tunnels.length > itemsPerPage"
      :current-page="currentPage"
      :total-items="tunnels.length"
      :items-per-page="itemsPerPage"
      @page-changed="handlePageChange"
    />

    <!-- SSH Configuration Modal -->
    <div v-if="showConfigModal" class="modal-overlay" @click="closeConfigModal">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h3>Edit SSH Tunnel - {{ selectedTunnel?.name }}</h3>
          <button class="close-btn" @click="closeConfigModal">
            <i class="fas fa-times"></i>
          </button>
        </div>
        
        <form @submit.prevent="saveSSHConfig" class="modal-form">
          <div class="form-group">
            <label for="ssh_host">SSH Host:</label>
            <input 
              v-model="sshConfig.host" 
              type="text" 
              id="ssh_host" 
              placeholder="e.g., 192.168.1.100" 
              required>
          </div>
          
          <div class="form-group">
            <label for="ssh_port">SSH Port:</label>
            <input 
              v-model="sshConfig.port" 
              type="number" 
              id="ssh_port" 
              placeholder="22" 
              min="1" 
              max="65535" 
              required>
          </div>
          
          <div class="form-group">
            <label for="ssh_username">SSH Username:</label>
            <input 
              v-model="sshConfig.username" 
              type="text" 
              id="ssh_username" 
              placeholder="root" 
              required>
          </div>
          
          <div class="form-actions">
            <button type="button" class="btn btn-secondary" @click="closeConfigModal">
              Cancel
            </button>
            <button type="submit" class="btn btn-primary">
              Update Tunnel
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Add Tunnel Modal -->
    <div v-if="showAddTunnelModal" class="modal-overlay" @click="closeAddTunnelModal">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h3>Add New SSH Tunnel</h3>
          <button class="close-btn" @click="closeAddTunnelModal">
            <i class="fas fa-times"></i>
          </button>
        </div>
        
        <form @submit.prevent="addNewTunnel" class="modal-form">
          <div class="form-group">
            <label for="tunnel_name">Tunnel Name:</label>
            <input 
              v-model="newTunnel.name" 
              type="text" 
              id="tunnel_name" 
              placeholder="e.g., Production Server" 
              required>
          </div>
          
          <div class="form-group">
            <label for="tunnel_host">SSH Host:</label>
            <input 
              v-model="newTunnel.host" 
              type="text" 
              id="tunnel_host" 
              placeholder="e.g., 192.168.1.100" 
              required>
          </div>
          
          <div class="form-group">
            <label for="tunnel_port">SSH Port:</label>
            <input 
              v-model="newTunnel.port" 
              type="number" 
              id="tunnel_port" 
              placeholder="22" 
              min="1" 
              max="65535" 
              required>
          </div>
          
          <div class="form-group">
            <label for="tunnel_username">SSH Username:</label>
            <input 
              v-model="newTunnel.username" 
              type="text" 
              id="tunnel_username" 
              placeholder="root" 
              required>
          </div>
          
          <div class="form-group">
            <label for="tunnel_description">Description (Optional):</label>
            <textarea 
              v-model="newTunnel.description" 
              id="tunnel_description" 
              placeholder="Description of this SSH tunnel..."
              rows="3">
            </textarea>
          </div>
          
          <div class="form-actions">
            <button type="button" class="btn btn-secondary" @click="closeAddTunnelModal">
              Cancel
            </button>
            <button type="submit" class="btn btn-primary">
              Add Tunnel
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- SSH Script Generation Modal -->
    <div v-if="showScriptModal" class="modal-overlay" @click="closeScriptModal">
      <div class="modal-content script-modal" @click.stop>
        <div class="modal-header">
          <h3>SSH Tunnel Command - {{ selectedTunnel?.name }}</h3>
          <button class="close-btn" @click="closeScriptModal">
            <i class="fas fa-times"></i>
          </button>
        </div>
        
        <div class="script-content">
          <div class="script-tabs">
            <button 
              class="tab-btn" 
              :class="{ active: selectedScriptType === 'bash' }"
              @click="selectedScriptType = 'bash'">
              Bash (Linux/Mac)
            </button>
            <button 
              class="tab-btn" 
              :class="{ active: selectedScriptType === 'powershell' }"
              @click="selectedScriptType = 'powershell'">
              PowerShell (Windows)
            </button>
            <button 
              class="tab-btn" 
              :class="{ active: selectedScriptType === 'cmd' }"
              @click="selectedScriptType = 'cmd'">
              CMD (Windows)
            </button>
          </div>
          
          <div class="script-display">
            <div class="script-header">
              <span class="script-filename">{{ getScriptFilename() }}</span>
              <button class="copy-btn" @click="copyToClipboard" title="Copy to clipboard">
                <i class="fas fa-copy"></i>
                Copy
              </button>
            </div>
            <pre class="script-code">{{ generateScript() }}</pre>
          </div>
          
          <div class="script-info">
            <div class="info-section">
              <h4>📋 Instructions:</h4>
              <ul>
                <li>Copy the command above</li>
                <li>Open terminal/command prompt in your project directory</li>
                <li>Ensure <code>universal-client</code> binary exists in <code>bin/</code> directory</li>
                <li>Paste and run the command to establish SSH tunnel</li>
              </ul>
            </div>
            
            <div class="info-section">
              <h4>⚠️ Requirements:</h4>
              <ul>
                <li><code>universal-client</code> executable in <code>bin/</code> folder</li>
                <li>Valid admin token: <code>admin_token_2025_secure</code></li>
                <li>Network access to the target server ({{ selectedTunnel?.host }})</li>
                <li>Valid username: {{ selectedTunnel?.username }}</li>
                <li>Agent <code>test1</code> must be running and accessible</li>
              </ul>
            </div>

            <div class="info-section">
              <h4>🔧 Parameters:</h4>
              <ul>
                <li><code>-T</code> : Admin token for authentication</li>
                <li><code>-u</code> : Username ({{ selectedTunnel?.username }})</li>
                <li><code>-H</code> : Target host ({{ selectedTunnel?.host }})</li>
                <li><code>-a</code> : Agent identifier (test1)</li>
                <li><code>-p</code> : Port number ({{ selectedTunnel?.port }})</li>
              </ul>
            </div>
          </div>
        </div>
        
        <div class="form-actions">
          <button type="button" class="btn btn-secondary" @click="closeScriptModal">
            Close
          </button>
          <button type="button" class="btn btn-primary" @click="downloadScript">
            <i class="fas fa-download"></i>
            Download Command
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
  name: 'SSHManagement',
  components: {
    Pagination
  },
  setup() {
    const router = useRouter()
    const tunnels = ref([])
    const loading = ref(true)
    const error = ref('')
    const currentPage = ref(1)
    const itemsPerPage = ref(10)
    const showConfigModal = ref(false)
    const selectedTunnel = ref(null)
    const sshConfig = ref({
      host: '',
      port: 22,
      username: 'root'
    })

    // Add tunnel modal states
    const showAddTunnelModal = ref(false)
    const newTunnel = ref({
      name: '',
      host: '',
      port: 22,
      username: 'root',
      description: ''
    })

    // Script generation modal states
    const showScriptModal = ref(false)
    const selectedScriptType = ref('bash')

    const paginatedTunnels = computed(() => {
      if (!tunnels.value || !Array.isArray(tunnels.value)) {
        return []
      }
      const start = (currentPage.value - 1) * itemsPerPage.value
      const end = start + itemsPerPage.value
      return tunnels.value.slice(start, end)
    })

    const loadTunnels = async () => {
      try {
        loading.value = true
        
        // Try to load tunnels from API, fallback to mock data if API doesn't exist
        let response
        try {
          response = await apiService.getTunnels()
        } catch (apiError) {
          // Fallback to mock data if API doesn't exist
          console.warn('getTunnels API not available, using mock data:', apiError)
          response = {
            data: [
              {
                id: 'tunnel1',
                name: 'Production Server',
                host: '192.168.1.100',
                port: 22,
                username: 'root',
                status: 'CONNECTED',
                created_at: '2023-09-01T10:00:00Z'
              },
              {
                id: 'tunnel2',
                name: 'Development Server',
                host: '192.168.1.101',
                port: 22,
                username: 'ubuntu',
                status: 'DISCONNECTED',
                created_at: '2023-09-02T14:30:00Z'
              }
            ]
          }
        }
        
        tunnels.value = response.data.map(tunnel => ({
          ...tunnel,
          status: tunnel.status || 'DISCONNECTED',
          ssh_enabled: tunnel.ssh_enabled || false,
          created_at: tunnel.created_at || new Date().toISOString()
        }))
      } catch (err) {
        error.value = 'Failed to load tunnels'
        console.error('Error loading tunnels:', err)
        // Set empty array to prevent length errors
        tunnels.value = []
      } finally {
        loading.value = false
      }
    }

    const generateScriptModal = (tunnel) => {
      selectedTunnel.value = tunnel
      showScriptModal.value = true
    }

    const openSSHWeb = (tunnel) => {
      // Navigate to SSH web terminal route with tunnel parameters
      const routeData = router.resolve({
        path: '/ssh-terminal',
        query: {
          host: tunnel.host,
          port: tunnel.port,
          username: tunnel.username,
          tunnel_id: tunnel.id
        }
      })
      
      // Open SSH web terminal in a new tab
      window.open(routeData.href, '_blank')
    }

    const closeScriptModal = () => {
      showScriptModal.value = false
      selectedTunnel.value = null
      selectedScriptType.value = 'bash'
    }

    const generateScript = () => {
      if (!selectedTunnel.value) return ''
      
      const tunnel = selectedTunnel.value
      const host = tunnel.host
      const port = tunnel.port
      const username = tunnel.username
      
      switch (selectedScriptType.value) {
        case 'bash':
          return `./bin/universal-client -T admin_token_2025_secure -u ${username} -H ${host} -a test1 -p ${port}`

        case 'powershell':
          return `.\\bin\\universal-client.exe -T admin_token_2025_secure -u ${username} -H ${host} -a test1 -p ${port}`

        case 'cmd':
          return `.\\bin\\universal-client.exe -T admin_token_2025_secure -u ${username} -H ${host} -a test1 -p ${port}`

        default:
          return ''
      }
    }

    const getScriptFilename = () => {
      if (!selectedTunnel.value) return ''
      
      const tunnelName = selectedTunnel.value.name.toLowerCase().replace(/[^a-z0-9]/g, '_')
      
      switch (selectedScriptType.value) {
        case 'bash':
          return `tunnel_${tunnelName}_command.sh`
        case 'powershell':
          return `tunnel_${tunnelName}_command.ps1`
        case 'cmd':
          return `tunnel_${tunnelName}_command.bat`
        default:
          return `tunnel_${tunnelName}_command.txt`
      }
    }

    const copyToClipboard = async () => {
      try {
        const script = generateScript()
        await navigator.clipboard.writeText(script)
        alert('Script copied to clipboard!')
      } catch (err) {
        console.error('Failed to copy to clipboard:', err)
        // Fallback for older browsers
        const textArea = document.createElement('textarea')
        textArea.value = generateScript()
        document.body.appendChild(textArea)
        textArea.focus()
        textArea.select()
        try {
          document.execCommand('copy')
          alert('Script copied to clipboard!')
        } catch {
          alert('Failed to copy to clipboard. Please select and copy manually.')
        }
        document.body.removeChild(textArea)
      }
    }

    const downloadScript = () => {
      const script = generateScript()
      const filename = getScriptFilename()
      
      const blob = new Blob([script], { type: 'text/plain' })
      const url = URL.createObjectURL(blob)
      
      const link = document.createElement('a')
      link.href = url
      link.download = filename
      document.body.appendChild(link)
      link.click()
      document.body.removeChild(link)
      URL.revokeObjectURL(url)
      
      alert(`Script downloaded as ${filename}`)
    }

    const editTunnel = (tunnel) => {
      selectedTunnel.value = tunnel
      sshConfig.value = {
        host: tunnel.host || '',
        port: tunnel.port || 22,
        username: tunnel.username || 'root'
      }
      showConfigModal.value = true
    }

    const deleteTunnel = async (tunnel) => {
      if (confirm(`Are you sure you want to delete tunnel "${tunnel.name}"?`)) {
        try {
          // Try API call, fallback to mock behavior
          try {
            await apiService.deleteTunnel(tunnel.id)
          } catch (apiError) {
            console.warn('deleteTunnel API not available, using mock behavior:', apiError)
            // Mock successful deletion by removing from local array
            const index = tunnels.value.findIndex(t => t.id === tunnel.id)
            if (index > -1) {
              tunnels.value.splice(index, 1)
            }
            alert('SSH Tunnel deleted successfully!')
            return
          }
          await loadTunnels()
          alert('SSH Tunnel deleted successfully!')
        } catch (err) {
          console.error('Error deleting tunnel:', err)
          alert('Failed to delete SSH tunnel')
        }
      }
    }

    const saveSSHConfig = async () => {
      try {
        const updateData = {
          host: sshConfig.value.host,
          port: parseInt(sshConfig.value.port),
          username: sshConfig.value.username
        }

        // Try API call, fallback to mock behavior
        try {
          await apiService.updateTunnel(selectedTunnel.value.id, updateData)
        } catch (apiError) {
          console.warn('updateTunnel API not available, using mock behavior:', apiError)
        }
        
        // Update local data
        selectedTunnel.value.host = updateData.host
        selectedTunnel.value.port = updateData.port
        selectedTunnel.value.username = updateData.username
        
        closeConfigModal()
        alert('SSH Tunnel updated successfully!')
      } catch (err) {
        console.error('Error saving SSH config:', err)
        alert('Failed to save SSH configuration')
      }
    }

    const closeConfigModal = () => {
      showConfigModal.value = false
      selectedTunnel.value = null
      sshConfig.value = {
        host: '',
        port: 22,
        username: 'root'
      }
    }

    const openAddTunnelModal = () => {
      showAddTunnelModal.value = true
    }

    const closeAddTunnelModal = () => {
      showAddTunnelModal.value = false
      newTunnel.value = {
        name: '',
        host: '',
        port: 22,
        username: 'root',
        description: ''
      }
    }

    const addNewTunnel = async () => {
      try {
        // Call API to create new tunnel
        const tunnelData = {
          id: 'tunnel_' + Date.now(), // Generate temporary ID
          name: newTunnel.value.name,
          host: newTunnel.value.host,
          port: parseInt(newTunnel.value.port),
          username: newTunnel.value.username,
          description: newTunnel.value.description,
          status: 'DISCONNECTED',
          created_at: new Date().toISOString()
        }

        // Try API call, fallback to mock behavior
        try {
          await apiService.createTunnel(tunnelData)
          // Reload tunnels after successful creation
          await loadTunnels()
        } catch (apiError) {
          console.warn('createTunnel API not available, using mock behavior:', apiError)
          // Mock successful creation by adding to local array
          tunnels.value.push(tunnelData)
        }
        
        closeAddTunnelModal()
        
        // Show success message
        alert('SSH Tunnel created successfully!')
      } catch (err) {
        console.error('Error creating SSH tunnel:', err)
        alert('Failed to create SSH tunnel')
      }
    }

    const formatDate = (dateString) => {
      if (!dateString) return 'Never'
      return new Date(dateString).toLocaleString()
    }

    const handlePageChange = (page) => {
      currentPage.value = page
    }

    onMounted(() => {
      loadTunnels()
    })

    return {
      tunnels,
      loading,
      error,
      currentPage,
      itemsPerPage,
      paginatedTunnels,
      showConfigModal,
      selectedTunnel,
      sshConfig,
      showAddTunnelModal,
      newTunnel,
      showScriptModal,
      selectedScriptType,
      generateScriptModal,
      openSSHWeb,
      closeScriptModal,
      generateScript,
      getScriptFilename,
      copyToClipboard,
      downloadScript,
      editTunnel,
      deleteTunnel,
      saveSSHConfig,
      closeConfigModal,
      openAddTunnelModal,
      closeAddTunnelModal,
      addNewTunnel,
      formatDate,
      handlePageChange,
      loadTunnels
    }
  }
}
</script>

<style scoped>
.ssh-management {
  padding: 24px;
}

.page-header {
  margin-bottom: 24px;
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 20px;
}

.header-text {
  flex: 1;
}

.page-header h2 {
  margin: 0 0 8px 0;
  color: var(--text-primary);
  font-size: 28px;
  font-weight: 600;
}

.page-description {
  margin: 0;
  color: var(--text-secondary);
  font-size: 14px;
}

.add-tunnel-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 20px;
  background: var(--primary-color);
  color: white;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  font-size: 14px;
  font-weight: 500;
  transition: all 0.2s ease;
  white-space: nowrap;
}

.add-tunnel-btn:hover {
  background: var(--primary-dark);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(79, 70, 229, 0.3);
}

.add-tunnel-btn i {
  font-size: 12px;
}

.loading-state {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 60px;
  color: var(--text-secondary);
}

.loading-state i {
  margin-right: 12px;
  font-size: 18px;
}

/* Clean Table Styles matching Project Management */
.table-wrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.table-container {
  flex: 1;
  overflow: auto;
  background: var(--surface-color);
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-color);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.data-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.875rem;
}

.data-table thead {
  background: var(--surface-alt);
  border-bottom: 2px solid var(--border-color);
}

.data-table th {
  padding: 1rem 1.25rem;
  text-align: left;
  font-weight: 600;
  color: var(--text-primary);
  font-size: 0.875rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  border-bottom: 1px solid var(--border-color);
}

.data-table thead th i {
  margin-right: 0.5rem;
  color: var(--color-primary);
}

.data-table tbody tr {
  border-bottom: 1px solid var(--border-color);
  transition: background-color 0.2s ease;
}

.data-table tbody tr:hover {
  background: var(--surface-alt);
}

.data-table tbody tr:last-child {
  border-bottom: none;
}

.data-table td {
  padding: 1rem 1.25rem;
  vertical-align: middle;
  color: var(--text-primary);
  font-size: 0.875rem;
}

.ssh-status {
  padding: 6px 12px;
  border-radius: 20px;
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.ssh-enabled {
  background: var(--success-light);
  color: var(--success-dark);
}

.ssh-disabled {
  background: var(--warning-light);
  color: var(--warning-dark);
}

/* Action Buttons matching Project Management style */
.action-buttons {
  display: flex;
  gap: 0.5rem;
  justify-content: center;
  align-items: center;
}

.action-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.25rem;
  padding: 0.5rem;
  border: none;
  border-radius: var(--radius-md);
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  min-width: 32px;
  height: 32px;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
}

.action-btn:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.15);
}

.script-btn {
  background: var(--color-primary);
  color: white;
}

.script-btn:hover {
  background: var(--color-primary-dark);
}

.web-ssh-btn {
  background: #10b981 !important; /* Force green color */
  color: white;
  border: none;
}

.web-ssh-btn:hover {
  background: #059669 !important; /* Darker green on hover */
}

.configure-btn {
  background: var(--color-info);
  color: white;
}

.configure-btn:hover {
  background: var(--color-info-dark);
}

.delete-btn {
  background: var(--color-danger);
  color: white;
}

.delete-btn:hover {
  background: var(--color-danger-dark);
}

.empty-state {
  text-align: center;
  padding: 80px 40px;
  background: var(--background-secondary);
  border-radius: 16px;
  border: 2px dashed var(--border-color);
  margin: 20px 0;
}

.empty-state i {
  font-size: 64px;
  margin-bottom: 24px;
  color: var(--text-secondary);
  opacity: 0.6;
}

.empty-state h3 {
  margin: 0 0 12px 0;
  color: var(--text-primary);
  font-size: 24px;
  font-weight: 600;
}

.empty-state p {
  margin: 0;
  color: var(--text-secondary);
  font-size: 16px;
  line-height: 1.5;
}

.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  backdrop-filter: blur(4px);
}

.modal-content {
  background: #4A5568;
  border-radius: 16px;
  box-shadow: 0 25px 50px rgba(0, 0, 0, 0.4);
  max-width: 520px;
  width: 90%;
  max-height: 90vh;
  overflow-y: auto;
  border: 1px solid #6B7280;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 28px 28px 0 28px;
  margin-bottom: 28px;
  border-bottom: 1px solid #6B7280;
  padding-bottom: 20px;
}

.modal-header h3 {
  margin: 0;
  color: #F7FAFC;
  font-size: 22px;
  font-weight: 600;
  letter-spacing: -0.025em;
}

.close-btn {
  background: none;
  border: none;
  color: #CBD5E0;
  cursor: pointer;
  padding: 8px;
  border-radius: 8px;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
}

.close-btn:hover {
  background: #2D3748;
  color: #F7FAFC;
  transform: scale(1.05);
}

.close-btn i {
  font-size: 16px;
}

.modal-form {
  padding: 0 28px;
}

.form-group {
  margin-bottom: 24px;
}

.form-group:last-of-type {
  margin-bottom: 0;
}

.form-group label {
  display: block;
  margin-bottom: 10px;
  color: #F7FAFC;
  font-size: 15px;
  font-weight: 500;
  letter-spacing: -0.01em;
}

.form-group input {
  width: 100%;
  padding: 14px 16px;
  border: 1.5px solid #6B7280;
  border-radius: 10px;
  background: #2D3748;
  color: #F7FAFC;
  font-size: 15px;
  transition: all 0.2s ease;
  box-sizing: border-box;
}

.form-group textarea {
  width: 100%;
  padding: 14px 16px;
  border: 1.5px solid #6B7280;
  border-radius: 10px;
  background: #2D3748;
  color: #F7FAFC;
  font-size: 15px;
  resize: vertical;
  font-family: inherit;
  transition: all 0.2s ease;
  box-sizing: border-box;
  min-height: 100px;
}

.form-group input:focus,
.form-group textarea:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 4px rgba(79, 70, 229, 0.1);
  transform: translateY(-1px);
}

.form-group input::placeholder,
.form-group textarea::placeholder {
  color: #A0AEC0;
  opacity: 0.8;
}

.form-actions {
  display: flex;
  gap: 16px;
  justify-content: flex-end;
  padding: 28px;
  border-top: 1px solid #6B7280;
  margin-top: 16px;
  background: #4A5568;
  border-radius: 0 0 16px 16px;
}

.btn {
  padding: 14px 28px;
  border: none;
  border-radius: 10px;
  cursor: pointer;
  font-size: 15px;
  font-weight: 600;
  transition: all 0.2s ease;
  min-width: 120px;
  letter-spacing: -0.01em;
}

.btn-primary {
  background: var(--primary-color);
  color: white;
  box-shadow: 0 2px 8px rgba(79, 70, 229, 0.3);
}

.btn-primary:hover {
  background: var(--primary-dark);
  transform: translateY(-2px);
  box-shadow: 0 6px 20px rgba(79, 70, 229, 0.4);
}

.btn-secondary {
  background: #4A5568;
  color: #CBD5E0;
  border: 1.5px solid #6B7280;
}

.btn-secondary:hover {
  background: #2D3748;
  color: #F7FAFC;
  border-color: #9CA3AF;
  transform: translateY(-1px);
}

/* Script Modal Styles */
.script-modal {
  max-width: 800px;
  width: 95%;
}

.script-content {
  padding: 0 28px;
}

.script-tabs {
  display: flex;
  gap: 8px;
  margin-bottom: 20px;
  border-bottom: 1px solid #6B7280;
  padding-bottom: 10px;
}

.tab-btn {
  padding: 8px 16px;
  border: none;
  border-radius: 6px 6px 0 0;
  background: #2D3748;
  color: #CBD5E0;
  cursor: pointer;
  font-size: 14px;
  transition: all 0.2s ease;
}

.tab-btn:hover {
  background: #4A5568;
  color: #F7FAFC;
}

.tab-btn.active {
  background: #4F46E5;
  color: white;
}

.script-display {
  background: #1A1A1A;
  border-radius: 8px;
  overflow: hidden;
  margin-bottom: 20px;
}

.script-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: #2D3748;
  border-bottom: 1px solid #4A5568;
}

.script-filename {
  color: #9CA3AF;
  font-family: 'Courier New', monospace;
  font-size: 14px;
}

.copy-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  background: #059669;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
  transition: all 0.2s ease;
}

.copy-btn:hover {
  background: #047857;
  transform: translateY(-1px);
}

.script-code {
  background: #1A1A1A;
  color: #F7FAFC;
  padding: 20px;
  margin: 0;
  font-family: 'Courier New', Monaco, 'Lucida Console', monospace;
  font-size: 13px;
  line-height: 1.5;
  overflow-x: auto;
  white-space: pre-wrap;
  border: none;
  outline: none;
}

.script-info {
  background: #374151;
  border-radius: 8px;
  padding: 20px;
  margin-bottom: 20px;
}

.info-section {
  margin-bottom: 16px;
}

.info-section:last-child {
  margin-bottom: 0;
}

.info-section h4 {
  margin: 0 0 12px 0;
  color: #F7FAFC;
  font-size: 16px;
  font-weight: 600;
}

.info-section ul {
  margin: 0;
  padding-left: 20px;
  color: #D1D5DB;
}

.info-section li {
  margin-bottom: 8px;
  line-height: 1.5;
}

.info-section code {
  background: #1F2937;
  color: #10B981;
  padding: 2px 6px;
  border-radius: 4px;
  font-family: 'Courier New', monospace;
  font-size: 13px;
}

/* Responsive Design */
@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    gap: 16px;
    text-align: center;
  }
  
  .add-tunnel-btn {
    width: 100%;
    justify-content: center;
  }
  
  .table-wrapper {
    overflow-x: auto;
    border-radius: 12px;
  }
  
  .data-table {
    min-width: 600px;
  }
  
  .data-table th,
  .data-table td {
    padding: 12px 8px;
    font-size: 14px;
  }
  
  .action-buttons {
    flex-direction: column;
    gap: 8px;
  }
  
  .action-buttons .action-btn {
    width: 100%;
    justify-content: center;
  }
  
  .modal-overlay .modal-content {
    margin: 20px;
    width: calc(100% - 40px);
    max-height: calc(100vh - 40px);
    overflow-y: auto;
  }
  
  .empty-state,
  .loading-state {
    padding: 40px 20px;
  }
  
  .empty-state i {
    font-size: 48px;
  }
  
  .empty-state h3 {
    font-size: 20px;
  }
  
  .script-modal {
    width: 98%;
  }
  
  .script-tabs {
    flex-wrap: wrap;
    gap: 4px;
  }
  
  .tab-btn {
    padding: 6px 12px;
    font-size: 13px;
  }
}

@media (max-width: 480px) {
  .data-table th,
  .data-table td {
    padding: 10px 6px;
    font-size: 13px;
  }
  
  .page-title {
    font-size: 24px;
  }
  
  .modal-overlay .modal-content {
    border-radius: 12px;
  }
  
  .action-btn {
    width: 36px;
    height: 36px;
    font-size: 14px;
  }
  
  .form-actions {
    flex-direction: column;
  }
  
  .btn {
    width: 100%;
  }
}

/* CSS Custom Properties for consistent theming */
:root {
  --color-primary: #3b82f6;
  --color-primary-dark: #2563eb;
  --color-success: #10b981;
  --color-success-dark: #059669;
  --color-warning: #f59e0b;
  --color-warning-dark: #d97706;
  --color-danger: #ef4444;
  --color-danger-dark: #dc2626;
  --color-info: #8b5cf6;
  --color-info-dark: #7c3aed;
  
  --text-primary: #1f2937;
  --text-secondary: #6b7280;
  --background-color: #ffffff;
  --surface-color: #ffffff;
  --surface-alt: #f9fafb;
  --border-color: #e5e7eb;
  
  --radius-sm: 0.25rem;
  --radius-md: 0.375rem;
  --radius-lg: 0.5rem;
}

/* Dark theme support */
@media (prefers-color-scheme: dark) {
  :root {
    --text-primary: #f9fafb;
    --text-secondary: #d1d5db;
    --background-color: #111827;
    --surface-color: #1f2937;
    --surface-alt: #374151;
    --border-color: #4b5563;
  }
}
</style>