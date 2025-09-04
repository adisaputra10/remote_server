<template>
  <div class="sessions">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>Active Sessions</span>
          <el-button type="primary" @click="loadSessions">
            <el-icon><Refresh /></el-icon>
            Refresh
          </el-button>
        </div>
      </template>

      <!-- Stats -->
      <el-row :gutter="20" class="mb-4">
        <el-col :span="8">
          <div class="stat-card">
            <div class="stat-number">{{ stats.total }}</div>
            <div class="stat-label">Total Sessions</div>
          </div>
        </el-col>
        <el-col :span="8">
          <div class="stat-card active">
            <div class="stat-number">{{ stats.active }}</div>
            <div class="stat-label">Active Sessions</div>
          </div>
        </el-col>
        <el-col :span="8">
          <div class="stat-card">
            <div class="stat-number">{{ stats.clients }}</div>
            <div class="stat-label">Connected Clients</div>
          </div>
        </el-col>
      </el-row>

      <!-- Table -->
      <el-table :data="sessions" v-loading="loading" style="width: 100%">
        <el-table-column prop="session_id" label="Session ID" width="200">
          <template #default="{ row }">
            <el-text truncated>{{ row.session_id }}</el-text>
          </template>
        </el-table-column>
        <el-table-column prop="client_name" label="Client Name" width="150">
          <template #default="{ row }">
            {{ row.client_name || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="agent_name" label="Agent Name" width="150">
          <template #default="{ row }">
            {{ row.agent_name || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="username" label="Username" width="120">
          <template #default="{ row }">
            {{ row.username || '-' }}
          </template>
        </el-table-column>
        <el-table-column prop="start_time" label="Start Time" width="180">
          <template #default="{ row }">
            {{ formatTimestamp(row.start_time) }}
          </template>
        </el-table-column>
        <el-table-column prop="last_activity" label="Last Activity" width="180">
          <template #default="{ row }">
            {{ formatTimestamp(row.last_activity) }}
          </template>
        </el-table-column>
        <el-table-column prop="status" label="Status" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)">{{ row.status }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="commands_count" label="Commands" width="100" align="center">
          <template #default="{ row }">
            <el-badge :value="row.commands_count" :max="99">
              <el-icon><Document /></el-icon>
            </el-badge>
          </template>
        </el-table-column>
        <el-table-column label="Actions" width="120" fixed="right">
          <template #default="{ row }">
            <el-button 
              type="danger" 
              size="small" 
              @click="terminateSession(row.session_id)"
              :disabled="row.status !== 'active'"
            >
              Terminate
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
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { Refresh, Document } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../services/api'

const sessions = ref([])
const loading = ref(false)

const stats = ref({
  total: 0,
  active: 0,
  clients: 0
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
    'active': 'success',
    'inactive': 'info',
    'terminated': 'danger'
  }
  return statusMap[status] || 'info'
}

const loadSessions = async () => {
  loading.value = true
  try {
    const response = await api.getSessions()
    sessions.value = response.data.sessions || []
    
    // Update stats
    stats.value.total = sessions.value.length
    stats.value.active = sessions.value.filter(s => s.status === 'active').length
    stats.value.clients = new Set(sessions.value.map(s => s.client_name)).size
    
    pagination.value.total = sessions.value.length
  } catch (error) {
    console.error('Failed to load sessions:', error)
    ElMessage.error('Failed to load sessions')
  } finally {
    loading.value = false
  }
}

const terminateSession = async (sessionId) => {
  try {
    await ElMessageBox.confirm(
      'This will terminate the session. Continue?',
      'Terminate Session',
      {
        confirmButtonText: 'OK',
        cancelButtonText: 'Cancel',
        type: 'warning',
      }
    )
    
    // Call API to terminate session
    await api.terminateSession(sessionId)
    ElMessage.success('Session terminated successfully')
    loadSessions()
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Failed to terminate session:', error)
      ElMessage.error('Failed to terminate session')
    }
  }
}

const handleSizeChange = (size) => {
  pagination.value.pageSize = size
  pagination.value.currentPage = 1
}

const handleCurrentChange = (page) => {
  pagination.value.currentPage = page
}

onMounted(() => {
  loadSessions()
})
</script>

<style scoped>
.sessions {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.stat-card {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: white;
  padding: 20px;
  border-radius: 8px;
  text-align: center;
}

.stat-card.active {
  background: linear-gradient(135deg, #84fab0 0%, #8fd3f4 100%);
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
