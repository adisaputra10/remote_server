<template>
  <div class="logs-table-container">
    <div class="table-header">
      <h2 class="table-title">Connection Logs</h2>
      <button class="btn btn-primary" @click="refreshData">
        <i class="fas fa-sync-alt"></i>
        Refresh
      </button>
    </div>
    
    <div v-if="loading" class="loading-state">
      <i class="fas fa-spinner fa-spin"></i>
      Loading logs...
    </div>
    
    <div v-else-if="error" class="error-state">
      <i class="fas fa-exclamation-triangle"></i>
      {{ error }}
    </div>
    
    <div v-else-if="allLogs.length === 0" class="empty-state">
      <i class="fas fa-list"></i>
      No connection logs from API
      <br>
      <small>Check relay server connection and logs endpoint</small>
    </div>
    
    <div v-else class="table-wrapper">
      <div class="table-container">
        <table class="table">
          <thead>
            <tr>
              <th>TIMESTAMP</th>
              <th>TYPE</th>
              <th>EVENT</th>
              <th>AGENT ID</th>
              <th>CLIENT ID</th>
              <th>DETAILS</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="log in paginatedLogs" :key="log.id">
              <td>{{ log.timestamp }}</td>
              <td>
                <span :class="['badge', 'badge-' + log.typeClass]">
                  {{ log.type }}
                </span>
              </td>
              <td>{{ log.event }}</td>
              <td>{{ log.agentId }}</td>
              <td>{{ log.clientId }}</td>
              <td>{{ log.details }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      
      <Pagination
        :current-page="currentPage"
        :total-items="allLogs.length"
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
  name: 'LogsTable',
  components: {
    Pagination
  },
  setup() {
    const allLogs = ref([])
    const loading = ref(false)
    const error = ref(null)
    const currentPage = ref(1)
    const itemsPerPage = ref(20)

    const paginatedLogs = computed(() => {
      const start = (currentPage.value - 1) * itemsPerPage.value
      const end = start + itemsPerPage.value
      return allLogs.value.slice(start, end)
    })

    const fetchLogs = async () => {
      try {
        loading.value = true
        error.value = null
        
        console.log('=== LogsTable Fetching Real API Data ===')
        console.log('API Base URL:', import.meta.env.VITE_API_BASE_URL)
        console.log('Full API URL:', import.meta.env.VITE_API_BASE_URL + '/api/logs')
        console.log('Auth token:', localStorage.getItem('auth_token') ? 'Present' : 'Missing')
        console.log('NO DUMMY DATA - ONLY API DATA')
        
        const response = await apiService.getConnectionLogs()
        console.log('=== RAW API RESPONSE ===')
        console.log('Response status:', response.status)
        console.log('Response headers:', response.headers)
        console.log('Raw response data:', response.data)
        console.log('Data type:', typeof response.data)
        console.log('Is Array:', Array.isArray(response.data))
        console.log('Number of records:', response.data ? response.data.length : 0)
        
        if (!response.data) {
          console.error('API returned null/undefined data')
          allLogs.value = []
          error.value = 'API returned no data'
          return
        }
        
        if (!Array.isArray(response.data)) {
          console.error('API data is not an array:', response.data)
          allLogs.value = []
          error.value = 'API returned invalid data format'
          return
        }
        
        if (response.data.length === 0) {
          console.warn('API returned empty array')
          allLogs.value = []
          return
        }
        
        // NO DUMMY DATA - ONLY TRANSFORM API DATA
        allLogs.value = response.data.map((log, index) => {
          console.log(`=== RAW LOG RECORD ${index + 1} ===`, log)
          console.log('Available fields:', Object.keys(log))
          
          // Parse JSON if log is JSON string, otherwise use direct fields
          let parsedData = log
          if (typeof log === 'string') {
            try {
              parsedData = JSON.parse(log)
              console.log('Parsed JSON data:', parsedData)
            } catch {
              console.log('Not JSON string, using as-is')
              parsedData = log
            }
          }
          
          let typeClass = 'info'
          let logType = 'client'
          
          // Determine log type from various fields
          const type = parsedData.type || parsedData.event_type || parsedData.source_type || parsedData.category || ''
          const typeUpper = type.toString().toUpperCase()
          
          if (typeUpper.includes('AGENT') || parsedData.agent_id || parsedData.agent) {
            typeClass = 'primary'
            logType = 'agent'
          } else if (typeUpper.includes('CLIENT') || parsedData.client_id || parsedData.client) {
            typeClass = 'success'
          }
          
          const transformedLog = {
            id: parsedData.id || parsedData.log_id || log.id || log.log_id || `LOG-${index + 1}`,
            timestamp: parsedData.timestamp || parsedData.time || parsedData.created_at || parsedData.log_time || log.timestamp || log.time || log.created_at || 'Unknown',
            type: logType,
            typeClass: typeClass,
            event: parsedData.event || parsedData.action || parsedData.status || parsedData.event_name || parsedData.operation || log.event || log.action || log.status || 'unknown',
            agentId: parsedData.agent_id || parsedData.agent || parsedData.target_agent || parsedData.source_agent || log.agent_id || log.agent || log.target_agent || '-',
            clientId: parsedData.client_id || parsedData.client || parsedData.target_client || parsedData.destination || log.client_id || log.client || log.target_client || '-',
            details: parsedData.details || parsedData.message || parsedData.description || parsedData.info || parsedData.data || log.details || log.message || log.description || '-'
          }
          
          console.log(`=== TRANSFORMED LOG ${index + 1} ===`, transformedLog)
          return transformedLog
        })
        
        console.log('=== FINAL TRANSFORMED DATA ===')
        console.log('Total logs processed:', allLogs.value.length)
        console.log('Sample logs:', allLogs.value.slice(0, 2))
        console.log('ALL DATA IS FROM API - NO DUMMY DATA')
        
      } catch (err) {
        console.error('=== API ERROR ===')
        console.error('Error fetching logs:', err)
        console.error('API URL:', import.meta.env.VITE_API_BASE_URL + '/api/logs')
        console.error('Error details:', {
          status: err.response?.status,
          statusText: err.response?.statusText,
          data: err.response?.data,
          message: err.message
        })
        
        if (err.response?.status === 401) {
          error.value = 'Authentication required. Please login to access connection logs.'
        } else if (err.response?.status === 404) {
          error.value = 'API endpoint not found. Please check relay server configuration.'
        } else if (err.code === 'ECONNREFUSED' || err.code === 'NETWORK_ERROR') {
          error.value = 'Cannot connect to relay server. Please check if server is running.'
        } else {
          error.value = `Failed to load connection logs: ${err.message}`
        }
        allLogs.value = []
      } finally {
        loading.value = false
      }
    }

    const refreshData = () => {
      console.log('=== MANUAL REFRESH - FETCHING FROM API ONLY ===')
      fetchLogs()
    }

    const handlePageChange = (page) => {
      currentPage.value = page
    }

    onMounted(() => {
      console.log('=== COMPONENT MOUNTED - FETCHING REAL API DATA ===')
      console.log('NO DUMMY DATA WILL BE USED')
      fetchLogs()
      
      // Auto-refresh every 10 seconds for logs
      setInterval(() => {
        console.log('=== AUTO-REFRESH - FETCHING FROM API ===')
        fetchLogs()
      }, 10000)
    })

    return {
      allLogs,
      paginatedLogs,
      loading,
      error,
      currentPage,
      itemsPerPage,
      refreshData,
      handlePageChange
    }
  }
}
</script>

<style scoped>
.logs-table-container {
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
</style>
