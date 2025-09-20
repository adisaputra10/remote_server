<template>
  <div class="queries-table-container">
    <div class="table-header">
      <h2 class="table-title">Database Queries</h2>
      <button class="btn btn-primary" @click="refreshData">
        <i class="fas fa-sync-alt"></i>
        Refresh
      </button>
    </div>
    
    <div v-if="loading" class="loading-state">
      <i class="fas fa-spinner fa-spin"></i>
      Loading queries...
    </div>
    
    <div v-else-if="error" class="error-state">
      <i class="fas fa-exclamation-triangle"></i>
      {{ error }}
    </div>
    
    <div v-else-if="allQueries.length === 0" class="empty-state">
      <i class="fas fa-database"></i>
      No query data from API ({{ apiBaseUrl }}/api/tunnel-logs)
      <br>
      <small>Check relay server connection</small>
    </div>
    
    <div v-else class="table-wrapper">
      <div class="table-container">
        <table class="table">
          <thead>
            <tr>
              <th>TIMESTAMP</th>
              <th>AGENT ID</th>
              <th>CLIENT ID</th>
              <th>OPERATION</th>
              <th>QUERY</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="query in paginatedQueries" :key="query.id">
              <td>{{ query.timestamp }}</td>
              <td>{{ query.agentId }}</td>
              <td>{{ query.clientId }}</td>
              <td>
                <span :class="['badge', 'badge-' + query.operationClass]">
                  {{ query.operation }}
                </span>
              </td>
              <td class="query-cell">
                <div class="query-text">
                  {{ query.query }}
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      
      <Pagination
        :current-page="currentPage"
        :total-items="allQueries.length"
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
  name: 'QueriesTable',
  components: {
    Pagination
  },
  setup() {
    const allQueries = ref([])
    const loading = ref(false)
    const error = ref(null)
    const currentPage = ref(1)
    const itemsPerPage = ref(20)

    const paginatedQueries = computed(() => {
      const start = (currentPage.value - 1) * itemsPerPage.value
      const end = start + itemsPerPage.value
      return allQueries.value.slice(start, end)
    })

    const apiBaseUrl = computed(() => {
      return import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080'
    })

    const fetchQueries = async () => {
      try {
        loading.value = true
        error.value = null
        
        console.log('=== QueriesTable Debug Info ===')
        console.log('API Base URL:', import.meta.env.VITE_API_BASE_URL)
        console.log('Full API URL:', import.meta.env.VITE_API_BASE_URL + '/api/tunnel-logs')
        console.log('Auth token:', localStorage.getItem('auth_token') ? 'Present' : 'Missing')
        
        const response = await apiService.getTunnelLogs()
        console.log('Tunnel logs response:', response.data)
        console.log('Response status:', response.status)
        console.log('Number of records:', response.data ? response.data.length : 0)
        
        if (!response.data || response.data.length === 0) {
          console.warn('No tunnel logs data received from API')
          allQueries.value = []
          return
        }
        
        // Transform data to match the expected format
        allQueries.value = (response.data || []).map((query, index) => {
          console.log('=== Raw query data ===', query) 
          console.log('Available fields:', Object.keys(query))
          
          // Parse JSON if query is JSON string, otherwise use direct fields
          let parsedData = query
          if (typeof query === 'string') {
            try {
              parsedData = JSON.parse(query)
            } catch {
              parsedData = query
            }
          }
          
          // Extract data from parsed JSON or direct fields
          const extractedData = {
            timestamp: parsedData.timestamp || parsedData.time || parsedData.created_at || parsedData.log_time || query.timestamp || query.time || query.created_at || 'Unknown',
            agentId: parsedData.agent_id || parsedData.agent || parsedData.target_agent || parsedData.source || query.agent_id || query.agent || query.target_agent || 'Unknown',
            clientId: parsedData.client_id || parsedData.client || parsedData.tunnel_id || parsedData.destination || query.client_id || query.client || query.tunnel_id || 'Unknown',
            operation: parsedData.operation || parsedData.action || parsedData.type || query.operation || query.action || query.type || 'SELECT',
            queryText: parsedData.query_text || parsedData.sql || parsedData.query || parsedData.command || parsedData.description || query.query_text || query.sql || query.query || query.command || 'No query available'
          }
          
          console.log('Extracted data:', extractedData)
          
          // Determine operation class for styling
          let operationClass = 'info'
          const upperOperation = extractedData.operation.toString().toUpperCase()
          
          if (upperOperation.includes('SELECT')) {
            operationClass = 'success'
          } else if (upperOperation.includes('UPDATE') || upperOperation.includes('INSERT') || upperOperation.includes('CREATE')) {
            operationClass = 'primary'
          } else if (upperOperation.includes('SHOW') || upperOperation.includes('DESC')) {
            operationClass = 'warning'
          }
          
          const transformedQuery = {
            id: parsedData.id || parsedData.log_id || query.id || query.log_id || `QRY-${index + 1}`,
            timestamp: extractedData.timestamp,
            agentId: extractedData.agentId,
            clientId: extractedData.clientId,
            operation: extractedData.operation,
            operationClass: operationClass,
            query: extractedData.queryText
          }
          
          console.log('=== Final transformed query ===', transformedQuery)
          return transformedQuery
        })
        
        console.log('All transformed queries:', allQueries.value.slice(0, 3))
        
      } catch (err) {
        console.error('Error fetching queries:', err)
        console.error('API URL:', import.meta.env.VITE_API_BASE_URL + '/api/tunnel-logs')
        console.error('Error details:', {
          status: err.response?.status,
          statusText: err.response?.statusText,
          data: err.response?.data,
          message: err.message
        })
        
        if (err.response?.status === 401) {
          error.value = 'Authentication required. Please login to access query data.'
        } else if (err.response?.status === 404) {
          error.value = 'API endpoint not found. Please check relay server configuration.'
        } else if (err.code === 'ECONNREFUSED' || err.code === 'NETWORK_ERROR') {
          error.value = 'Cannot connect to relay server. Please check if server is running.'
        } else {
          error.value = `Failed to load queries data: ${err.message}`
        }
        allQueries.value = []
      } finally {
        loading.value = false
      }
    }

    const refreshData = () => {
      console.log('Refreshing queries data...')
      fetchQueries()
    }

    const handlePageChange = (page) => {
      currentPage.value = page
    }

    onMounted(() => {
      fetchQueries()
      
      // Auto-refresh every 15 seconds for queries
      setInterval(fetchQueries, 15000)
    })

    return {
      allQueries,
      paginatedQueries,
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
.queries-table-container {
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

.query-cell {
  font-family: 'Courier New', monospace;
  font-size: 0.875rem;
  max-width: 400px;
  padding: 0.75rem;
}

.query-text {
  font-family: 'Courier New', monospace;
  background: var(--surface-alt);
  padding: 0.5rem;
  border-radius: var(--radius-sm);
  color: var(--text-primary);
  word-break: break-word;
  max-height: 100px;
  overflow-y: auto;
}
</style>
