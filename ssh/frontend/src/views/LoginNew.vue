<template>
  <div class="login-container">
    <div class="login-box">
      <div class="logo">
        <div class="logo-icon">
          <i class="fas fa-server"></i>
        </div>
        <h1>ServerHub Pro</h1>
        <p>Professional Management Dashboard</p>
      </div>
      
      <div v-if="errorMessage" class="error-message">
        <i class="fas fa-exclamation-triangle"></i>
        <span>{{ errorMessage }}</span>
      </div>
      
      <form @submit.prevent="handleLogin">
        <div class="form-group">
          <label class="form-label" for="username">Username or Email</label>
          <div class="input-wrapper">
            <i class="fas fa-user input-icon"></i>
            <input 
              type="text" 
              id="username" 
              v-model="form.username"
              class="form-input" 
              placeholder="Enter your username" 
              required
            >
          </div>
        </div>
        
        <div class="form-group">
          <label class="form-label" for="password">Password</label>
          <div class="input-wrapper">
            <i class="fas fa-lock input-icon"></i>
            <input 
              type="password" 
              id="password" 
              v-model="form.password"
              class="form-input" 
              placeholder="Enter your password" 
              required
            >
          </div>
        </div>
        
        <div class="form-checkbox">
          <input type="checkbox" id="remember" v-model="form.remember">
          <label for="remember">Remember me for 30 days</label>
        </div>
        
        <button type="submit" class="btn btn-primary" :disabled="isLoading">
          <i class="fas fa-sign-in-alt"></i>
          {{ isLoading ? 'Signing In...' : 'Sign In' }}
        </button>
      </form>
      
      <div class="forgot-password">
        <a href="#" @click="showForgotPassword">Forgot your password?</a>
      </div>
    </div>
  </div>
</template>

<script>
import { defineComponent, ref } from 'vue'
import { useRouter } from 'vue-router'
import { apiService } from '../config/api.js'
import { setUser } from '../utils/auth.js'

export default defineComponent({
  name: 'Login',
  setup() {
    const router = useRouter()
    const isLoading = ref(false)
    const errorMessage = ref('')
    
    const form = ref({
      username: '',
      password: '',
      remember: false
    })

    const handleLogin = async () => {
      try {
        isLoading.value = true
        errorMessage.value = ''
        
        // Encode credentials for Basic Auth
        const credentials = btoa(`${form.value.username}:${form.value.password}`)
        
        const response = await apiService.login({
          username: form.value.username,
          password: form.value.password
        })
        
        // Get user role from response or default to 'user'
        let userRole = 'user'
        if (response.data && response.data.role) {
          userRole = response.data.role
        } else {
          // If no role in response, assume admin for 'admin' username, user for others
          userRole = form.value.username === 'admin' ? 'admin' : 'user'
        }
        
        // Set user in auth store
        setUser({
          username: form.value.username,
          role: userRole,
          token: credentials
        })
        
        console.log(`User ${form.value.username} logged in with role: ${userRole}`)
        
        // Redirect to dashboard
        router.push('/dashboard')
        
      } catch (error) {
        console.error('Login error:', error)
        if (error.response?.status === 401) {
          errorMessage.value = 'Invalid username or password. Please try again.'
        } else if (error.response?.status === 403) {
          errorMessage.value = 'Access denied.'
        } else {
          errorMessage.value = 'Login failed. Please try again.'
        }
      } finally {
        isLoading.value = false
      }
    }

    const showForgotPassword = () => {
      alert('Password reset functionality would be implemented here')
    }

    // Check if already logged in
    const token = localStorage.getItem('auth_token')
    if (token) {
      router.push('/dashboard')
    }

    return {
      form,
      isLoading,
      errorMessage,
      handleLogin,
      showForgotPassword
    }
  }
})
</script>

<style scoped>
/* Login Page Styles - copied from server-dashboard.html */
.login-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--primary-gradient);
  position: relative;
  overflow: hidden;
  padding: 20px;
}

