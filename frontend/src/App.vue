<template>
  <div v-if="showLayout" class="app-container">
    <el-header class="app-header">
      <div class="header-content">
        <div class="logo">
          <el-icon><Monitor /></el-icon>
          <span>GoTeleport Dashboard</span>
        </div>
        <div class="nav-menu">
          <el-menu mode="horizontal" :default-active="$route.path" router>
            <el-menu-item index="/dashboard">
              <el-icon><Odometer /></el-icon>
              Dashboard
            </el-menu-item>
            <el-menu-item index="/command-logs">
              <el-icon><Document /></el-icon>
              Command Logs
            </el-menu-item>
            <el-menu-item index="/access-logs">
              <el-icon><List /></el-icon>
              Access Logs
            </el-menu-item>
            <el-menu-item index="/sessions">
              <el-icon><Connection /></el-icon>
              Sessions
            </el-menu-item>
            <el-menu-item index="/agents">
              <el-icon><Monitor /></el-icon>
              Agents
            </el-menu-item>
            <el-menu-item 
              v-if="authStore.user?.role === 'admin'" 
              index="/users"
            >
              <el-icon><UserFilled /></el-icon>
              User Management
            </el-menu-item>
            <el-menu-item 
              v-if="authStore.user?.role === 'admin'" 
              index="/user-assignments"
            >
              <el-icon><Link /></el-icon>
              User Assignments
            </el-menu-item>
          </el-menu>
        </div>
        <div class="header-actions">
          <el-dropdown @command="handleUserAction">
            <span class="user-dropdown">
              <el-icon><User /></el-icon>
              {{ authStore.user?.username }}
              <el-icon class="el-icon--right"><ArrowDown /></el-icon>
            </span>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="profile">
                  <el-icon><User /></el-icon>
                  Profile
                </el-dropdown-item>
                <el-dropdown-item command="logout" divided>
                  <el-icon><SwitchButton /></el-icon>
                  Logout
                </el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </div>
    </el-header>
    
    <el-main class="app-main">
      <router-view />
    </el-main>
    
    <el-footer class="app-footer">
      <div class="footer-content">
        <span>GoTeleport © 2025</span>
        <span>Remote Server Management</span>
      </div>
    </el-footer>
  </div>
  
  <router-view v-else />
</template>

<script setup>
import { computed } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Monitor, Odometer, Document, List, Connection, User, UserFilled, ArrowDown, SwitchButton, Link } from '@element-plus/icons-vue'
import { useAuthStore } from './stores/auth'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()

// Show layout for authenticated routes only
const showLayout = computed(() => {
  return authStore.isAuthenticated && route.name !== 'Login'
})

const handleUserAction = async (command) => {
  if (command === 'logout') {
    try {
      await ElMessageBox.confirm(
        'Are you sure you want to logout?',
        'Logout',
        {
          confirmButtonText: 'Yes',
          cancelButtonText: 'Cancel',
          type: 'warning',
        }
      )
      
      authStore.logout()
      ElMessage.success('Logged out successfully')
      router.push('/login')
    } catch (error) {
      // User cancelled or other error - no action needed
      console.log('Logout cancelled or failed:', error)
    }
  } else if (command === 'profile') {
    ElMessage.info('Profile feature coming soon!')
  }
}
</script>

<style scoped>
.app-container {
  height: 100vh;
}

.app-header {
  background-color: #409EFF;
  color: white;
  padding: 0;
}

.header-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  height: 100%;
  padding: 0 20px;
}

.logo {
  display: flex;
  align-items: center;
  font-size: 20px;
  font-weight: bold;
}

.logo .el-icon {
  margin-right: 10px;
  font-size: 24px;
}

.nav-menu {
  flex: 1;
  margin-left: 40px;
}

.nav-menu :deep(.el-menu) {
  background-color: transparent;
  border-bottom: none;
}

.nav-menu :deep(.el-menu-item) {
  color: white;
  border-bottom: 2px solid transparent;
}

.nav-menu :deep(.el-menu-item:hover),
.nav-menu :deep(.el-menu-item.is-active) {
  background-color: rgba(255, 255, 255, 0.1);
  color: white;
  border-bottom-color: white;
}

.header-actions {
  display: flex;
  align-items: center;
}

.user-dropdown {
  display: flex;
  align-items: center;
  padding: 8px 12px;
  cursor: pointer;
  border-radius: 4px;
  transition: background-color 0.3s;
}

.user-dropdown:hover {
  background-color: rgba(255, 255, 255, 0.1);
}

.user-dropdown .el-icon {
  margin-right: 5px;
}

.app-main {
  background-color: #f5f5f5;
  padding: 20px;
}

.app-footer {
  background-color: #303133;
  color: white;
  text-align: center;
  line-height: 40px;
}

.footer-content {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 0 20px;
}

:deep(.el-container) {
  height: 100vh;
}
</style>
