<template>
  <div class="user-management-container">
    <div class="table-header">
      <h2 class="table-title">User Management</h2>
      <div class="table-actions">
        <button class="btn btn-success" @click="openAddUserModal">
          <i class="fas fa-plus"></i>
          Add User
        </button>
        <button class="btn btn-primary" @click="refreshData">
          <i class="fas fa-sync-alt"></i>
          Refresh
        </button>
      </div>
    </div>
    
    <div v-if="loading" class="loading-state">
      <i class="fas fa-spinner fa-spin"></i>
      Loading users...
    </div>
    
    <div v-else-if="error" class="error-state">
      <i class="fas fa-exclamation-triangle"></i>
      {{ error }}
    </div>
    
    <div v-else-if="allUsers.length === 0" class="empty-state">
      <i class="fas fa-users"></i>
      No users found
    </div>
    
    <div v-else class="table-wrapper">
      <div class="table-container">
        <table class="table">
          <thead>
            <tr>
              <th>ID</th>
              <th>USERNAME</th>
              <th>ROLE</th>
              <th>TOKEN</th>
              <th>CREATED AT</th>
              <th>ACTIONS</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="user in paginatedUsers" :key="user.id">
              <td>{{ user.id }}</td>
              <td>{{ user.username }}</td>
              <td>
                <span :class="['badge', 'badge-' + getRoleBadgeClass(user.role)]">
                  {{ (user.role || 'user').toUpperCase() }}
                </span>
              </td>
              <td>
                <div class="token-cell">
                  <code class="token-display">{{ maskToken(user.token) }}</code>
                  <button 
                    class="btn-token-copy" 
                    @click="copyToken(user.token)"
                    title="Copy Token">
                    <i class="fas fa-copy"></i>
                  </button>
                </div>
              </td>
              <td>{{ formatDateTime(user.created_at) }}</td>
              <td>
                <div class="action-buttons">
                  <button 
                    class="action-btn edit-btn" 
                    @click="openEditUserModal(user)"
                    title="Edit User">
                    <i class="fas fa-edit"></i>
                  </button>
                  <button 
                    class="action-btn role-btn" 
                    @click="openRoleModal(user)"
                    title="Change Role">
                    <i class="fas fa-user-cog"></i>
                  </button>
                  <button 
                    v-if="user.username !== 'admin'"
                    class="action-btn delete-btn" 
                    @click="showDelete(user.id, user.username)"
                    title="Delete User">
                    <i class="fas fa-trash"></i>
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      
      <Pagination
        :current-page="currentPage"
        :total-items="allUsers.length"
        :items-per-page="itemsPerPage"
        @page-changed="handlePageChange"
      />
    </div>

    <!-- Add User Modal -->
    <div v-if="showAddModal" class="modal-overlay" @click="closeAddUserModal">
      <div class="modal" @click.stop>
        <div class="modal-header">
          <h3>
            <i class="fas fa-user-plus"></i>
            Add New User
          </h3>
          <button class="btn-close" @click="closeAddUserModal">
            <i class="fas fa-times"></i>
          </button>
        </div>
        
        <div class="modal-body">
          <div class="form-group">
            <label for="username">Username *</label>
            <input
              id="username"
              v-model="newUser.username"
              type="text"
              class="form-input"
              placeholder="Enter username"
              required
            />
            <small class="form-help">Username must be unique</small>
          </div>

          <div class="form-group">
            <label for="password">Password *</label>
            <input
              id="password"
              v-model="newUser.password"
              type="password"
              class="form-input"
              placeholder="Enter password"
              required
            />
            <small class="form-help">Minimum 6 characters</small>
          </div>

          <div class="form-group">
            <label for="role">Role *</label>
            <select
              id="role"
              v-model="newUser.role"
              class="form-input"
              required
            >
              <option value="user">User</option>
              <option value="admin">Admin</option>
            </select>
            <small class="form-help">Admin has full access, User has limited access</small>
          </div>

          <div class="form-group">
            <label for="token">Access Token</label>
            <div class="token-input-group">
              <input
                id="token"
                v-model="newUser.token"
                :type="showToken ? 'text' : 'password'"
                class="form-input"
                placeholder="Auto-generated token"
                readonly
              />
              <button type="button" class="btn btn-show" @click="toggleTokenVisibility">
                <i :class="showToken ? 'fas fa-eye-slash' : 'fas fa-eye'"></i>
                {{ showToken ? 'Hide' : 'Show' }}
              </button>
              <button type="button" class="btn btn-generate" @click="generateToken">
                <i class="fas fa-refresh"></i>
                Generate
              </button>
            </div>
            <small class="form-help">Token will be auto-generated if not provided</small>
          </div>
        </div>
        
        <div class="modal-footer">
          <button type="button" class="btn btn-secondary" @click="closeAddUserModal">
            Cancel
          </button>
          <button 
            type="button" 
            class="btn btn-success" 
            @click="submitUser"
            :disabled="submitting || !newUser.username || !newUser.password">
            <i class="fas fa-user-plus" v-if="!submitting"></i>
            <i class="fas fa-spinner fa-spin" v-else></i>
            {{ submitting ? 'Adding...' : 'Add User' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Edit User Modal -->
    <div v-if="showEditModal" class="modal-overlay" @click="closeEditUserModal">
      <div class="modal" @click.stop>
        <div class="modal-header">
          <h3>
            <i class="fas fa-user-edit"></i>
            Edit User
          </h3>
          <button class="btn-close" @click="closeEditUserModal">
            <i class="fas fa-times"></i>
          </button>
        </div>
        
        <div class="modal-body">
          <div class="form-group">
            <label for="edit-username">Username *</label>
            <input
              id="edit-username"
              v-model="editUser.username"
              type="text"
              class="form-input"
              placeholder="Enter username"
              required
            />
          </div>

          <div class="form-group">
            <label for="edit-password">New Password</label>
            <input
              id="edit-password"
              v-model="editUser.password"
              type="password"
              class="form-input"
              placeholder="Leave empty to keep current password"
            />
            <small class="form-help">Leave empty to keep current password</small>
          </div>

          <div class="form-group">
            <label for="edit-role">Role *</label>
            <select
              id="edit-role"
              v-model="editUser.role"
              class="form-input"
              required
            >
              <option value="user">User</option>
              <option value="admin">Admin</option>
            </select>
          </div>
        </div>
        
        <div class="modal-footer">
          <button type="button" class="btn btn-secondary" @click="closeEditUserModal">
            Cancel
          </button>
          <button 
            type="button" 
            class="btn btn-primary" 
            @click="submitEditUser"
            :disabled="submitting || !editUser.username">
            <i class="fas fa-save" v-if="!submitting"></i>
            <i class="fas fa-spinner fa-spin" v-else></i>
            {{ submitting ? 'Saving...' : 'Save Changes' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Role Change Modal -->
    <div v-if="showRoleModal" class="modal-overlay" @click="closeRoleModal">
      <div class="modal role-modal" @click.stop>
        <div class="modal-header">
          <h3>
            <i class="fas fa-user-cog"></i>
            Change User Role
          </h3>
          <button class="btn-close" @click="closeRoleModal">
            <i class="fas fa-times"></i>
          </button>
        </div>
        
        <div class="modal-body">
          <p class="role-change-info">
            Change role for user: <strong>{{ roleUser.username }}</strong>
          </p>
          <p class="current-role">
            Current role: <span :class="['badge', 'badge-' + getRoleBadgeClass(roleUser.role)]">{{ (roleUser.role || 'user').toUpperCase() }}</span>
          </p>
          
          <div class="form-group">
            <label for="new-role">New Role *</label>
            <select
              id="new-role"
              v-model="newRole"
              class="form-input"
              required
            >
              <option value="user">User</option>
              <option value="admin">Admin</option>
            </select>
            <small class="form-help">
              <strong>Admin:</strong> Full system access<br>
              <strong>User:</strong> Limited access to assigned resources
            </small>
          </div>
        </div>
        
        <div class="modal-footer">
          <button type="button" class="btn btn-secondary" @click="closeRoleModal">
            Cancel
          </button>
          <button 
            type="button" 
            class="btn btn-warning" 
            @click="submitRoleChange"
            :disabled="submitting || newRole === roleUser.role">
            <i class="fas fa-user-cog" v-if="!submitting"></i>
            <i class="fas fa-spinner fa-spin" v-else></i>
            {{ submitting ? 'Changing...' : 'Change Role' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Delete User Confirmation Modal -->
    <div v-if="showDeleteModal" class="modal-overlay" @click="closeDeleteModal">
      <div class="modal delete-modal" @click.stop>
        <div class="modal-header">
          <h3>
            <i class="fas fa-exclamation-triangle text-warning"></i>
            Confirm Delete
          </h3>
          <button class="btn-close" @click="closeDeleteModal">
            <i class="fas fa-times"></i>
          </button>
        </div>
        
        <div class="modal-body">
          <p class="delete-warning">
            Are you sure you want to delete user <strong>{{ userToDelete.username }}</strong>?
          </p>
          <p class="delete-note">
            This action cannot be undone. The user will be permanently removed from the system.
          </p>
        </div>
        
        <div class="modal-footer">
          <button type="button" class="btn btn-secondary" @click="closeDeleteModal">
            Cancel
          </button>
          <button type="button" class="btn btn-danger" @click="deleteUser">
            <i class="fas fa-trash"></i>
            Delete User
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { ref, onMounted, computed } from 'vue'
import { apiService } from '../config/api.js'
import Pagination from './Pagination.vue'

export default {
  name: 'UserManagement',
  components: {
    Pagination
  },
  setup() {
    const allUsers = ref([])
    const loading = ref(false)
    const error = ref(null)
    const currentPage = ref(1)
    const itemsPerPage = ref(10)

    // Add User Modal Data
    const showAddModal = ref(false)
    const submitting = ref(false)
    const showToken = ref(false)
    const newUser = ref({
      username: '',
      password: '',
      role: 'user',
      token: ''
    })

    // Edit User Modal Data
    const showEditModal = ref(false)
    const editUser = ref({
      id: null,
      username: '',
      password: '',
      role: 'user'
    })

    // Role Change Modal Data
    const showRoleModal = ref(false)
    const roleUser = ref({})
    const newRole = ref('')

    // Delete Modal Data  
    const showDeleteModal = ref(false)
    const userToDelete = ref({})

    const paginatedUsers = computed(() => {
      const start = (currentPage.value - 1) * itemsPerPage.value
      const end = start + itemsPerPage.value
      return allUsers.value.slice(start, end)
    })

    const fetchUsers = async () => {
      try {
        loading.value = true
        error.value = null
        
        console.log('=== USER MANAGEMENT - FETCHING FROM API ===')
        const response = await apiService.getUsers()
        console.log('Users API response:', response.data)
        
        allUsers.value = response.data || []
        
      } catch (err) {
        console.error('Error fetching users:', err)
        error.value = 'Failed to load users data'
        allUsers.value = []
      } finally {
        loading.value = false
      }
    }

    const refreshData = () => {
      console.log('Refreshing users data...')
      fetchUsers()
    }

    const handlePageChange = (page) => {
      currentPage.value = page
    }

    const getRoleBadgeClass = (role) => {
      return role === 'admin' ? 'danger' : 'primary'
    }

    const maskToken = (token) => {
      if (!token) return ''
      return token.substring(0, 8) + '...' + token.substring(token.length - 4)
    }

    const copyToken = (token) => {
      navigator.clipboard.writeText(token).then(() => {
        console.log('Token copied to clipboard')
        // You could add a toast notification here
        alert('Token copied to clipboard!')
      }).catch(err => {
        console.error('Failed to copy token:', err)
        alert('Failed to copy token')
      })
    }

    const formatDateTime = (dateString) => {
      if (!dateString) return 'Unknown'
      const date = new Date(dateString)
      return date.toLocaleString('en-US', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit'
      })
    }

    // Add User Modal Functions
    const openAddUserModal = () => {
      console.log('Opening Add User modal')
      showAddModal.value = true
      // Reset form with auto-generated token
      newUser.value = {
        username: '',
        password: '',
        role: 'user',
        token: generateRandomToken()
      }
      showToken.value = false
    }

    const generateRandomToken = () => {
      // Generate a secure random token (32 characters)
      const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
      let result = ''
      for (let i = 0; i < 32; i++) {
        result += chars.charAt(Math.floor(Math.random() * chars.length))
      }
      return result
    }

    const generateToken = () => {
      console.log('Generating new token...')
      newUser.value.token = generateRandomToken()
    }

    const toggleTokenVisibility = () => {
      showToken.value = !showToken.value
      console.log('Token visibility toggled:', showToken.value ? 'visible' : 'hidden')
    }

    const closeAddUserModal = () => {
      console.log('Closing Add User modal')
      showAddModal.value = false
      submitting.value = false
      showToken.value = false
      // Reset form
      newUser.value = {
        username: '',
        password: '',
        role: 'user',
        token: ''
      }
    }

    const submitUser = async () => {
      console.log('=== SUBMITTING NEW USER ===')
      console.log('User data:', newUser.value)
      
      if (!newUser.value.username || !newUser.value.password) {
        alert('Please fill in all required fields')
        return
      }

      if (newUser.value.password.length < 6) {
        alert('Password must be at least 6 characters long')
        return
      }
      
      submitting.value = true
      
      try {
        console.log('Adding user via API:', newUser.value)
        
        const userData = {
          username: newUser.value.username,
          password: newUser.value.password,
          role: newUser.value.role,
          token: newUser.value.token || generateRandomToken()
        }
        
        const response = await apiService.addUser(userData)
        console.log('User added successfully:', response.data)
        
        // Show success message
        alert(`User "${newUser.value.username}" added successfully!`)
        
        // Close modal and refresh data
        closeAddUserModal()
        fetchUsers()
        
      } catch (error) {
        console.error('Error adding user:', error)
        if (error.response) {
          console.error('Response error:', error.response.data)
          alert(`Failed to add user: ${error.response.data.error || error.message}`)
        } else {
          alert('Failed to add user. Please try again.')
        }
      } finally {
        submitting.value = false
      }
    }

    // Edit User Modal Functions
    const openEditUserModal = (user) => {
      console.log('Opening Edit User modal for:', user)
      editUser.value = {
        id: user.id,
        username: user.username,
        password: '',
        role: user.role
      }
      showEditModal.value = true
    }

    const closeEditUserModal = () => {
      console.log('Closing Edit User modal')
      showEditModal.value = false
      submitting.value = false
      editUser.value = {
        id: null,
        username: '',
        password: '',
        role: 'user'
      }
    }

    const submitEditUser = async () => {
      console.log('=== SUBMITTING EDIT USER ===')
      console.log('Edit user data:', editUser.value)
      
      if (!editUser.value.username) {
        alert('Username is required')
        return
      }

      if (editUser.value.password && editUser.value.password.length < 6) {
        alert('Password must be at least 6 characters long')
        return
      }
      
      submitting.value = true
      
      try {
        const userData = {
          username: editUser.value.username,
          role: editUser.value.role
        }
        
        // Only include password if it's provided
        if (editUser.value.password) {
          userData.password = editUser.value.password
        }
        
        const response = await apiService.updateUser(editUser.value.id, userData)
        console.log('User updated successfully:', response.data)
        
        // Show success message
        alert(`User "${editUser.value.username}" updated successfully!`)
        
        // Close modal and refresh data
        closeEditUserModal()
        fetchUsers()
        
      } catch (error) {
        console.error('Error updating user:', error)
        if (error.response) {
          console.error('Response error:', error.response.data)
          alert(`Failed to update user: ${error.response.data.error || error.message}`)
        } else {
          alert('Failed to update user. Please try again.')
        }
      } finally {
        submitting.value = false
      }
    }

    // Role Change Modal Functions
    const openRoleModal = (user) => {
      console.log('Opening Role Change modal for:', user)
      roleUser.value = { ...user }
      newRole.value = user.role
      showRoleModal.value = true
    }

    const closeRoleModal = () => {
      console.log('Closing Role Change modal')
      showRoleModal.value = false
      submitting.value = false
      roleUser.value = {}
      newRole.value = ''
    }

    const submitRoleChange = async () => {
      console.log('=== SUBMITTING ROLE CHANGE ===')
      console.log('Role change data:', { userId: roleUser.value.id, newRole: newRole.value })
      
      if (newRole.value === roleUser.value.role) {
        alert('No changes detected')
        return
      }
      
      submitting.value = true
      
      try {
        const response = await apiService.updateUserRole(roleUser.value.id, newRole.value)
        console.log('User role updated successfully:', response.data)
        
        // Show success message
        alert(`User "${roleUser.value.username}" role changed to ${newRole.value.toUpperCase()} successfully!`)
        
        // Close modal and refresh data
        closeRoleModal()
        fetchUsers()
        
      } catch (error) {
        console.error('Error updating user role:', error)
        if (error.response) {
          console.error('Response error:', error.response.data)
          alert(`Failed to update user role: ${error.response.data.error || error.message}`)
        } else {
          alert('Failed to update user role. Please try again.')
        }
      } finally {
        submitting.value = false
      }
    }

    // Delete Functions
    const showDelete = (userId, username) => {
      console.log(`Showing delete confirmation for user: ${username} (${userId})`)
      userToDelete.value = { id: userId, username }
      showDeleteModal.value = true
    }

    const closeDeleteModal = () => {
      showDeleteModal.value = false
      userToDelete.value = {}
    }

    const deleteUser = async () => {
      if (!userToDelete.value.id) return
      
      try {
        console.log(`Deleting user: ${userToDelete.value.username} (${userToDelete.value.id})`)
        await apiService.deleteUser(userToDelete.value.id)
        
        // Refresh the table after deletion
        await fetchUsers()
        closeDeleteModal()
        
        console.log(`User ${userToDelete.value.username} deleted successfully`)
        alert(`User ${userToDelete.value.username} deleted successfully`)
      } catch (error) {
        console.error('Error deleting user:', error)
        alert('Failed to delete user: ' + (error.response?.data?.error || error.message))
      }
    }

    onMounted(() => {
      fetchUsers()
    })

    return {
      allUsers,
      paginatedUsers,
      loading,
      error,
      currentPage,
      itemsPerPage,
      showAddModal,
      showEditModal,
      showRoleModal,
      showDeleteModal,
      submitting,
      showToken,
      newUser,
      editUser,
      roleUser,
      newRole,
      userToDelete,
      refreshData,
      handlePageChange,
      getRoleBadgeClass,
      maskToken,
      copyToken,
      formatDateTime,
      openAddUserModal,
      closeAddUserModal,
      submitUser,
      generateToken,
      toggleTokenVisibility,
      openEditUserModal,
      closeEditUserModal,
      submitEditUser,
      openRoleModal,
      closeRoleModal,
      submitRoleChange,
      showDelete,
      closeDeleteModal,
      deleteUser
    }
  }
}
</script>

<style scoped>
/* CSS Custom Properties for consistency */
:root {
  --text-primary: #1f2937;
  --text-secondary: #6b7280;
  --surface-color: #ffffff;
  --surface-alt: #f9fafb;
  --background-color: #ffffff;
  --border-color: #e5e7eb;
  --primary-color: #3b82f6;
  --primary-color-dark: #2563eb;
  --color-primary: #3b82f6;
  --color-primary-dark: #2563eb;
  --color-success: #10b981;
  --color-success-dark: #059669;
  --color-warning: #f59e0b;
  --color-warning-dark: #d97706;
  --color-danger: #ef4444;
  --color-danger-dark: #dc2626;
  --radius-sm: 0.25rem;
  --radius-md: 0.375rem;
  --radius-lg: 0.5rem;
}

.user-management-container {
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

.table-actions {
  display: flex;
  gap: 0.75rem;
  align-items: center;
}

/* Button base styles */
.btn {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1.5rem;
  border: none;
  border-radius: var(--radius-md);
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  text-decoration: none;
  line-height: 1;
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-primary {
  background: var(--color-primary);
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background: var(--color-primary-dark);
  transform: translateY(-1px);
}

.btn-success {
  background: var(--color-success);
  color: white;
}

.btn-success:hover:not(:disabled) {
  background: var(--color-success-dark);
  transform: translateY(-1px);
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

/* Table styles */
.table {
  width: 100%;
  border-collapse: collapse;
  background: var(--surface-color);
  border-radius: var(--radius-md);
  overflow: hidden;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.table th,
.table td {
  padding: 1rem;
  text-align: left;
  border-bottom: 1px solid var(--border-color);
}

.table th {
  background: var(--surface-alt);
  font-weight: 600;
  color: var(--text-primary);
  font-size: 0.875rem;
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.table td {
  color: var(--text-primary);
  font-size: 0.875rem;
}

.table tbody tr:hover {
  background: var(--surface-alt);
}

/* Badge styles for roles */
.badge {
  padding: 0.25rem 0.5rem;
  border-radius: var(--radius-sm);
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
}

.badge-primary {
  background: var(--color-primary);
  color: white;
}

.badge-danger {
  background: var(--color-danger);
  color: white;
}

/* Token cell styling */
.token-cell {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.token-display {
  background: var(--surface-alt);
  padding: 0.25rem 0.5rem;
  border-radius: var(--radius-sm);
  font-family: monospace;
  font-size: 0.75rem;
  color: var(--text-secondary);
}

.btn-token-copy {
  background: none;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  padding: 0.25rem 0.5rem;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s ease;
  font-size: 0.75rem;
}

.btn-token-copy:hover {
  background: var(--color-primary);
  color: white;
  border-color: var(--color-primary);
}

/* Action buttons */
.action-buttons {
  display: flex;
  gap: 0.25rem;
}

.action-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  background: var(--surface-color);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.3s ease;
}

.action-btn:hover {
  transform: translateY(-1px);
}

.action-btn.edit-btn:hover {
  background: var(--color-primary);
  color: white;
  border-color: var(--color-primary);
}

.action-btn.role-btn:hover {
  background: var(--color-warning);
  color: white;
  border-color: var(--color-warning);
}

.action-btn.delete-btn:hover {
  background: var(--color-danger);
  color: white;
  border-color: var(--color-danger);
}

/* Modal styles */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal {
  background: var(--surface-color);
  border-radius: var(--radius-lg);
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.15);
  width: 90%;
  max-width: 500px;
  max-height: 90vh;
  overflow: hidden;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.5rem;
  border-bottom: 1px solid var(--border-color);
}

.modal-header h3 {
  margin: 0;
  color: var(--text-primary);
  font-size: 1.25rem;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.btn-close {
  background: none;
  border: none;
  font-size: 1.25rem;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 0.25rem;
  border-radius: var(--radius-sm);
  transition: all 0.2s ease;
}

.btn-close:hover {
  background: var(--surface-alt);
  color: var(--text-primary);
}

.modal-body {
  padding: 1.5rem;
  max-height: 60vh;
  overflow-y: auto;
}

.form-group {
  margin-bottom: 1.5rem;
}

.form-group label {
  display: block;
  margin-bottom: 0.5rem;
  color: var(--text-primary);
  font-weight: 500;
  font-size: 0.875rem;
}

.form-input, .form-input select {
  width: 100%;
  padding: 0.75rem;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  background: var(--background-color);
  color: var(--text-primary);
  font-size: 0.875rem;
  transition: all 0.2s ease;
  box-sizing: border-box;
}

.form-input:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.form-input::placeholder {
  color: var(--text-secondary);
}

.form-help {
  display: block;
  margin-top: 0.375rem;
  font-size: 0.75rem;
  color: var(--text-secondary);
  line-height: 1.4;
}

/* Token input group */
.token-input-group {
  display: flex;
  gap: 0.5rem;
  align-items: stretch;
}

.token-input-group .form-input {
  flex: 1;
}

.token-input-group .btn {
  padding: 0.75rem 1rem;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  background: var(--surface-alt);
  color: var(--text-secondary);
  font-size: 0.75rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  gap: 0.375rem;
  white-space: nowrap;
}

.token-input-group .btn:hover {
  background: var(--primary-color);
  color: white;
  border-color: var(--primary-color);
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  padding: 1.5rem;
  border-top: 1px solid var(--border-color);
}

.modal-footer .btn {
  padding: 0.75rem 1.5rem;
  border: none;
  border-radius: var(--radius-md);
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.modal-footer .btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.modal-footer .btn-secondary {
  background: var(--surface-alt);
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
}

.modal-footer .btn-secondary:hover:not(:disabled) {
  background: var(--border-color);
  color: var(--text-primary);
}

.modal-footer .btn-success {
  background: var(--color-success);
  color: white;
}

.modal-footer .btn-success:hover:not(:disabled) {
  background: var(--color-success-dark);
  transform: translateY(-1px);
}

.modal-footer .btn-primary {
  background: var(--color-primary);
  color: white;
}

.modal-footer .btn-primary:hover:not(:disabled) {
  background: var(--color-primary-dark);
  transform: translateY(-1px);
}

.modal-footer .btn-warning {
  background: var(--color-warning);
  color: white;
}

.modal-footer .btn-warning:hover:not(:disabled) {
  background: var(--color-warning-dark);
  transform: translateY(-1px);
}

.modal-footer .btn-danger {
  background: var(--color-danger);
  color: white;
}

.modal-footer .btn-danger:hover:not(:disabled) {
  background: var(--color-danger-dark);
  transform: translateY(-1px);
}

/* Role modal specific styles */
.role-modal {
  max-width: 450px;
}

.role-change-info {
  margin-bottom: 1rem;
  color: var(--text-primary);
  font-size: 1rem;
}

.current-role {
  margin-bottom: 1.5rem;
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

/* Delete modal styles */
.delete-modal {
  max-width: 450px;
}

.delete-warning {
  margin-bottom: 1rem;
  color: var(--text-primary);
  font-size: 1rem;
}

.delete-note {
  margin-bottom: 0;
  color: var(--text-secondary);
  font-size: 0.875rem;
}

.text-warning {
  color: var(--color-warning);
}
</style>