.login-container::before {
  content: '';
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: 
    radial-gradient(circle at 30% 20%, rgba(255, 255, 255, 0.1) 0%, transparent 50%),
    radial-gradient(circle at 80% 80%, rgba(255, 255, 255, 0.08) 0%, transparent 50%),
    radial-gradient(circle at 40% 70%, rgba(255, 255, 255, 0.05) 0%, transparent 50%);
}

.login-box {
  background: var(--surface-color);
  border-radius: var(--radius-xl);
  padding: 48px 40px;
  box-shadow: var(--shadow-xl);
  width: 100%;
  max-width: 480px;
  position: relative;
  z-index: 1;
  border: 1px solid rgba(255, 255, 255, 0.1);
  backdrop-filter: blur(10px);
}

.logo {
  text-align: center;
  margin-bottom: 40px;
}

.logo-icon {
  width: 64px;
  height: 64px;
  margin: 0 auto 20px;
  background: var(--primary-gradient);
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: white;
  font-size: 28px;
  box-shadow: var(--shadow-lg);
}

.logo h1 {
  font-size: 28px;
  font-weight: 800;
  color: var(--text-primary);
  margin-bottom: 8px;
  background: var(--primary-gradient);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.logo p {
  color: var(--text-secondary);
  font-size: 16px;
  font-weight: 500;
}

.error-message {
  background: linear-gradient(135deg, #fed7d7 0%, #feb2b2 100%);
  color: #742a2a;
  padding: 16px 20px;
  border-radius: var(--radius-lg);
  margin-bottom: 24px;
  display: flex;
  align-items: center;
  gap: 12px;
  border: 1px solid #feb2b2;
  font-weight: 500;
  animation: shake 0.5s ease-in-out;
}

@keyframes shake {
  0%, 100% { transform: translateX(0); }
  25% { transform: translateX(-5px); }
  75% { transform: translateX(5px); }
}

.form-group {
  margin-bottom: 24px;
}

.form-label {
  display: block;
  margin-bottom: 8px;
  font-weight: 600;
  color: var(--text-primary);
  font-size: 14px;
}

.input-wrapper {
  position: relative;
}

.input-icon {
  position: absolute;
  left: 16px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--text-light);
  font-size: 16px;
  z-index: 1;
}

.form-input {
  width: 100%;
  padding: 16px 16px 16px 48px;
  border: 2px solid var(--border-color);
  border-radius: var(--radius-lg);
  font-size: 16px;
  font-weight: 500;
  background: var(--surface-color);
  color: var(--text-primary);
  transition: all 0.3s ease;
}

.form-input:focus {
  outline: none;
  border-color: var(--primary-color);
  box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
}

.form-checkbox {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 32px;
}

.form-checkbox input[type="checkbox"] {
  width: 18px;
  height: 18px;
  border: 2px solid var(--border-color);
  border-radius: 4px;
  cursor: pointer;
}

.form-checkbox label {
  color: var(--text-secondary);
  font-weight: 500;
  cursor: pointer;
  font-size: 14px;
}

.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 16px 24px;
  border: none;
  border-radius: var(--radius-lg);
  font-size: 16px;
  font-weight: 600;
  text-decoration: none;
  cursor: pointer;
  transition: all 0.3s ease;
  width: 100%;
}

.btn-primary {
  background: var(--primary-gradient);
  color: white;
  box-shadow: var(--shadow-md);
}

.btn-primary:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: var(--shadow-lg);
}

.btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  transform: none;
}

.forgot-password {
  text-align: center;
  margin-top: 24px;
}

.forgot-password a {
  color: var(--primary-color);
  text-decoration: none;
  font-weight: 600;
  font-size: 14px;
  transition: all 0.3s ease;
}

.forgot-password a:hover {
  color: var(--primary-dark);
  text-decoration: underline;
}

/* Dark theme adjustments */
[data-theme="dark"] .login-box {
  background: var(--surface-color);
  border: 1px solid var(--border-color);
}

[data-theme="dark"] .error-message {
  background: linear-gradient(135deg, #742a2a 0%, #9b2c2c 100%);
  color: #fed7d7;
  border-color: #9b2c2c;
}
</style>