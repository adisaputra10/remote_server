<template>
  <div class="ssh-logs-table-container">
    <div class="table-header">
      <h2 class="table-title">SSH Command Logs</h2>
      <div class="header-actions">
        <button class="btn btn-primary" @click="refreshData">
          <i class="fas fa-sync-alt" :class="{ 'fa-spin': loading }"></i>
          Refresh
        </button>
        <div class="search-container">
          <input 
            type="text" 
            v-model="searchQuery" 
            placeholder="Search commands..." 
            class="search-input"
          >
          <i class="fas fa-search search-icon"></i>
        </div>
      </div>
    </div>

    <!-- Stats Cards -->
    <div class="stats-container">
      <div class="stat-card">
        <div class="stat-value">{{ filteredLogs.length }}</div>
        <div class="stat-label">Total Commands</div>
      </div>
      <div class="stat-card">
        <div class="stat-value">{{ uniqueUsers.length }}</div>
        <div class="stat-label">Unique Users</div>
      </div>
      <div class="stat-card">
        <div class="stat-value">{{ uniqueHosts.length }}</div>
        <div class="stat-label">Unique Hosts</div>
      </div>
      <div class="stat-card">
        <div class="stat-value">{{ todayCommands }}</div>
        <div class="stat-label">Today's Commands</div>
      </div>
    </div>
    
    <div v-if="loading" class="loading-state">
      <i class="fas fa-spinner fa-spin"></i>
      Loading SSH logs...
    </div>
    
    <div v-else-if="error" class="error-state">
      <i class="fas fa-exclamation-triangle"></i>
      {{ error }}
    </div>
    
    <div v-else-if="allSSHLogs.length === 0" class="empty-state">
      <i class="fas fa-terminal"></i>
      No SSH command logs available
      <br>
      <small>API: {{ apiBaseUrl }}/api/ssh-logs</small>
    </div>
    
    <div v-else class="table-wrapper">
      <div class="table-container">
        <table class="table">
                    <thead>
            <tr>
              <th @click="sortBy('timestamp')" class="sortable">
                TIMESTAMP
                <i :class="getSortIcon('timestamp')"></i>
              </th>
              <th @click="sortBy('sessionId')" class="sortable">
                SESSION ID
                <i :class="getSortIcon('sessionId')"></i>
              </th>
              <th @click="sortBy('ssh_user')" class="sortable">
                USER
                <i :class="getSortIcon('ssh_user')"></i>
              </th>
              <th @click="sortBy('ssh_host')" class="sortable">
                HOST
                <i :class="getSortIcon('ssh_host')"></i>
              </th>
              <th class="command-header">COMMAND</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="log in paginatedSSHLogs" :key="log.session_id + log.timestamp">
              <td class="timestamp-cell">
                <div class="timestamp">{{ formatTimestamp(log.timestamp) }}</div>
              </td>
              <td class="session-cell">
                <div class="session-id">{{ log.sessionId || log.session_id || '-' }}</div>
              </td>
              <td class="user-cell">
                <div class="user-info">
                  <i class="fas fa-user"></i>
                  {{ log.ssh_user || '-' }}
                </div>
              </td>
              <td class="host-cell">
                <div class="host-info">
                  <i class="fas fa-server"></i>
                  {{ log.ssh_host || '-' }}
                </div>
              </td>
              <td class="command-cell-wide">
                <div class="command-text-wide" :title="log.command">
                  {{ log.command || '-' }}
                </div>
                <div class="command-direction" :class="log.direction">
                  {{ log.direction || 'input' }}
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      
      <Pagination
        :current-page="currentPage"
        :total-items="filteredLogs.length"
        :items-per-page="itemsPerPage"
        @page-changed="handlePageChange"
      />
    </div>
  </div>
</template>

<script>
import { ref, onMounted, computed, watch } from 'vue'
import { apiService } from '../config/api.js'
import Pagination from './Pagination.vue'

