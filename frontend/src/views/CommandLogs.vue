<template>
  <div class="command-logs">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>Command Logs</span>
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

      <!-- Stats Cards -->
      <el-row :gutter="20" class="mb-4">
        <el-col :span="6">
          <div class="stat-card">
            <div class="stat-number">{{ stats.total }}</div>
            <div class="stat-label">Total Commands</div>
          </div>
        </el-col>
        <el-col :span="6">
          <div class="stat-card">
            <div class="stat-number">{{ stats.executed }}</div>
            <div class="stat-label">Executed</div>
          </div>
        </el-col>
        <el-col :span="6">
          <div class="stat-card">
            <div class="stat-number">{{ stats.completed }}</div>
            <div class="stat-label">Completed</div>
          </div>
        </el-col>
        <el-col :span="6">
          <div class="stat-card">
            <div class="stat-number">{{ stats.failed }}</div>
            <div class="stat-label">Failed</div>
          </div>
        </el-col>
      </el-row>

      <!-- Filters -->
      <el-form :model="filters" inline class="filter-form mb-4">
        <el-form-item label="Session ID">
          <el-input v-model="filters.sessionId" placeholder="Session ID" clearable />
        </el-form-item>
        <el-form-item label="Client ID">
          <el-input v-model="filters.clientId" placeholder="Client ID" clearable />
        </el-form-item>
        <el-form-item label="Agent ID">
          <el-input v-model="filters.agentId" placeholder="Agent ID" clearable />
        </el-form-item>
        <el-form-item label="Status">
          <el-select v-model="filters.status" placeholder="All Status" clearable>
            <el-option label="All Status" value="" />
            <el-option label="Sent" value="sent" />
            <el-option label="Executed" value="executed" />
            <el-option label="Completed" value="completed" />
            <el-option label="Failed" value="failed" />
          </el-select>
        </el-form-item>
        <el-form-item label="Command">
          <el-input v-model="filters.command" placeholder="Search command..." clearable />
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
        <el-table-column prop="session_id" label="Session ID" width="150">
          <template #default="{ row }">
            <el-text truncated>{{ row.session_id || '-' }}</el-text>
          </template>
        </el-table-column>
        <el-table-column prop="client_name" label="Client Name" width="120">
          <template #default="{ row }">
            {{ row.client_name || '-' }}
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
        <el-table-column prop="command" label="Command" min-width="200">
          <template #default="{ row }">
            <el-text truncated>{{ row.command || '-' }}</el-text>
          </template>
        </el-table-column>
        <el-table-column prop="output" label="Output" min-width="200">
          <template #default="{ row }">
            <el-text truncated>{{ row.output || '-' }}</el-text>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="Status" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="duration_ms" label="Duration" width="100">
          <template #default="{ row }">
            {{ row.duration_ms }}ms
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
import { Download, Refresh, Search, RefreshLeft } from '@element-plus/icons-vue'
import api from '../services/api'

const logs = ref([])
const loading = ref(false)

const stats = ref({
  total: 0,
  executed: 0,
  completed: 0,
  failed: 0
})

const filters = ref({
  sessionId: '',
  clientId: '',
  agentId: '',
  status: '',
  command: ''
})

const pagination = ref({
  currentPage: 1,
  pageSize: 20,
  total: 0
})

const formatTimestamp = (timestamp) => {
  return new Date(timestamp).toLocaleString()
}

const getStatusType = (status) => {
  const statusMap = {
    'completed': 'success',
    'failed': 'danger',
    'sent': 'warning',
    'executed': 'info'
  }
  return statusMap[status] || 'info'
}

const loadLogs = async () => {
  loading.value = true
  try {
    const params = {
      limit: pagination.value.pageSize,
      offset: (pagination.value.currentPage - 1) * pagination.value.pageSize,
      ...filters.value
    }
    
    // Remove empty filters
    Object.keys(params).forEach(key => {
      if (params[key] === '') delete params[key]
    })
    
    const response = await api.getCommandLogs(params)
    logs.value = response.data.logs || []
    pagination.value.total = response.data.total || logs.value.length
    
    // Update stats
    updateStats()
  } catch (error) {
    console.error('Failed to load logs:', error)
    ElMessage.error('Failed to load command logs')
  } finally {
    loading.value = false
  }
}

const updateStats = () => {
  stats.value.total = logs.value.length
  stats.value.executed = logs.value.filter(log => log.status === 'executed').length
  stats.value.completed = logs.value.filter(log => log.status === 'completed').length
  stats.value.failed = logs.value.filter(log => log.status === 'failed').length
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
  
  const headers = ['Timestamp', 'Session ID', 'Client Name', 'Agent Name', 'Username', 'Command', 'Output', 'Status', 'Duration (ms)']
  const csvContent = [
    headers.join(','),
    ...logs.value.map(log => [
      formatTimestamp(log.timestamp),
      log.session_id || '',
      log.client_name || '',
      log.agent_name || '',
      log.username || '',
      `"${(log.command || '').replace(/"/g, '""')}"`,
      `"${(log.output || '').replace(/"/g, '""')}"`,
      log.status || '',
      log.duration_ms || ''
    ].join(','))
  ].join('\n')
  
  const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' })
  const link = document.createElement('a')
  link.href = URL.createObjectURL(blob)
  link.download = `command-logs-${new Date().toISOString().split('T')[0]}.csv`
  link.click()
}

onMounted(() => {
  loadLogs()
})
</script>

<style scoped>
.command-logs {
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

.stat-card {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  padding: 20px;
  border-radius: 8px;
  text-align: center;
}

.stat-number {
  font-size: 32px;
  font-weight: bold;
  line-height: 1;
}

.stat-label {
  font-size: 14px;
  opacity: 0.9;
  margin-top: 5px;
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
