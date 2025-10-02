<template>
  <div class="ssh-management">
    <div class="page-header">
      <div class="header-content">
        <div class="header-text">
          <h2>SSH Management</h2>
          <p class="page-description">
            Manage SSH connections for virtual machines. Configure VM SSH access to remote servers.
          </p>
        </div>
        <button class="add-vm-ssh-btn" @click="openAddTunnelModal">
          <i class="fas fa-plus"></i>
          Add VM SSH
        </button>
      </div>
    </div>

    <div v-if="loading" class="loading-state">
      <i class="fas fa-spinner fa-spin"></i>
      Loading VM SSH connections...
    </div>

    <div v-else-if="tunnels && tunnels.length > 0" class="table-wrapper">
      <div class="table-container">
        <table class="data-table">
          <thead>
            <tr>
              <th>VM ID</th>
              <th>VM NAME</th>
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
                    <span class="btn-caption">Script</span>
                  </button>
                  <button 
                    class="action-btn web-ssh-btn" 
                    @click="openSSHWeb(tunnel)"
                    title="Open SSH Web Terminal">
                    <i class="fas fa-desktop"></i>
                    <span class="btn-caption">Web SSH</span>
                  </button>
                  <button 
                    class="action-btn configure-btn" 
                    @click="editTunnel(tunnel)"
                    title="Edit Tunnel">
                    <i class="fas fa-edit"></i>
                    <span class="btn-caption">Edit</span>
                  </button>
                  <button 
                    class="action-btn delete-btn" 
                    @click="deleteTunnel(tunnel)"
                    title="Delete Tunnel">
                    <i class="fas fa-trash"></i>
                    <span class="btn-caption">Delete</span>
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
      No VM SSH connections available
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
          <h3>Edit VM SSH - {{ selectedTunnel?.name }}</h3>
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
          
          <div class="form-group">
            <label for="ssh_password">SSH Password:</label>
            <input 
              v-model="sshConfig.password" 
              type="password" 
              id="ssh_password" 
              placeholder="Enter SSH password" 
              required>
          </div>
          
          <div class="form-actions">
            <button type="button" class="btn btn-secondary" @click="closeConfigModal">
              Cancel
            </button>
            <button type="submit" class="btn btn-primary">
              Update VM SSH
            </button>
          </div>
        </form>
      </div>
    </div>

    <!-- Add VM SSH Modal -->
    <div v-if="showAddTunnelModal" class="modal-overlay" @click="closeAddTunnelModal">
      <div class="modal-content" @click.stop>
        <div class="modal-header">
          <h3>Add New VM SSH</h3>
          <button class="close-btn" @click="closeAddTunnelModal">
            <i class="fas fa-times"></i>
          </button>
        </div>
        
        <form @submit.prevent="addNewTunnel" class="modal-form">
          <div class="form-group">
            <label for="tunnel_name">VM Name:</label>
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
            <label for="tunnel_password">SSH Password:</label>
            <input 
              v-model="newTunnel.password" 
              type="password" 
              id="tunnel_password" 
              placeholder="Enter SSH password" 
              required>
          </div>
          
          <div class="form-group">
            <label for="tunnel_description">Description (Optional):</label>
            <textarea 
              v-model="newTunnel.description" 
              id="tunnel_description" 
              placeholder="Description of this VM SSH connection..."
              rows="3">
            </textarea>
          </div>
          
          <div class="form-actions">
            <button type="button" class="btn btn-secondary" @click="closeAddTunnelModal">
              Cancel
            </button>
            <button type="submit" class="btn btn-primary">
              Add VM SSH
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
  props: {
    agentId: {
      type: String,
      default: null
    }
  },
  components: {
    Pagination
  },
  setup(props) {
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
      username: 'root',
      password: ''
    })

    // Add tunnel modal states
    const showAddTunnelModal = ref(false)
    const newTunnel = ref({
      name: '',
      host: '',
      port: 22,
      username: 'root',
      password: '',
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
        
        // Load tunnels from database API
        const response = await apiService.getTunnels()
        
        console.log('API Response:', response)
        console.log('Response data:', response.data)
        
        // Handle different response formats
        let tunnelsData = []
        if (response.data && Array.isArray(response.data.data)) {
          // Backend sends { data: [...] }
          tunnelsData = response.data.data
        } else if (response.data && Array.isArray(response.data)) {
          // Backend sends [...] directly
          tunnelsData = response.data
        } else {
          console.error('Unexpected response format:', response.data)
          tunnelsData = []
        }
        
        tunnels.value = tunnelsData.map(tunnel => ({
          ...tunnel,
          status: tunnel.status || 'DISCONNECTED',
          ssh_enabled: tunnel.ssh_enabled !== false, // default to true
          created_at: tunnel.created_at || new Date().toISOString()
        }))
      } catch (err) {
        error.value = 'Failed to load tunnels from database'
        console.error('Error loading tunnels:', err)
        console.error('Error details:', err.response?.data)
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
      // Create secure connection data for auto-login
      const connectionData = {
        host: tunnel.host,
        port: tunnel.port,
        username: tunnel.username,
        password: tunnel.password || '',
        tunnel_id: tunnel.id
      };
      
      // Store connection data in session storage for security
      sessionStorage.setItem('ssh_connection_data', JSON.stringify(connectionData));
      
      // Open SSH Web Terminal using router path
      const routeData = router.resolve({
        path: '/ssh-terminal',
        query: {} // No sensitive data in URL
      });
      
      // Open in new tab
      window.open(routeData.href, '_blank');
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
        username: tunnel.username || 'root',
        password: tunnel.password || ''
      }
      showConfigModal.value = true
    }

    const deleteTunnel = async (tunnel) => {
      if (confirm(`Are you sure you want to delete tunnel "${tunnel.name}"?`)) {
        try {
          await apiService.deleteTunnel(tunnel.id)
          await loadTunnels()
          alert('VM SSH connection deleted successfully!')
        } catch (err) {
          console.error('Error deleting tunnel:', err)
          alert('Failed to delete VM SSH connection')
        }
      }
    }

    const saveSSHConfig = async () => {
      try {
        const updateData = {
          host: sshConfig.value.host,
          port: parseInt(sshConfig.value.port),
          username: sshConfig.value.username,
          password: sshConfig.value.password
        }

        await apiService.updateTunnel(selectedTunnel.value.id, updateData)
        await loadTunnels() // Refresh data from database
        
        closeConfigModal()
        alert('VM SSH connection updated successfully!')
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
        username: 'root',
        password: ''
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
        password: '',
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
          password: newTunnel.value.password,
          description: newTunnel.value.description,
          status: 'DISCONNECTED'
        }

        await apiService.createTunnel(tunnelData)
        await loadTunnels() // Reload tunnels from database
        
        closeAddTunnelModal()
        alert('VM SSH connection created successfully!')
      } catch (err) {
        console.error('Error creating SSH tunnel:', err)
        alert('Failed to create VM SSH connection')
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
      
      // If agentId is provided, automatically open SSH Web Terminal
      if (props.agentId) {
        // Navigate to SSH Web Terminal with agentId
        router.push({ name: 'SSHWebTerminal', query: { agentId: props.agentId } })
      }
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

.add-vm-ssh-btn {
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

.add-vm-ssh-btn:hover {
  background: var(--primary-dark);
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(79, 70, 229, 0.3);
}

.add-vm-ssh-btn i {
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
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 8px 6px;
  min-width: 60px;
}

.script-btn:hover {
  background: var(--color-primary-dark);
}

.script-btn .btn-caption {
  font-size: 10px;
  font-weight: 500;
  line-height: 1;
}

.web-ssh-btn {
  background: #10b981 !important; /* Force green color */
  color: white;
  border: none;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 8px 6px;
  min-width: 60px;
}

.web-ssh-btn:hover {
  background: #059669 !important; /* Darker green on hover */
}

.web-ssh-btn .btn-caption {
  font-size: 10px;
  font-weight: 500;
  line-height: 1;
}

.configure-btn {
  background: var(--color-info);
  color: white;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 8px 6px;
  min-width: 60px;
}

.configure-btn:hover {
  background: var(--color-info-dark);
}

.configure-btn .btn-caption {
  font-size: 10px;
  font-weight: 500;
  line-height: 1;
}

.delete-btn {
  background: var(--color-danger);
  color: white;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 8px 6px;
  min-width: 60px;
}

.delete-btn:hover {
  background: var(--color-danger-dark);
}

.delete-btn .btn-caption {
  font-size: 10px;
  font-weight: 500;
  line-height: 1;
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
  
  .add-vm-ssh-btn {
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