export default {
  name: 'SSHLogsTable',
  components: {
    Pagination
  },
  setup() {
    const allSSHLogs = ref([])
    const loading = ref(false)
    const error = ref(null)
    const currentPage = ref(1)
    const itemsPerPage = ref(20)
    const searchQuery = ref('')
    const sortField = ref('timestamp')
    const sortDirection = ref('desc')

    // Computed properties for filtering and stats
    const filteredLogs = computed(() => {
      let filtered = allSSHLogs.value

      // Apply search filter
      if (searchQuery.value) {
        const query = searchQuery.value.toLowerCase()
        filtered = filtered.filter(log => 
          (log.command && log.command.toLowerCase().includes(query)) ||
          (log.ssh_user && log.ssh_user.toLowerCase().includes(query)) ||
          (log.ssh_host && log.ssh_host.toLowerCase().includes(query)) ||
          (log.sessionId && log.sessionId.toLowerCase().includes(query)) ||
          (log.session_id && log.session_id.toLowerCase().includes(query))
        )
      }

      // Apply sorting
      filtered.sort((a, b) => {
        let aVal = a[sortField.value]
        let bVal = b[sortField.value]
        
        if (sortField.value === 'timestamp') {
          aVal = new Date(aVal)
          bVal = new Date(bVal)
        }
        
        const comparison = aVal < bVal ? -1 : aVal > bVal ? 1 : 0
        return sortDirection.value === 'asc' ? comparison : -comparison
      })

      return filtered
    })

    const uniqueUsers = computed(() => {
      return [...new Set(filteredLogs.value.map(log => log.ssh_user).filter(Boolean))]
    })

    const uniqueHosts = computed(() => {
      return [...new Set(filteredLogs.value.map(log => log.ssh_host).filter(Boolean))]
    })

    const todayCommands = computed(() => {
      const today = new Date().toDateString()
      return filteredLogs.value.filter(log => 
        new Date(log.timestamp).toDateString() === today
      ).length
    })

    const paginatedSSHLogs = computed(() => {
      const start = (currentPage.value - 1) * itemsPerPage.value
      const end = start + itemsPerPage.value
      return filteredLogs.value.slice(start, end)
    })

    const apiBaseUrl = computed(() => {
      return import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080'
    })

    const fetchSSHLogs = async () => {
      try {
        loading.value = true
        error.value = null
        
        console.log('=== SSH LOGS - FETCHING FROM API ===')
        console.log('API URL:', import.meta.env.VITE_API_BASE_URL + '/api/ssh-logs')
        
        // Clear existing data first
        allSSHLogs.value = []
        
        const response = await apiService.getSSHLogs()
        console.log('API Response received, processing data...')
        
        // Validate API response
        if (!response.data || !Array.isArray(response.data)) {
          console.error('❌ Invalid API response format')
          allSSHLogs.value = []
          error.value = 'Invalid API response format'
          return
        }
        
        if (response.data.length === 0) {
          console.log('⚠️ No SSH logs available')
          allSSHLogs.value = []
          return
        }
        
        console.log('✅ API returned valid data, processing records...')
        
        // PROCESS EACH RECORD FROM API - NO DUMMY DATA
        allSSHLogs.value = response.data.map((log, index) => {
          console.log(`=== PROCESSING API RECORD ${index + 1}/${response.data.length} ===`)
          console.log('Raw log record:', log)
          console.log('Record type:', typeof log)
          console.log('Available fields:', Object.keys(log))
          
          // Parse JSON if needed
          let parsedData = log
          if (typeof log === 'string') {
            try {
              parsedData = JSON.parse(log)
              console.log('✅ Parsed JSON data:', parsedData)
            } catch (parseError) {
              console.log('⚠️ Not JSON string, using raw data:', parseError.message)
              parsedData = log
            }
          }
          
          // Extract all possible fields from API data
          const extractedFields = {
            id: parsedData.id || parsedData.log_id || log.id || log.log_id,
            timestamp: parsedData.timestamp || parsedData.time || parsedData.created_at || parsedData.log_time || log.timestamp || log.time || log.created_at,
            sessionId: parsedData.session_id || parsedData.sessionId || parsedData.ssh_session || log.session_id || log.sessionId || log.ssh_session,
            agent: parsedData.agent_id || parsedData.agent || parsedData.agent_name || log.agent_id || log.agent || log.agent_name,
            client: parsedData.client_id || parsedData.client || parsedData.client_name || log.client_id || log.client || log.client_name,
            
            // Build USER@HOST:PORT from individual fields
            ssh_root: parsedData.ssh_root || parsedData.user || parsedData.ssh_user || log.ssh_root || log.user || log.ssh_user,
            ssh_host: parsedData.ssh_host || parsedData.host || parsedData.hostname || log.ssh_host || log.host || log.hostname,
            ssh_port: parsedData.ssh_port || parsedData.port || log.ssh_port || log.port,
            
            direction: parsedData.direction || parsedData.type || parsedData.action || log.direction || log.type || log.action,
            command: parsedData.command || parsedData.cmd || parsedData.ssh_command || log.command || log.cmd || log.ssh_command,
            data: parsedData.data || parsedData.content || parsedData.output || log.data || log.content || log.output || ''
          }
          
          // Build USER@HOST:PORT format
          let userHostPort = 'API-No-Connection-Info'
          if (extractedFields.ssh_root || extractedFields.ssh_host || extractedFields.ssh_port) {
            const user = extractedFields.ssh_root || 'unknown'
            const host = extractedFields.ssh_host || 'unknown'
            const port = extractedFields.ssh_port || '22'
            userHostPort = `${user}@${host}:${port}`
          }
          
          console.log('Extracted fields from API:', extractedFields)
          console.log('Built USER@HOST:PORT:', userHostPort)
          
          const transformedLog = {
            id: extractedFields.id || `SSH-API-${index + 1}`,
            timestamp: extractedFields.timestamp || 'API-No-Timestamp',
            sessionId: extractedFields.sessionId || `api-session-${index + 1}`,
            agent: extractedFields.agent || 'API-Unknown-Agent',
            client: extractedFields.client || 'API-Unknown-Client',
            userHostPort: userHostPort,
            ssh_user: extractedFields.ssh_root || 'unknown',
            ssh_host: extractedFields.ssh_host || 'unknown',
            ssh_port: extractedFields.ssh_port || '22',
            direction: extractedFields.direction || 'API-Unknown-Direction',
            command: extractedFields.command || (extractedFields.direction === 'INPUT' ? '(no command)' : '(output)'),
            data: extractedFields.data || 'No data available'
          }
          
          console.log(`✅ FINAL TRANSFORMED RECORD ${index + 1}:`, transformedLog)
          console.log('SOURCE: 100% FROM API - NO DUMMY DATA')
          
          return transformedLog
        })
        
        console.log('=== FINAL SSH LOGS PROCESSING COMPLETE ===')
        console.log('Total records processed from API:', allSSHLogs.value.length)
        console.log('Sample records:', allSSHLogs.value.slice(0, 2))
        console.log('✅ ALL DATA IS FROM API ENDPOINT:', import.meta.env.VITE_API_BASE_URL + '/api/ssh-logs')
        console.log('✅ NO DUMMY DATA USED ANYWHERE')
        
      } catch (err) {
        console.error('=== SSH LOGS API ERROR ===')
        console.error('❌ Failed to fetch from API:', err)
        console.error('API URL that failed:', import.meta.env.VITE_API_BASE_URL + '/api/ssh-logs')
        console.error('Error details:', {
          status: err.response?.status,
          statusText: err.response?.statusText,
          data: err.response?.data,
          message: err.message,
          code: err.code
        })
        
        if (err.response?.status === 401) {
          error.value = 'Authentication required for SSH logs API. Please login.'
        } else if (err.response?.status === 404) {
          error.value = 'SSH logs API endpoint not found. Check server configuration.'
        } else if (err.code === 'ECONNREFUSED' || err.code === 'NETWORK_ERROR') {
          error.value = 'Cannot connect to SSH logs API. Check if relay server is running.'
        } else {
          error.value = `Failed to load SSH logs from API: ${err.message}`
        }
        
        // ENSURE NO DUMMY DATA ON ERROR
        allSSHLogs.value = []
        console.log('✅ Set empty array on error - NO DUMMY FALLBACK')
      } finally {
        loading.value = false
        console.log('=== SSH LOGS FETCH COMPLETE ===')
      }
    }

    const refreshData = () => {
      console.log('=== MANUAL SSH LOGS REFRESH - API ONLY ===')
      console.log('Clearing existing data and fetching fresh from API...')
      allSSHLogs.value = []
      fetchSSHLogs()
    }

    // Utility functions for SSH command logging
    const sortBy = (field) => {
      if (sortField.value === field) {
        sortDirection.value = sortDirection.value === 'asc' ? 'desc' : 'asc'
      } else {
        sortField.value = field
        sortDirection.value = 'desc'
      }
    }

    const getSortIcon = (field) => {
      if (sortField.value !== field) return 'fas fa-sort'
      return sortDirection.value === 'asc' ? 'fas fa-sort-up' : 'fas fa-sort-down'
    }

    const formatTimestamp = (timestamp) => {
      return new Date(timestamp).toLocaleString()
    }

    const handlePageChange = (page) => {
      currentPage.value = page
    }

    onMounted(() => {
      console.log('=== SSH LOGS COMPONENT MOUNTED ===')
      console.log('Initializing with ZERO dummy data')
      console.log('Will fetch ONLY from API:', import.meta.env.VITE_API_BASE_URL + '/api/ssh-logs')
      
      // ENSURE CLEAN START - NO DUMMY DATA
      allSSHLogs.value = []
      
      fetchSSHLogs()
      
      // Auto-refresh every 15 seconds for SSH logs
      setInterval(() => {
        console.log('=== SSH LOGS AUTO-REFRESH - API ONLY ===')
        fetchSSHLogs()
      }, 15000)
    })

    // Watch for search query changes and reset pagination
    watch(searchQuery, () => {
      currentPage.value = 1
    })

    return {
      allSSHLogs,
      paginatedSSHLogs,
      loading,
      error,
      currentPage,
      itemsPerPage,
      apiBaseUrl,
      searchQuery,
      filteredLogs,
      uniqueUsers,
      uniqueHosts,
      todayCommands,
      refreshData,
      handlePageChange,
      sortBy,
      getSortIcon,
      formatTimestamp,
      truncateData: (text, maxLength) => {
        if (!text) return ''
        if (text.length <= maxLength) return text
        return text.substring(0, maxLength) + '...'
      }
    }
  }
}
</script>

