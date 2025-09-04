<template>
  <div class="agents">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>Agent Management</span>
          <div class="header-actions">
            <el-button type="primary" @click="loadAgents">
              <el-icon><Refresh /></el-icon>
              Refresh
            </el-button>
          </div>
        </div>
      </template>

      <!-- Stats -->
      <el-row :gutter="20" class="mb-4">
        <el-col :span="8">
          <div class="stat-card">
            <div class="stat-number">{{ stats.total }}</div>
            <div class="stat-label">Total Agents</div>
          </div>
        </el-col>
        <el-col :span="8">
          <div class="stat-card online">
            <div class="stat-number">{{ stats.online }}</div>
            <div class="stat-label">Online Agents</div>
          </div>
        </el-col>
        <el-col :span="8">
          <div class="stat-card offline">
            <div class="stat-number">{{ stats.offline }}</div>
            <div class="stat-label">Offline Agents</div>
          </div>
        </el-col>
      </el-row>

      <!-- Filters -->
      <el-form :model="filters" inline class="filter-form mb-4">
        <el-form-item label="Status">
          <el-select v-model="filters.status" placeholder="All Status" clearable>
            <el-option label="All Status" value="" />
            <el-option label="Online" value="online" />
            <el-option label="Offline" value="offline" />
          </el-select>
        </el-form-item>
        <el-form-item label="Agent Name">
          <el-input v-model="filters.name" placeholder="Search agent name..." clearable />
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
      <el-table :data="filteredAgents" v-loading="loading" style="width: 100%">
        <el-table-column prop="id" label="Agent ID" width="200">
          <template #default="{ row }">
            <el-text truncated>{{ row.id }}</el-text>
          </template>
        </el-table-column>
        <el-table-column prop="name" label="Name" width="150">
          <template #default="{ row }">
            {{ row.name || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="status" label="Status" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)">
              <el-icon><Connection /></el-icon>
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="platform" label="Platform" width="120">
          <template #default="{ row }">
            {{ row.platform || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="connected_at" label="Connected At" width="180">
          <template #default="{ row }">
            {{ row.connected_at ? formatTimestamp(row.connected_at) : '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="last_seen" label="Last Seen" width="180">
          <template #default="{ row }">
            {{ formatTimestamp(row.last_seen) }}
          </template>
        </el-table-column>
        <el-table-column prop="address" label="Address" width="150">
          <template #default="{ row }">
            {{ row.address || '-' }}
          </template>
        </el-table-column>
        <el-table-column label="Uptime" width="120">
          <template #default="{ row }">
            <span v-if="row.status === 'online' && row.connected_at">
              {{ calculateUptime(row.connected_at) }}
            </span>
            <span v-else>-</span>
          </template>
        </el-table-column>
        <el-table-column label="Actions" width="150" fixed="right">
          <template #default="{ row }">
            <el-button 
              type="info" 
              size="small" 
              @click="viewAgentDetails(row)"
            >
              <el-icon><View /></el-icon>
              Details
            </el-button>
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

    <!-- Agent Details Dialog -->
    <el-dialog v-model="detailsVisible" title="Agent Details" width="600px">
      <div v-if="selectedAgent">
        <el-descriptions :column="2" border>
          <el-descriptions-item label="Agent ID">{{ selectedAgent.id }}</el-descriptions-item>
          <el-descriptions-item label="Name">{{ selectedAgent.name || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Status">
            <el-tag :type="getStatusType(selectedAgent.status)">{{ selectedAgent.status }}</el-tag>
          </el-descriptions-item>
          <el-descriptions-item label="Platform">{{ selectedAgent.platform || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Address">{{ selectedAgent.address || '-' }}</el-descriptions-item>
          <el-descriptions-item label="Connected At">
            {{ selectedAgent.connected_at ? formatTimestamp(selectedAgent.connected_at) : '-' }}
          </el-descriptions-item>
          <el-descriptions-item label="Last Seen">
            {{ formatTimestamp(selectedAgent.last_seen) }}
          </el-descriptions-item>
          <el-descriptions-item label="Uptime">
            <span v-if="selectedAgent.status === 'online' && selectedAgent.connected_at">
              {{ calculateUptime(selectedAgent.connected_at) }}
            </span>
            <span v-else>-</span>
          </el-descriptions-item>
        </el-descriptions>
        
        <div v-if="selectedAgent.metadata" class="mt-4">
          <h4>Metadata:</h4>
          <el-code>{{ JSON.stringify(selectedAgent.metadata, null, 2) }}</el-code>
        </div>
      </div>
      
      <template #footer>
        <el-button @click="detailsVisible = false">Close</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { Refresh, Search, RefreshLeft, Connection, View } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import api from '../services/api'

const agents = ref([])
const loading = ref(false)
const detailsVisible = ref(false)
const selectedAgent = ref(null)

const stats = ref({
  total: 0,
  online: 0,
  offline: 0
})

const filters = ref({
  status: '',
  name: ''
})

const pagination = ref({
  currentPage: 1,
  pageSize: 20,
  total: 0
})

const filteredAgents = computed(() => {
  let filtered = agents.value

  if (filters.value.status) {
    filtered = filtered.filter(agent => agent.status === filters.value.status)
  }

  if (filters.value.name) {
    const searchTerm = filters.value.name.toLowerCase()
    filtered = filtered.filter(agent => 
      agent.name && agent.name.toLowerCase().includes(searchTerm)
    )
  }

  pagination.value.total = filtered.length
  
  const startIndex = (pagination.value.currentPage - 1) * pagination.value.pageSize
  const endIndex = startIndex + pagination.value.pageSize
  
  return filtered.slice(startIndex, endIndex)
})

const formatTimestamp = (timestamp) => {
  return new Date(timestamp).toLocaleString()
}

const getStatusType = (status) => {
  const statusMap = {
    'online': 'success',
    'offline': 'info',
    'error': 'danger'
  }
  return statusMap[status] || 'info'
}

const calculateUptime = (connectedAt) => {
  if (!connectedAt) return '-'
  
  const now = new Date()
  const connected = new Date(connectedAt)
  const diffMs = now - connected
  
  const days = Math.floor(diffMs / (1000 * 60 * 60 * 24))
  const hours = Math.floor((diffMs % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60))
  const minutes = Math.floor((diffMs % (1000 * 60 * 60)) / (1000 * 60))
  
  if (days > 0) {
    return `${days}d ${hours}h`
  } else if (hours > 0) {
    return `${hours}h ${minutes}m`
  } else {
    return `${minutes}m`
  }
}

const loadAgents = async () => {
  loading.value = true
  try {
    const response = await api.getAgents()
    agents.value = response.data.agents || []
    
    // Update stats
    updateStats()
  } catch (error) {
    console.error('Failed to load agents:', error)
    ElMessage.error('Failed to load agents')
  } finally {
    loading.value = false
  }
}

const updateStats = () => {
  stats.value.total = agents.value.length
  stats.value.online = agents.value.filter(agent => agent.status === 'online').length
  stats.value.offline = agents.value.filter(agent => agent.status === 'offline').length
}

const applyFilters = () => {
  pagination.value.currentPage = 1
}

const clearFilters = () => {
  Object.keys(filters.value).forEach(key => {
    filters.value[key] = ''
  })
  pagination.value.currentPage = 1
}

const handleSizeChange = (size) => {
  pagination.value.pageSize = size
  pagination.value.currentPage = 1
}

const handleCurrentChange = (page) => {
  pagination.value.currentPage = page
}

const viewAgentDetails = (agent) => {
  selectedAgent.value = agent
  detailsVisible.value = true
}

onMounted(() => {
  loadAgents()
  
  // Auto-refresh every 30 seconds
  setInterval(loadAgents, 30000)
})
</script>

<style scoped>
.agents {
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

.stat-card.online {
  background: linear-gradient(135deg, #84fab0 0%, #8fd3f4 100%);
}

.stat-card.offline {
  background: linear-gradient(135deg, #ffecd2 0%, #fcb69f 100%);
  color: #333;
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

:deep(.el-code) {
  white-space: pre-wrap;
  background: #f5f5f5;
  padding: 10px;
  border-radius: 4px;
  font-family: 'Courier New', monospace;
  font-size: 12px;
  max-height: 200px;
  overflow-y: auto;
}
</style>
