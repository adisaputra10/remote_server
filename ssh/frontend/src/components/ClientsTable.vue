<template>
  <div class="clients-table-container">
        <div class="table-header">
      <h2 class="table-title">History Client</h2>
      <div class="table-actions">
        <button class="btn btn-primary" @click="refreshData">
          <i class="fas fa-sync-alt"></i>
          Refresh
        </button>
      </div>
    </div>
    
    <div v-if="loading" class="loading-state">
      <i class="fas fa-spinner fa-spin"></i>
      Loading clients...
    </div>
    
    <div v-else-if="error" class="error-state">
      <i class="fas fa-exclamation-triangle"></i>
      {{ error }}
    </div>
    
    <div v-else-if="allClients.length === 0" class="empty-state">
      <i class="fas fa-users"></i>
      No clients connected
    </div>
    
    <div v-else class="table-wrapper">
      <div class="table-container">
        <table class="table">
          <thead>
            <tr>
              <th>CLIENT ID</th>
              <th>NAME</th>
              <th>STATUS</th>
              <th>CONNECTED AT</th>
              <th>AGENT ID</th>
              <th>ACTIONS</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="client in paginatedClients" :key="client.id">
              <td>{{ client.id }}</td>
              <td>{{ client.name }}</td>
              <td>
                <span :class="['badge', 'badge-' + client.status]">
                  {{ getStatusText(client.status) }}
                </span>
              </td>
              <td>{{ client.connectedAt }}</td>
              <td>{{ client.agentId || '-' }}</td>
              <td>
                <div class="action-buttons">
                  <button 
                    class="action-btn delete-btn with-text" 
                    @click="showDelete(client.id)"
                    title="Delete Client">
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
        :total-items="allClients.length"
        :items-per-page="itemsPerPage"
        @page-changed="handlePageChange"
      />
    </div>

    <!-- Delete Client Confirmation Modal -->
    <div v-if="showDeleteModal" class="modal-overlay" @click="closeDeleteModal">
      <div class="modal delete-modal" @click.stop>
        <div class="modal-header">
          <h3>
            <i class="fas fa-exclamation-triangle text-warning"></i>
            Confirm Delete
          </h3>
          <button class="btn-close" @click="closeDeleteModal">
            <i class="fas fa-times"></i>
          </button>
        </div>
        
        <div class="modal-body">
          <p class="delete-warning">
            Are you sure you want to delete client <strong>{{ clientToDelete }}</strong>?
          </p>
          <p class="delete-note">
            This action cannot be undone. The client will be permanently removed from the system.
          </p>
        </div>
        
        <div class="modal-footer">
          <button type="button" class="btn btn-secondary" @click="closeDeleteModal">
            Cancel
          </button>
          <button type="button" class="btn btn-danger" @click="deleteClient">
            <i class="fas fa-trash"></i>
            Delete Client
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
  name: 'ClientsTable',
  components: {
    Pagination
  },
  setup() {
    const allClients = ref([])
    const loading = ref(false)
    const error = ref(null)
    const currentPage = ref(1)
    const itemsPerPage = ref(20)

    // Delete Modal Data  
    const showDeleteModal = ref(false)
    const clientToDelete = ref('')

    // Server Settings Data
    const serverSettings = ref({
      serverIP: '168.231.119.242',
      serverPort: '8080'
    })

    const paginatedClients = computed(() => {
      const start = (currentPage.value - 1) * itemsPerPage.value
      const end = start + itemsPerPage.value
      return allClients.value.slice(start, end)
    })

    const fetchClients = async () => {
      try {
        loading.value = true
        error.value = null
        
        console.log('=== CLIENTS TABLE - FETCHING FROM API ===')
        console.log('API URL:', import.meta.env.VITE_API_BASE_URL + '/api/clients')
        
        const response = await apiService.getClients()
        console.log('=== RAW CLIENTS API RESPONSE ===')
        console.log('Response data:', response.data)
        console.log('Number of clients:', response.data ? response.data.length : 0)
        
        // Transform data to match the expected format
        allClients.value = (response.data || []).map((client, index) => {
          console.log(`=== CLIENT ${index + 1} RAW DATA ===`, client)
          
          let statusClass = 'warning'
          const rawStatus = client.status || client.active || 'unknown'
          console.log(`Client ${client.id || client.client_id}: Raw status = ${rawStatus}`)
          
          if (client.status === 'connected' || client.active) {
            statusClass = 'success'
          } else if (client.status === 'disconnected') {
            statusClass = 'danger'
          }
          
          const transformedClient = {
            id: client.id || client.client_id || 'Unknown',
            name: client.name || client.description || client.tunnel_type || 'MySQL Database Tunnel',
            status: statusClass,
            connectedAt: client.connected_at || client.created_at || client.start_time || 'Unknown',
            lastPing: client.last_ping || client.last_seen || client.updated_at || 'Unknown',
            agentId: client.agent_id || '-'
          }
          
          console.log(`=== TRANSFORMED CLIENT ${index + 1} ===`, transformedClient)
          console.log(`Final status for ${transformedClient.id}: ${statusClass} (${rawStatus})`)
          
          return transformedClient
        })
        
        console.log('=== FINAL CLIENTS DATA ===')
        console.log('Total clients:', allClients.value.length)
        console.log('All clients status:', allClients.value.map(c => `${c.id}: ${c.status}`))
        
      } catch (err) {
        console.error('=== CLIENTS API ERROR ===')
        console.error('Error fetching clients:', err)
        error.value = 'Failed to load clients data from relay server'
        allClients.value = []
      } finally {
        loading.value = false
      }
    }

    const refreshData = () => {
      console.log('Refreshing clients data...')
      fetchClients()
    }

    const loadServerSettings = async () => {
      try {
        console.log('=== LOADING SERVER SETTINGS ===')
        const response = await apiService.getSettings()
        console.log('Settings API response:', response.data)
        
        if (response.data && response.data.length > 0) {
          const settings = response.data[0]
          serverSettings.value = {
            serverIP: settings.server_ip || '168.231.119.242',
            serverPort: settings.server_port || '8080'
          }
          console.log('Server settings loaded:', serverSettings.value)
        }
      } catch (error) {
        console.error('Error loading server settings:', error)
        // Keep default values if API fails
      }
    }

    const viewDetails = (clientId) => {
      console.log('Viewing details for client:', clientId)
      alert(`Client Details: ${clientId}`)
    }

    const disconnectClient = (clientId) => {
      if (confirm(`Are you sure you want to disconnect client ${clientId}?`)) {
        console.log('Disconnecting client:', clientId)
        // Implement disconnect functionality
        alert('Disconnect functionality: This would send a disconnect request to the API')
      }
    }

    const handlePageChange = (page) => {
      currentPage.value = page
    }

    // Delete Functions
    const showDelete = (clientId) => {
      console.log(`Showing delete confirmation for client: ${clientId}`)
      clientToDelete.value = clientId
      showDeleteModal.value = true
    }

    const closeDeleteModal = () => {
      showDeleteModal.value = false
      clientToDelete.value = ''
    }

    const deleteClient = async () => {
      if (!clientToDelete.value) return
      
      try {
        console.log(`Deleting client: ${clientToDelete.value}`)
        await apiService.deleteClient(clientToDelete.value)
        
        // Refresh the table after deletion
        await fetchClients()
        closeDeleteModal()
        
        console.log(`Client ${clientToDelete.value} deleted successfully`)
        alert(`Client ${clientToDelete.value} deleted successfully`)
      } catch (error) {
        console.error('Error deleting client:', error)
        alert('Failed to delete client: ' + (error.response?.data?.error || error.message))
      }
    }

    const getStatusText = (status) => {
      switch (status) {
        case 'success':
          return 'connected'
        case 'danger':
          return 'disconnected'
        case 'warning':
          return 'warning'
        default:
          return status
      }
    }

    onMounted(() => {
      fetchClients()
      loadServerSettings()
      
      // Auto-refresh every 10 seconds (same as Connection Logs for consistency)
      setInterval(fetchClients, 10000)
    })

    return {
      allClients,
      paginatedClients,
      loading,
      error,
      currentPage,
      itemsPerPage,
      showDeleteModal,
      clientToDelete,
      serverSettings,
      refreshData,
      viewDetails,
      disconnectClient,
      handlePageChange,
      getStatusText,
      showDelete,
      closeDeleteModal,
      deleteClient
    }
  }
}
</script>