<style scoped>
.ssh-logs-table-container {
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
  flex-wrap: wrap;
  gap: 10px;
}

.header-actions {
  display: flex;
  gap: 10px;
  align-items: center;
  flex-wrap: wrap;
}

.search-container {
  position: relative;
}

.search-input {
  padding: 8px 35px 8px 12px;
  border: 1px solid #475569;
  border-radius: 5px;
  background: rgba(0, 0, 0, 0.3);
  color: #ffffff;
  min-width: 200px;
}

.search-input::placeholder {
  color: #94a3b8;
}

.search-icon {
  position: absolute;
  right: 10px;
  top: 50%;
  transform: translateY(-50%);
  color: #94a3b8;
}

.stats-container {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 15px;
  margin-bottom: 20px;
}

.stat-card {
  background: rgba(0, 0, 0, 0.3);
  padding: 15px;
  border-radius: 8px;
  border: 1px solid #475569;
  text-align: center;
}

.stat-value {
  font-size: 1.5rem;
  font-weight: bold;
  color: #00ff41;
  margin-bottom: 5px;
}

.stat-label {
  color: #94a3b8;
  font-size: 0.8rem;
}

.sortable {
  cursor: pointer;
  user-select: none;
  transition: background-color 0.3s ease;
}

.sortable:hover {
  background: rgba(0, 255, 65, 0.1);
}

