<template>
  <div class="user-assignments">
    <el-card>
      <template #header>
        <div class="card-header">
          <h3>User-Agent Assignments</h3>
          <el-button type="primary" @click="showAssignDialog = true" :icon="Plus">
            Assign User to Agent
          </el-button>
        </div>
      </template>

      <!-- Filters -->
      <div class="filters">
        <el-row :gutter="16">
          <el-col :span="8">
            <el-select
              v-model="selectedUser"
              placeholder="Filter by User"
              clearable
              @change="loadAssignments"
            >
              <el-option
                v-for="user in users"
                :key="user.id"
                :label="user.username"
                :value="user.id"
              />
            </el-select>
          </el-col>
          <el-col :span="8">
            <el-select
              v-model="selectedAgent"
              placeholder="Filter by Agent"
              clearable
              @change="loadAssignments"
            >
              <el-option
                v-for="agent in agents"
                :key="agent.id"
                :label="agent.name || agent.id"
                :value="agent.id"
              />
            </el-select>
          </el-col>
          <el-col :span="8">
            <el-button @click="loadAssignments" :icon="Refresh">Refresh</el-button>
          </el-col>
        </el-row>
      </div>

      <!-- Assignments Table -->
      <el-table
        :data="assignments"
        v-loading="loading"
        style="width: 100%"
        empty-text="No assignments found"
      >
        <el-table-column prop="username" label="User" width="200">
          <template #default="scope">
            <el-tag type="info">{{ scope.row.username }}</el-tag>
          </template>
        </el-table-column>
        
        <el-table-column prop="agent_id" label="Agent ID" width="300">
          <template #default="scope">
            <el-tag>{{ scope.row.agent_id }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="assigned_by_username" label="Assigned By" width="200">
          <template #default="scope">
            <el-tag type="success">{{ scope.row.assigned_by_username }}</el-tag>
          </template>
        </el-table-column>

        <el-table-column prop="assigned_at" label="Assigned At" width="200">
          <template #default="scope">
            {{ formatDateTime(scope.row.assigned_at) }}
          </template>
        </el-table-column>

        <el-table-column label="Actions" width="150">
          <template #default="scope">
            <el-button
              type="danger"
              size="small"
              @click="confirmUnassign(scope.row)"
              :icon="Delete"
            >
              Unassign
            </el-button>
          </template>
        </el-table-column>
      </el-table>

      <!-- Pagination -->
      <div class="pagination" v-if="assignments.length > 0">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="total"
          layout="total, sizes, prev, pager, next, jumper"
          @size-change="loadAssignments"
          @current-change="loadAssignments"
        />
      </div>
    </el-card>

    <!-- Assignment Dialog -->
    <el-dialog
      v-model="showAssignDialog"
      title="Assign User to Agent"
      width="500px"
    >
      <el-form
        ref="assignFormRef"
        :model="assignForm"
        :rules="assignRules"
        label-width="120px"
      >
        <el-form-item label="User" prop="user_id">
          <el-select
            v-model="assignForm.user_id"
            placeholder="Select User"
            style="width: 100%"
          >
            <el-option
              v-for="user in users.filter(u => u.role !== 'admin')"
              :key="user.id"
              :label="`${user.username} (${user.role})`"
              :value="user.id"
            />
          </el-select>
        </el-form-item>

        <el-form-item label="Agent" prop="agent_id">
          <el-select
            v-model="assignForm.agent_id"
            placeholder="Select Agent"
            style="width: 100%"
          >
            <el-option
              v-for="agent in agents"
              :key="agent.id"
              :label="`${agent.name || agent.id} (${agent.status})`"
              :value="agent.id"
            />
          </el-select>
        </el-form-item>
      </el-form>

      <template #footer>
        <span class="dialog-footer">
          <el-button @click="showAssignDialog = false">Cancel</el-button>
          <el-button
            type="primary"
            @click="handleAssign"
            :loading="assignLoading"
          >
            Assign
          </el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Refresh, Delete } from '@element-plus/icons-vue'
