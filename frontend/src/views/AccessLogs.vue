<template>
  <div class="access-logs">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>Access Logs</span>
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
        <el-form-item label="Client ID">
          <el-input v-model="filters.clientId" placeholder="Client ID" clearable />
        </el-form-item>
        <el-form-item label="Agent ID">
          <el-input v-model="filters.agentId" placeholder="Agent ID" clearable />
        </el-form-item>
        <el-form-item label="Username">
          <el-input v-model="filters.username" placeholder="Username" clearable />
        </el-form-item>
        <el-form-item label="Action">
          <el-select v-model="filters.action" placeholder="All Actions" clearable>
            <el-option label="All Actions" value="" />
            <el-option label="Connect" value="connect" />
            <el-option label="Disconnect" value="disconnect" />
            <el-option label="Command" value="command" />
            <el-option label="Login" value="login" />
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
        <el-table-column prop="client_name" label="Client Name" width="120">
          <template #default="{ row }">
            {{ row.client_name || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="username" label="Username" width="100">
          <template #default="{ row }">
            {{ row.username || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="agent_name" label="Agent Name" width="120">
          <template #default="{ row }">
            {{ row.agent_name || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="action" label="Action" width="100">
          <template #default="{ row }">
            <el-tag :type="getActionType(row.action)">{{ row.action }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="details" label="Details" min-width="200">
          <template #default="{ row }">
            <el-text truncated>{{ row.details || '-' }}</el-text>
          </template>
        </el-table-column>
        <el-table-column prop="ip_address" label="IP Address" width="120">
          <template #default="{ row }">
            {{ row.ip_address || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="user_agent" label="User Agent" min-width="200">
          <template #default="{ row }">
            <el-text truncated>{{ row.user_agent || '-' }}</el-text>
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

const filters = ref({
  clientId: '',
  agentId: '',
  username: '',
  action: ''
})

const pagination = ref({
  currentPage: 1,
  pageSize: 20,
  total: 0
})

const formatTimestamp = (timestamp) => {
  return new Date(timestamp).toLocaleString()
}

const getActionType = (action) => {
  const actionMap = {
    'connect': 'success',
    'disconnect': 'info',
    'command': 'warning',
    'login': 'primary'
  }
  return actionMap[action] || 'info'
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
    
    const response = await api.getAccessLogs(params)
    logs.value = response.data.logs || []
    pagination.value.total = response.data.total || logs.value.length
  } catch (error) {
    console.error('Failed to load access logs:', error)
    ElMessage.error('Failed to load access logs')
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
  
  const headers = ['Timestamp', 'Client Name', 'Username', 'Agent Name', 'Action', 'Details', 'IP Address', 'User Agent']
  const csvContent = [
    headers.join(','),
    ...logs.value.map(log => [
      formatTimestamp(log.timestamp),
      log.client_name || '',
      log.username || '',
      log.agent_name || '',
      log.action || '',
      `"${(log.details || '').replace(/"/g, '""')}"`,
      log.ip_address || '',
      `"${(log.user_agent || '').replace(/"/g, '""')}"`,
    ].join(','))
  ].join('\n')
  
  const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' })
  const link = document.createElement('a')
  link.href = URL.createObjectURL(blob)
  link.download = `access-logs-${new Date().toISOString().split('T')[0]}.csv`
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