<style scoped>
.clients-table-container {
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

.table-title {
  font-size: 1.25rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.header-actions {
  display: flex;
  gap: 0.75rem;
  align-items: center;
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

.action-btn.danger:hover {
  background: var(--danger-color);
  border-color: var(--danger-color);
}

/* Add Client Modal Styles */
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

.modal {
  background: var(--surface-color);
  border-radius: var(--radius-lg);
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.15);
  width: 90%;
  max-width: 500px;
  max-height: 90vh;
  overflow: hidden;
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
  background: var(--surface-alt);
  color: var(--text-primary);
}

.modal-body {
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
  font-size: 0.875rem;
}

.form-input {
  width: 100%;
  padding: 0.75rem;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  background: var(--background-color);
  color: var(--text-primary);
  font-size: 0.875rem;
  transition: all 0.2s ease;
  box-sizing: border-box;
}

.form-input:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.form-input::placeholder {
  color: var(--text-secondary);
}

.token-input-group {
  display: flex;
  gap: 0.5rem;
  align-items: stretch;
}

.token-input-group .form-input {
  flex: 1;
}

.token-input-group .btn {
  padding: 0.75rem 1rem;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  background: var(--surface-alt);
  color: var(--text-secondary);
  font-size: 0.75rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  gap: 0.375rem;
  white-space: nowrap;
}

.token-input-group .btn:hover {
  background: var(--primary-color);
  color: white;
  border-color: var(--primary-color);
}

.token-input-group .btn-show {
  min-width: 70px;
}

.token-input-group .btn-generate {
  min-width: 90px;
}

.form-help {
  display: block;
  margin-top: 0.375rem;
  font-size: 0.75rem;
  color: var(--text-secondary);
  line-height: 1.4;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  padding-top: 1rem;
  border-top: 1px solid var(--border-color);
  margin-top: 1rem;
}

.modal-footer .btn {
  padding: 0.75rem 1.5rem;
  border: none;
  border-radius: var(--radius-md);
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.modal-footer .btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.modal-footer .btn-secondary {
  background: var(--surface-alt);
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
}

.modal-footer .btn-secondary:hover:not(:disabled) {
  background: var(--border-color);
  color: var(--text-primary);
}

.modal-footer .btn-success {
  background: var(--color-success);
  color: white;
}

.modal-footer .btn-success:hover:not(:disabled) {
  background: var(--color-success-dark);
  transform: translateY(-1px);
}

.modal-footer .btn-danger {
  background: var(--color-danger);
  color: white;
}

.modal-footer .btn-danger:hover:not(:disabled) {
  background: var(--color-danger-dark);
  transform: translateY(-1px);
}

/* Setup Modal Styles */
.setup-modal {
  max-width: 800px;
  max-height: 90vh;
  overflow-y: auto;
}

.setup-tabs {
  display: flex;
  border-bottom: 1px solid var(--border-color);
  margin-bottom: 1.5rem;
}

.tab-btn {
  flex: 1;
  padding: 0.75rem 1rem;
  border: none;
  background: transparent;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  font-size: 0.875rem;
  font-weight: 500;
}

.tab-btn:hover {
  background: var(--surface-alt);
  color: var(--text-primary);
}

.tab-btn.active {
  color: var(--color-primary);
  border-bottom: 2px solid var(--color-primary);
  background: var(--surface-alt);
}

.setup-section h4 {
  color: var(--text-primary);
  margin-bottom: 1.5rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 1.1rem;
}

.step {
  margin-bottom: 2rem;
}

.step h5 {
  color: var(--text-primary);
  margin-bottom: 0.75rem;
  font-size: 1rem;
}

.code-block {
  position: relative;
  background: var(--surface-alt);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  padding: 1rem;
  margin-bottom: 1rem;
}

.code-block pre {
  margin: 0;
  white-space: pre-wrap;
  word-wrap: break-word;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.85rem;
  line-height: 1.4;
  color: var(--text-primary);
}

.copy-btn {
  position: absolute;
  top: 0.5rem;
  right: 0.5rem;
  background: var(--color-primary);
  color: white;
  border: none;
  border-radius: var(--radius-sm);
  padding: 0.5rem;
  cursor: pointer;
  transition: all 0.2s ease;
  font-size: 0.75rem;
}

.copy-btn:hover {
  background: var(--color-primary-dark);
  transform: scale(1.05);
}

/* Action buttons with text */
.action-btn.with-text {
  width: auto;
  padding: 0.5rem 0.75rem;
  gap: 0.375rem;
  font-size: 0.75rem;
  font-weight: 500;
  min-width: 70px;
}

.action-btn.with-text i {
  font-size: 0.8rem;
}

/* Delete modal styles */
.delete-modal {
  max-width: 450px;
}

.delete-warning {
  margin-bottom: 1rem;
  color: var(--text-primary);
  font-size: 1rem;
}

.delete-note {
  margin-bottom: 0;
  color: var(--text-secondary);
  font-size: 0.875rem;
}

.text-warning {
  color: var(--color-warning);
}
</style>