import api from '../services/api'

// Reactive data
const loading = ref(false)
const assignments = ref([])
const users = ref([])
const agents = ref([])
const total = ref(0)
const currentPage = ref(1)
const pageSize = ref(20)

// Filters
const selectedUser = ref('')
const selectedAgent = ref('')

// Assignment dialog
const showAssignDialog = ref(false)
const assignLoading = ref(false)
const assignFormRef = ref()
const assignForm = reactive({
  user_id: '',
  agent_id: ''
})

const assignRules = {
  user_id: [
    { required: true, message: 'Please select a user', trigger: 'change' }
  ],
  agent_id: [
    { required: true, message: 'Please select an agent', trigger: 'change' }
  ]
}

// Methods
const loadAssignments = async () => {
  try {
    loading.value = true
    const params = {}
    
    if (selectedUser.value) {
      params.user_id = selectedUser.value
    }
    
    if (selectedAgent.value) {
      params.agent_id = selectedAgent.value
    }

    const response = await api.getUserAssignments(params)
    assignments.value = response.data.assignments || []
    total.value = response.data.total || 0
  } catch (error) {
    console.error('Error loading assignments:', error)
    ElMessage.error('Failed to load assignments')
    assignments.value = []
  } finally {
    loading.value = false
  }
}

const loadUsers = async () => {
  try {
    const response = await api.getUsers()
    users.value = response.data.users || []
  } catch (error) {
    console.error('Error loading users:', error)
    ElMessage.error('Failed to load users')
  }
}

const loadAgents = async () => {
  try {
    const response = await api.getAgents()
    agents.value = response.data.agents || []
  } catch (error) {
    console.error('Error loading agents:', error)
    ElMessage.error('Failed to load agents')
  }
}

const handleAssign = async () => {
  try {
    await assignFormRef.value.validate()
    
    assignLoading.value = true
    
    await api.createUserAssignment({
      user_id: assignForm.user_id,
      agent_id: assignForm.agent_id
    })
    
    ElMessage.success('User assigned to agent successfully')
    showAssignDialog.value = false
    
    // Reset form
    assignForm.user_id = ''
    assignForm.agent_id = ''
    assignFormRef.value.resetFields()
    
    // Reload assignments
    await loadAssignments()
  } catch (error) {
    console.error('Error creating assignment:', error)
    if (error.response?.status === 409) {
      ElMessage.error('Assignment already exists')
    } else {
      ElMessage.error('Failed to create assignment')
    }
  } finally {
    assignLoading.value = false
  }
}

const confirmUnassign = async (assignment) => {
  try {
    await ElMessageBox.confirm(
      `Are you sure you want to unassign user "${assignment.username}" from agent "${assignment.agent_id}"?`,
      'Confirm Unassignment',
      {
        confirmButtonText: 'Yes, Unassign',
        cancelButtonText: 'Cancel',
        type: 'warning'
      }
    )
    
    await api.deleteUserAssignment(assignment.user_id, assignment.agent_id)
    ElMessage.success('User unassigned successfully')
    
    // Reload assignments
    await loadAssignments()
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Error unassigning user:', error)
      ElMessage.error('Failed to unassign user')
    }
  }
}

const formatDateTime = (dateTime) => {
  if (!dateTime) return 'N/A'
  return new Date(dateTime).toLocaleString()
}

// Lifecycle
onMounted(async () => {
  await Promise.all([
    loadAssignments(),
    loadUsers(),
    loadAgents()
  ])
})
</script>

<style scoped>
.user-assignments {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-header h3 {
  margin: 0;
}

.filters {
  margin-bottom: 20px;
  padding: 16px;
  background-color: #f5f7fa;
  border-radius: 4px;
}

.pagination {
  margin-top: 20px;
  text-align: right;
}

.dialog-footer {
  text-align: right;
}

.el-table {
  margin-top: 16px;
}
</style>
