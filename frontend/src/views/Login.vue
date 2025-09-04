<template>
  <div class="login-container">
    <div class="login-card">
      <div class="login-header">
        <div class="logo">
          <el-icon class="logo-icon"><Monitor /></el-icon>
          <h1>GoTeleport</h1>
        </div>
        <p class="subtitle">Remote Server Management</p>
      </div>

      <el-form
        ref="loginFormRef"
        :model="loginForm"
        :rules="loginRules"
        class="login-form"
        @submit.prevent="handleLogin"
      >
        <el-form-item prop="username">
          <el-input
            v-model="loginForm.username"
            placeholder="Username"
            size="large"
            :prefix-icon="User"
            clearable
          />
        </el-form-item>

        <el-form-item prop="password">
          <el-input
            v-model="loginForm.password"
            type="password"
            placeholder="Password"
            size="large"
            :prefix-icon="Lock"
            show-password
            @keyup.enter="handleLogin"
          />
        </el-form-item>

        <el-form-item>
          <el-button
            type="primary"
            size="large"
            class="login-button"
            :loading="loading"
            @click="handleLogin"
          >
            <span v-if="!loading">Login</span>
            <span v-else>Signing in...</span>
          </el-button>
        </el-form-item>
      </el-form>

      <div class="login-footer">
        <div class="demo-accounts">
          <h3>Demo Accounts:</h3>
          <div class="demo-account">
            <strong>Admin:</strong> admin / admin123
          </div>
          <div class="demo-account">
            <strong>User:</strong> user / user123
          </div>
        </div>
      </div>
    </div>

    <div class="background-pattern"></div>
  </div>
</template>

<script setup>
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Monitor, User, Lock } from '@element-plus/icons-vue'
import api from '../services/api'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const authStore = useAuthStore()

const loginFormRef = ref()
const loading = ref(false)

const loginForm = reactive({
  username: '',
  password: ''
})

const loginRules = {
  username: [
    { required: true, message: 'Please enter username', trigger: 'blur' },
    { min: 2, max: 50, message: 'Username length should be 2 to 50 characters', trigger: 'blur' }
  ],
  password: [
    { required: true, message: 'Please enter password', trigger: 'blur' },
    { min: 3, max: 100, message: 'Password length should be 3 to 100 characters', trigger: 'blur' }
  ]
}

const handleLogin = async () => {
  if (!loginFormRef.value) return

  try {
    await loginFormRef.value.validate()
    
    loading.value = true
    
    const response = await api.login({
      username: loginForm.username,
      password: loginForm.password
    })

    // Store authentication data
    const { token, user } = response.data
    authStore.setAuth(token, user)

    ElMessage.success(`Welcome back, ${user.username}!`)
    
    // Redirect to dashboard
    router.push('/dashboard')
    
  } catch (error) {
    console.error('Login error:', error)
    
    if (error.response?.status === 401) {
      ElMessage.error('Invalid username or password')
    } else if (error.response?.data?.message) {
      ElMessage.error(error.response.data.message)
    } else {
      ElMessage.error('Login failed. Please try again.')
    }
  } finally {
    loading.value = false
  }
}

// Quick login for demo
const quickLogin = (username, password) => {
  loginForm.username = username
  loginForm.password = password
  handleLogin()
}
</script>

<style scoped>
.login-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  position: relative;
  overflow: hidden;
}

.background-pattern {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background-image: 
    radial-gradient(circle at 25% 25%, rgba(255, 255, 255, 0.1) 0%, transparent 50%),
    radial-gradient(circle at 75% 75%, rgba(255, 255, 255, 0.1) 0%, transparent 50%);
  pointer-events: none;
}

.login-card {
  background: white;
  border-radius: 16px;
  padding: 40px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.1);
  width: 100%;
  max-width: 400px;
  position: relative;
  z-index: 1;
}

.login-header {
  text-align: center;
  margin-bottom: 30px;
}

.logo {
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 10px;
}

.logo-icon {
  font-size: 32px;
  color: #409EFF;
  margin-right: 10px;
}

.logo h1 {
  margin: 0;
  color: #303133;
  font-size: 28px;
  font-weight: 600;
}

.subtitle {
  color: #909399;
  margin: 0;
  font-size: 14px;
}

.login-form {
  margin-bottom: 20px;
}

.login-form :deep(.el-form-item) {
  margin-bottom: 20px;
}

.login-button {
  width: 100%;
  height: 44px;
  font-size: 16px;
  font-weight: 500;
}

.login-footer {
  border-top: 1px solid #EBEEF5;
  padding-top: 20px;
}

.demo-accounts {
  text-align: center;
}

.demo-accounts h3 {
  margin: 0 0 15px 0;
  color: #606266;
  font-size: 14px;
  font-weight: 500;
}

.demo-account {
  margin: 8px 0;
  padding: 8px 12px;
  background: #f8f9fa;
  border-radius: 6px;
  font-size: 13px;
  color: #606266;
  cursor: pointer;
  transition: all 0.3s;
}

.demo-account:hover {
  background: #e3f2fd;
  color: #409EFF;
}

.demo-account strong {
  color: #303133;
}

@media (max-width: 480px) {
  .login-card {
    margin: 20px;
    padding: 30px 20px;
  }
  
  .logo h1 {
    font-size: 24px;
  }
}
</style>