.timestamp-cell {
  min-width: 150px;
}

.timestamp {
  font-weight: 500;
  color: #ffffff;
}

.session-cell {
  min-width: 200px;
}

.session-id {
  font-size: 0.8rem;
  color: #00ff41;
  font-family: monospace;
  font-weight: 500;
}

.user-info, .host-info {
  display: flex;
  align-items: center;
  gap: 5px;
}

.command-cell {
  max-width: 400px;
}

.command-cell-wide {
  max-width: 600px;
  min-width: 300px;
}

.command-text {
  font-family: 'Courier New', monospace;
  color: #ffffff;
  word-break: break-all;
}

.command-text-wide {
  font-family: 'Courier New', monospace;
  color: #ffffff;
  word-break: break-all;
  white-space: pre-wrap;
  max-height: 100px;
  overflow-y: auto;
}

.command-direction {
  font-size: 0.7rem;
  padding: 2px 6px;
  border-radius: 3px;
  display: inline-block;
  margin-top: 3px;
}

.command-direction.input {
  background: rgba(0, 255, 65, 0.2);
  color: #00ff41;
}

.command-direction.output {
  background: rgba(59, 130, 246, 0.2);
  color: #3b82f6;
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

/* SSH Commands Table Column Width Optimization */
.table th:nth-child(1), /* TIMESTAMP */
.table td:nth-child(1) {
  width: 15%;
  min-width: 140px;
}

.table th:nth-child(2), /* SESSION ID */
.table td:nth-child(2) {
  width: 20%;
  min-width: 180px;
}

.table th:nth-child(3), /* USER */
.table td:nth-child(3) {
  width: 10%;
  min-width: 80px;
}

.table th:nth-child(4), /* HOST */
.table td:nth-child(4) {
  width: 15%;
  min-width: 120px;
}

.table th:nth-child(5), /* COMMAND */
.table td:nth-child(5) {
  width: 40%;
  min-width: 300px;
}

.command-header {
  min-width: 300px !important;
  width: 40% !important;
}
.table td:nth-child(5) {
  width: 61%;
  min-width: 400px;
}

.data-cell {
  font-family: 'Courier New', monospace;
  background: var(--surface-light);
  padding: 0.75rem;
  border-radius: var(--radius-sm);
  font-size: 0.85rem;
  max-width: none;
  width: 100%;
}

.data-content {
  white-space: pre-wrap;
  word-break: break-word;
  line-height: 1.4;
  cursor: text;
  overflow: visible;
  max-height: none;
  min-height: 40px;
}

.data-content:hover {
  background: var(--surface-alt);
  color: var(--text-primary);
}
</style>
