<template>
  <div class="user-management">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>User Management</span>
          <el-button type="primary" @click="showAddUserDialog">
            <el-icon><Plus /></el-icon>
            Add User
          </el-button>
        </div>
      </template>

      <!-- Users Table -->
      <el-table :data="users" v-loading="loading" style="width: 100%">
        <el-table-column prop="id" label="ID" width="60" />
        <el-table-column prop="username" label="Username" width="150" />
        <el-table-column prop="full_name" label="Full Name" width="200" />
        <el-table-column prop="email" label="Email" width="250" />
        <el-table-column prop="role" label="Role" width="100">
          <template #default="{ row }">
            <el-tag :type="row.role === 'admin' ? 'danger' : 'primary'">
              {{ row.role }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="Status" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 'active' ? 'success' : 'info'">
              {{ row.status }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="created_at" label="Created" width="180">
          <template #default="{ row }">
            {{ formatDateTime(row.created_at) }}
          </template>
        </el-table-column>
        <el-table-column label="Actions" width="200" fixed="right">
          <template #default="{ row }">
            <el-button 
              type="primary" 
              size="small" 
              @click="showEditUserDialog(row)"
            >
              <el-icon><Edit /></el-icon>
              Edit
            </el-button>
            <el-button 
              type="danger" 
              size="small" 
              @click="confirmDeleteUser(row)"
              :disabled="row.role === 'admin' && adminCount <= 1"
            >
              <el-icon><Delete /></el-icon>
              Delete
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- Add/Edit User Dialog -->
    <el-dialog 
      v-model="userDialogVisible" 
      :title="isEditMode ? 'Edit User' : 'Add User'"
      width="500px"
    >
      <el-form :model="userForm" :rules="userRules" ref="userFormRef" label-width="120px">
        <el-form-item label="Username" prop="username">
          <el-input 
            v-model="userForm.username" 
            :disabled="isEditMode"
            placeholder="Enter username"
          />
        </el-form-item>
        
        <el-form-item label="Full Name" prop="full_name">
          <el-input v-model="userForm.full_name" placeholder="Enter full name" />
        </el-form-item>
        
        <el-form-item label="Email" prop="email">
          <el-input v-model="userForm.email" placeholder="Enter email" />
        </el-form-item>
        
        <el-form-item label="Role" prop="role">
          <el-select v-model="userForm.role" placeholder="Select role" style="width: 100%">
            <el-option label="User" value="user" />
            <el-option label="Admin" value="admin" />
          </el-select>
        </el-form-item>
        
        <el-form-item v-if="isEditMode" label="Status" prop="status">
          <el-select v-model="userForm.status" placeholder="Select status" style="width: 100%">
            <el-option label="Active" value="active" />
            <el-option label="Inactive" value="inactive" />
          </el-select>
        </el-form-item>
        
        <el-form-item 
          :label="isEditMode ? 'New Password' : 'Password'" 
          prop="password"
          :required="!isEditMode"
        >
          <el-input 
            v-model="userForm.password" 
            type="password" 
            :placeholder="isEditMode ? 'Leave empty to keep current password' : 'Enter password'"
            show-password
          />
        </el-form-item>
        
        <el-form-item v-if="!isEditMode || userForm.password" label="Confirm Password" prop="confirmPassword">
          <el-input 
            v-model="userForm.confirmPassword" 
            type="password" 
            placeholder="Confirm password"
            show-password
          />
        </el-form-item>
      </el-form>
      
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="userDialogVisible = false">Cancel</el-button>
          <el-button type="primary" @click="saveUser" :loading="saving">
            {{ isEditMode ? 'Update' : 'Create' }}
          </el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import { Plus, Edit, Delete } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import api from '../services/api'

const users = ref([])
const loading = ref(false)
const userDialogVisible = ref(false)
const isEditMode = ref(false)
const saving = ref(false)
const userFormRef = ref()

const userForm = ref({
  id: null,
  username: '',
  full_name: '',
  email: '',
  role: 'user',
  status: 'active',
  password: '',
  confirmPassword: ''
})

const adminCount = computed(() => {
  return users.value.filter(user => user.role === 'admin' && user.status === 'active').length
})

const userRules = {
  username: [
    { required: true, message: 'Username is required', trigger: 'blur' },
    { min: 3, max: 50, message: 'Username must be 3-50 characters', trigger: 'blur' }
  ],
  full_name: [
    { required: true, message: 'Full name is required', trigger: 'blur' }
  ],
  email: [
    { required: true, message: 'Email is required', trigger: 'blur' },
    { type: 'email', message: 'Please enter a valid email', trigger: 'blur' }
  ],
  role: [
    { required: true, message: 'Role is required', trigger: 'change' }
  ],
  password: [
    { 
      validator: (rule, value, callback) => {
        if (!isEditMode.value && !value) {
          callback(new Error('Password is required'))
        } else if (value && value.length < 6) {
          callback(new Error('Password must be at least 6 characters'))
        } else {
          callback()
        }
      }, 
      trigger: 'blur' 
    }
  ],
  confirmPassword: [
    { 
      validator: (rule, value, callback) => {
        if (userForm.value.password && value !== userForm.value.password) {
          callback(new Error('Passwords do not match'))
        } else {
          callback()
        }
      }, 
      trigger: 'blur' 
    }
  ]
}

const formatDateTime = (dateTime) => {
  return new Date(dateTime).toLocaleString()
}

const loadUsers = async () => {
  loading.value = true
  try {
    const response = await api.getUsers()
    users.value = response.data.users || []
  } catch (error) {
    console.error('Failed to load users:', error)
    ElMessage.error('Failed to load users')
  } finally {
    loading.value = false
  }
}

const showAddUserDialog = () => {
  isEditMode.value = false
  userForm.value = {
    id: null,
    username: '',
    full_name: '',
    email: '',
    role: 'user',
    status: 'active',
    password: '',
    confirmPassword: ''
  }
  userDialogVisible.value = true
}

const showEditUserDialog = (user) => {
  isEditMode.value = true
  userForm.value = {
    id: user.id,
    username: user.username,
    full_name: user.full_name,
    email: user.email,
    role: user.role,
    status: user.status,
    password: '',
    confirmPassword: ''
  }
  userDialogVisible.value = true
}

const saveUser = async () => {
  if (!userFormRef.value) return
  
  const valid = await userFormRef.value.validate().catch(() => false)
  if (!valid) return
  
  saving.value = true
  try {
    const userData = {
      username: userForm.value.username,
      full_name: userForm.value.full_name,
      email: userForm.value.email,
      role: userForm.value.role,
      status: userForm.value.status
    }
    
    // Only include password if it's provided
    if (userForm.value.password) {
      userData.password = userForm.value.password
    }
    
    if (isEditMode.value) {
      await api.updateUser(userForm.value.id, userData)
      ElMessage.success('User updated successfully')
    } else {
      await api.createUser(userData)
      ElMessage.success('User created successfully')
    }
    
    userDialogVisible.value = false
    loadUsers()
  } catch (error) {
    console.error('Failed to save user:', error)
    ElMessage.error(error.response?.data || 'Failed to save user')
  } finally {
    saving.value = false
  }
}

const confirmDeleteUser = async (user) => {
  try {
    await ElMessageBox.confirm(
      `This will permanently delete user "${user.username}". Continue?`,
      'Delete User',
      {
        confirmButtonText: 'Delete',
        cancelButtonText: 'Cancel',
        type: 'warning',
      }
    )
    
    await api.deleteUser(user.id)
    ElMessage.success('User deleted successfully')
    loadUsers()
  } catch (error) {
    if (error !== 'cancel') {
      console.error('Failed to delete user:', error)
      ElMessage.error(error.response?.data || 'Failed to delete user')
    }
  }
}

onMounted(() => {
  loadUsers()
})
</script>

<style scoped>
.user-management {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>
