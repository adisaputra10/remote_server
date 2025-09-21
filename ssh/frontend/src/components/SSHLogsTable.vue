<template>
  <div class="ssh-logs-table-container">
    <div class="table-header">
      <h2 class="table-title">SSH Commands</h2>
      <button class="btn btn-primary" @click="refreshData">
        <i class="fas fa-sync-alt"></i>
        Refresh
      </button>
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
      No SSH logs from API - Only real data shown
      <br>
      <small>API: {{ apiBaseUrl }}/api/ssh-logs</small>
      <br>
      <small>No dummy/fallback data used</small>
    </div>
    
    <div v-else class="table-wrapper">
      <div class="table-container">
        <table class="table">
                    <thead>
            <tr>
              <th>TIMESTAMP</th>
              <th>AGENT</th>
              <th>CLIENT</th>
              <th>USER@HOST:PORT</th>
              <th>DIRECTION</th>
              <th>COMMAND</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="log in paginatedSSHLogs" :key="log.id">
              <td>{{ log.timestamp }}</td>
              <td>
                <span class="badge badge-success">
                  {{ log.agent }}
                </span>
              </td>
              <td>
                <span class="badge badge-primary">
                  {{ log.client }}
                </span>
              </td>
              <td>{{ log.userHostPort }}</td>
              <td>
                <span class="badge badge-warning">
                  {{ log.direction }}
                </span>
              </td>
              <td class="command-cell">{{ log.command }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      
      <Pagination
        :current-page="currentPage"
        :total-items="allSSHLogs.length"
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

    const paginatedSSHLogs = computed(() => {
      const start = (currentPage.value - 1) * itemsPerPage.value
      const end = start + itemsPerPage.value
      return allSSHLogs.value.slice(start, end)
    })

    const apiBaseUrl = computed(() => {
      return import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080'
    })

    const fetchSSHLogs = async () => {
      try {
        loading.value = true
        error.value = null
        
        console.log('=== SSH LOGS - ENSURING 100% API DATA ===')
        console.log('API Base URL:', import.meta.env.VITE_API_BASE_URL)
        console.log('Full SSH Logs API URL:', import.meta.env.VITE_API_BASE_URL + '/api/ssh-logs')
        console.log('Auth token:', localStorage.getItem('auth_token') ? 'Present' : 'Missing')
        console.log('ABSOLUTELY NO DUMMY DATA - ONLY REAL API CALLS')
        
        // CLEAR ANY EXISTING DATA FIRST
        allSSHLogs.value = []
        console.log('Cleared existing data, making fresh API call...')
        
        const response = await apiService.getSSHLogs()
        console.log('=== RAW SSH LOGS API RESPONSE ===')
        console.log('Response status:', response.status)
        console.log('Response config URL:', response.config?.url)
        console.log('Response headers:', response.headers)
        console.log('Raw response data:', response.data)
        console.log('Data type:', typeof response.data)
        console.log('Is Array:', Array.isArray(response.data))
        console.log('Data length:', response.data ? response.data.length : 0)
        
        // VALIDATE API RESPONSE
        if (!response.data) {
          console.error('❌ API returned null/undefined data')
          allSSHLogs.value = []
          error.value = 'API returned no data - not dummy data'
          return
        }
        
        if (!Array.isArray(response.data)) {
          console.error('❌ API data is not an array:', response.data)
          allSSHLogs.value = []
          error.value = 'API returned invalid data format - not dummy data'
          return
        }
        
        if (response.data.length === 0) {
          console.warn('⚠️ API returned empty array - no SSH logs available')
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
            command: parsedData.command || parsedData.cmd || parsedData.ssh_command || log.command || log.cmd || log.ssh_command
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
            direction: extractedFields.direction || 'API-Unknown-Direction',
            command: extractedFields.command || (extractedFields.direction === 'INPUT' ? '(no command)' : '(output)')
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

    return {
      allSSHLogs,
      paginatedSSHLogs,
      loading,
      error,
      currentPage,
      itemsPerPage,
      apiBaseUrl,
      refreshData,
      handlePageChange
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
  width: 12%;
  min-width: 120px;
}

.table th:nth-child(2), /* AGENT */
.table td:nth-child(2) {
  width: 10%;
  min-width: 80px;
}

.table th:nth-child(3), /* CLIENT */
.table td:nth-child(3) {
  width: 10%;
  min-width: 80px;
}

.table th:nth-child(4), /* USER@HOST:PORT */
.table td:nth-child(4) {
  width: 12%;
  min-width: 100px;
  font-size: 0.8rem;
}

.table th:nth-child(5), /* DIRECTION */
.table td:nth-child(5) {
  width: 8%;
  min-width: 70px;
}

.table th:nth-child(6), /* COMMAND */
.table td:nth-child(6) {
  width: 48%;
  min-width: 200px;
}

.command-cell {
  font-family: 'Courier New', monospace;
  background: var(--surface-alt);
  padding: 0.5rem;
  border-radius: var(--radius-sm);
  font-size: 0.875rem;
  max-width: none; /* Remove max-width limit */
  overflow: visible; /* Allow text to be fully visible */
  text-overflow: clip;
  white-space: normal; /* Allow wrapping if needed */
  word-break: break-word;
}
</style>
