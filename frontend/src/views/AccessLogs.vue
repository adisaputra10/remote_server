<template>
  <div class="query-logs">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>Query Logs</span>
          <div class="header-actions">
            <el-button type="success" @click="exportToCSV" :disabled="!logs.length">
              <el-icon><Download /></el-icon>
              Export CSV
            </el-button>
            <el-button type="primary" @click="loadLogs">
              <el-icon><Refresh /></el-icon>
              Refresh
            </el-button>
          </div>
        </div>
      </template>

      <!-- Filters -->
      <el-form :model="filters" inline class="filter-form mb-4">
        <el-form-item label="Client Name">
          <el-input v-model="filters.agentId" placeholder="Client Name" clearable />
        </el-form-item>
        <el-form-item label="Username">
          <el-input v-model="filters.username" placeholder="Username" clearable />
        </el-form-item>
        <el-form-item label="Event Type">
          <el-select v-model="filters.eventType" placeholder="All Types" clearable>
            <el-option label="All Types" value="" />
            <el-option label="CMD_EXEC" value="CMD_EXEC" />
            <el-option label="DB_COMMAND" value="DB_COMMAND" />
            <el-option label="AGENT_START" value="AGENT_START" />
            <el-option label="AGENT_CONNECT" value="AGENT_CONNECT" />
            <el-option label="AGENT_DISCONNECT" value="AGENT_DISCONNECT" />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="applyFilters">
            <el-icon><Search /></el-icon>
            Filter
          </el-button>
          <el-button @click="clearFilters">
            <el-icon><RefreshLeft /></el-icon>
            Clear
          </el-button>
        </el-form-item>
      </el-form>

      <!-- Table -->
      <el-table :data="logs" v-loading="loading" style="width: 100%">
        <el-table-column prop="timestamp" label="Timestamp" width="180">
          <template #default="{ row }">
            {{ formatTimestamp(row.timestamp) }}
          </template>
        </el-table-column>
        <el-table-column prop="agent_id" label="Client Name" width="120">
          <template #default="{ row }">
            {{ row.agent_id || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="agent_name" label="Agent Name" width="120">
          <template #default="{ row }">
            {{ row.agent_name || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="username" label="Username" width="100">
          <template #default="{ row }">
            {{ row.username || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="protocol" label="Protocol" width="100">
          <template #default="{ row }">
            {{ row.protocol || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="event_type" label="Event Type" width="120">
          <template #default="{ row }">
            <el-tag :type="getQueryType(row.event_type)" size="small">
              {{ row.event_type || '-' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="command" label="Command/Query" min-width="300">
          <template #default="{ row }">
            <el-tooltip effect="dark" :content="row.command" placement="top">
              <el-text truncated>{{ row.command || '-' }}</el-text>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column prop="session_id" label="Session ID" width="150">
          <template #default="{ row }">
            {{ row.session_id || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="affected_rows" label="Rows" width="80">
          <template #default="{ row }">
            {{ row.affected_rows || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="status" label="Status" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)" size="small">
              {{ row.status || '-' }}
            </el-tag>
          </template>
        </el-table-column>
      </el-table>

      <!-- Pagination -->
      <div class="pagination-container mt-4">
        <el-pagination
          v-model:current-page="pagination.currentPage"
          v-model:page-size="pagination.pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="pagination.total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Download, Refresh, Search, RefreshLeft } from '@element-plus/icons-vue'
import api from '../services/api'

const logs = ref([])
const loading = ref(false)

const filters = ref({
  agentId: '',
  username: '',
  eventType: ''
})

const pagination = ref({
  currentPage: 1,
  pageSize: 20,
  total: 0
})

const formatTimestamp = (timestamp) => {
  return new Date(timestamp).toLocaleString()
}

const getQueryType = (eventType) => {
  const typeMap = {
    'CMD_EXEC': 'primary',
    'DB_COMMAND': 'success',
    'AGENT_START': 'info',
    'AGENT_CONNECT': 'success',
    'AGENT_DISCONNECT': 'warning'
  }
  return typeMap[eventType] || 'info'
}

const getStatusType = (status) => {
  const statusMap = {
    'success': 'success',
    'error': 'danger',
    'warning': 'warning'
  }
  return statusMap[status] || 'info'
}

const loadLogs = async () => {
  loading.value = true
  try {
    const params = {
      limit: pagination.value.pageSize,
      offset: (pagination.value.currentPage - 1) * pagination.value.pageSize
    }
    
    // Map frontend filter names to backend parameter names
    if (filters.value.agentId) params.agent_id = filters.value.agentId
    if (filters.value.username) params.username = filters.value.username
    if (filters.value.eventType) params.event_type = filters.value.eventType
    
    const response = await api.getQueryLogs(params)
    logs.value = response.data.logs || []
    pagination.value.total = response.data.total || logs.value.length
  } catch (error) {
    console.error('Failed to load query logs:', error)
    ElMessage.error('Failed to load query logs')
  } finally {
    loading.value = false
  }
}

const applyFilters = () => {
  pagination.value.currentPage = 1
  loadLogs()
}

const clearFilters = () => {
  Object.keys(filters.value).forEach(key => {
    filters.value[key] = ''
  })
  pagination.value.currentPage = 1
  loadLogs()
}

const handleSizeChange = (size) => {
  pagination.value.pageSize = size
  pagination.value.currentPage = 1
  loadLogs()
}

const handleCurrentChange = (page) => {
  pagination.value.currentPage = page
  loadLogs()
}

const exportToCSV = () => {
  if (!logs.value.length) return
  
  const headers = ['Timestamp', 'Client Name', 'Client ID', 'Username', 'Database', 'Query Type', 'Query Text', 'Execution Time (ms)', 'Affected Rows', 'Status']
  const csvContent = [
    headers.join(','),
    ...logs.value.map(log => [
      formatTimestamp(log.timestamp),
      log.agent_id || '',
      log.client_id || '', 
      log.username || '',
      log.database_name || '',
      log.query_type || '',
      `"${(log.query_text || '').replace(/"/g, '""')}"`,
      log.execution_time || '',
      log.affected_rows || '',
      log.status || ''
    ].join(','))
  ].join('\n')
  
  const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' })
  const link = document.createElement('a')
  link.href = URL.createObjectURL(blob)
  link.download = `query-logs-${new Date().toISOString().split('T')[0]}.csv`
  link.click()
}

onMounted(() => {
  loadLogs()
})
</script>

<style scoped>
.access-logs {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.header-actions {
  display: flex;
  gap: 10px;
}

.filter-form {
  background: #f8f9fa;
  padding: 20px;
  border-radius: 8px;
}

.pagination-container {
  display: flex;
  justify-content: center;
  margin-top: 20px;
}

.mb-4 {
  margin-bottom: 20px;
}

.mt-4 {
  margin-top: 20px;
}
</style>
