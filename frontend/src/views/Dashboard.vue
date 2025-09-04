<template>
  <div class="dashboard">
    <el-row :gutter="20" class="mb-4">
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <el-icon class="stat-icon primary"><Connection /></el-icon>
            <div class="stat-details">
              <div class="stat-number">{{ stats.activeConnections }}</div>
              <div class="stat-label">Active Connections</div>
            </div>
          </div>
        </el-card>
      </el-col>
      
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <el-icon class="stat-icon success"><Document /></el-icon>
            <div class="stat-details">
              <div class="stat-number">{{ stats.totalCommands }}</div>
              <div class="stat-label">Total Commands</div>
            </div>
          </div>
        </el-card>
      </el-col>
      
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <el-icon class="stat-icon warning"><User /></el-icon>
            <div class="stat-details">
              <div class="stat-number">{{ stats.activeUsers }}</div>
              <div class="stat-label">Active Users</div>
            </div>
          </div>
        </el-card>
      </el-col>
      
      <el-col :span="6">
        <el-card class="stat-card">
          <div class="stat-content">
            <el-icon class="stat-icon danger"><Monitor /></el-icon>
            <div class="stat-details">
              <div class="stat-number">{{ stats.activeAgents }}</div>
              <div class="stat-label">Active Agents</div>
            </div>
          </div>
        </el-card>
      </el-col>
    </el-row>

    <el-row :gutter="20">
      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>Recent Command Logs</span>
              <el-button type="primary" size="small" @click="$router.push('/command-logs')">
                View All
              </el-button>
            </div>
          </template>
          <el-table :data="recentCommands" style="width: 100%" v-loading="loadingCommands">
            <el-table-column prop="timestamp" label="Time" width="120">
              <template #default="{ row }">
                {{ formatTime(row.timestamp) }}
              </template>
            </el-table-column>
            <el-table-column prop="command" label="Command" show-overflow-tooltip />
            <el-table-column prop="status" label="Status" width="100">
              <template #default="{ row }">
                <el-tag :type="getStatusType(row.status)">{{ row.status }}</el-tag>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
      
      <el-col :span="12">
        <el-card>
          <template #header>
            <div class="card-header">
              <span>Recent Access Logs</span>
              <el-button type="primary" size="small" @click="$router.push('/access-logs')">
                View All
              </el-button>
            </div>
          </template>
          <el-table :data="recentAccess" style="width: 100%" v-loading="loadingAccess">
            <el-table-column prop="timestamp" label="Time" width="120">
              <template #default="{ row }">
                {{ formatTime(row.timestamp) }}
              </template>
            </el-table-column>
            <el-table-column prop="username" label="User" />
            <el-table-column prop="action" label="Action" />
            <el-table-column prop="client_name" label="Client" show-overflow-tooltip />
          </el-table>
        </el-card>
      </el-col>
    </el-row>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { Connection, Document, User, Monitor } from '@element-plus/icons-vue'
import api from '../services/api'

const stats = ref({
  activeConnections: 0,
  totalCommands: 0,
  activeUsers: 0,
  activeAgents: 0
})

const recentCommands = ref([])
const recentAccess = ref([])
const loadingCommands = ref(false)
const loadingAccess = ref(false)

const formatTime = (timestamp) => {
  return new Date(timestamp).toLocaleTimeString()
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

const loadStats = async () => {
  try {
    const response = await api.getStats()
    stats.value = response.data
  } catch (error) {
    console.error('Failed to load stats:', error)
  }
}

const loadRecentCommands = async () => {
  loadingCommands.value = true
  try {
    const response = await api.getCommandLogs({ limit: 5 })
    recentCommands.value = response.data.logs || []
  } catch (error) {
    console.error('Failed to load recent commands:', error)
  } finally {
    loadingCommands.value = false
  }
}

const loadRecentAccess = async () => {
  loadingAccess.value = true
  try {
    const response = await api.getAccessLogs({ limit: 5 })
    recentAccess.value = response.data.logs || []
  } catch (error) {
    console.error('Failed to load recent access:', error)
  } finally {
    loadingAccess.value = false
  }
}

onMounted(() => {
  loadStats()
  loadRecentCommands()
  loadRecentAccess()
})
</script>

<style scoped>
.dashboard {
  padding: 20px;
}

.stat-card {
  height: 120px;
}

.stat-content {
  display: flex;
  align-items: center;
  height: 100%;
}

.stat-icon {
  font-size: 40px;
  margin-right: 15px;
}

.stat-icon.primary {
  color: #409EFF;
}

.stat-icon.success {
  color: #67C23A;
}

.stat-icon.warning {
  color: #E6A23C;
}

.stat-icon.danger {
  color: #F56C6C;
}

.stat-details {
  flex: 1;
}

.stat-number {
  font-size: 28px;
  font-weight: bold;
  line-height: 1;
}

.stat-label {
  color: #909399;
  font-size: 14px;
  margin-top: 5px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.mb-4 {
  margin-bottom: 20px;
}
</style>
