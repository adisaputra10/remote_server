<template>
  <div class="settings-container">
    <div class="settings-header">
      <h2 class="settings-title">Server Settings</h2>
      <button class="btn btn-primary" @click="saveSettings" :disabled="saving">
        <i class="fas fa-save"></i>
        {{ saving ? 'Saving...' : 'Save Changes' }}
      </button>
    </div>
    
    <div v-if="loading" class="loading-state">
      <i class="fas fa-spinner fa-spin"></i>
      Loading settings...
    </div>
    
    <div v-else-if="error" class="error-state">
      <i class="fas fa-exclamation-triangle"></i>
      <p>{{ error }}</p>
      <button class="btn btn-secondary" @click="loadSettings">
        <i class="fas fa-retry"></i>
        Retry
      </button>
    </div>
    
    <div v-else class="settings-content">
      <!-- Server Configuration Table -->
      <div class="settings-section">
        <h3><i class="fas fa-server"></i> Server Configuration</h3>
        <div class="table-wrapper">
          <table class="settings-table">
            <thead>
              <tr>
                <th style="width: 200px;">Setting</th>
                <th style="width: 300px;">Value</th>
                <th>Description</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td class="setting-name">
                  <strong>Server IP</strong>
                </td>
                <td class="setting-value">
                  <input 
                    type="text" 
                    v-model="localSettings.server_ip" 
                    class="form-input"
                    placeholder="192.168.1.115"
                    @input="markAsChanged">
                </td>
                <td class="setting-description">
                  IP address of the relay server. Used for agent connections.
                </td>
              </tr>
              <tr>
                <td class="setting-name">
                  <strong>Server Port</strong>
                </td>
                <td class="setting-value">
                  <input 
                    type="number" 
                    v-model="localSettings.server_port" 
                    class="form-input"
                    placeholder="8080"
                    min="1"
                    max="65535"
                    @input="markAsChanged">
                </td>
                <td class="setting-description">
                  Port number for the relay server. Default is 8080.
                </td>
              </tr>
              <tr>
                <td class="setting-name">
                  <strong>WebSocket URL</strong>
                </td>
                <td class="setting-value">
                  <code class="websocket-url">ws://{{ localSettings.server_ip }}:{{ localSettings.server_port }}/ws/agent</code>
                </td>
                <td class="setting-description">
                  Generated WebSocket URL for agent connections. Auto-updated based on IP/Port.
                </td>
              </tr>
              <tr>
                <td class="setting-name">
                  <strong>Agent Command</strong>
                </td>
                <td class="setting-value" colspan="2">
                  <div class="command-example">
                    <code>bin/agent -a &lt;agent-id&gt; -r ws://{{ localSettings.server_ip }}:{{ localSettings.server_port }}/ws/agent</code>
                    <button class="copy-btn" @click="copyCommand" title="Copy command">
                      <i class="fas fa-copy"></i>
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
    
    <!-- Success/Error Messages -->
    <div v-if="successMessage" class="alert alert-success">
      <i class="fas fa-check-circle"></i>
      {{ successMessage }}
    </div>
    
    <div v-if="errorMessage" class="alert alert-error">
      <i class="fas fa-exclamation-circle"></i>
      {{ errorMessage }}
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, computed } from 'vue'
import apiService from '../config/api.js'

// Reactive data
const loading = ref(true)
const saving = ref(false)
const error = ref('')
const successMessage = ref('')
const errorMessage = ref('')
const hasChanges = ref(false)

const localSettings = ref({
  server_ip: '192.168.1.115',
  server_port: 8080
})

// Computed properties
const websocketUrl = computed(() => {
  return `ws://${localSettings.value.server_ip}:${localSettings.value.server_port}/ws/agent`
})

// Methods
const loadSettings = async () => {
  try {
    loading.value = true
    error.value = ''
    
    const response = await apiService.getSettings()
    console.log('Settings loaded:', response)
    
    if (response && response.success && response.data) {
      const settingsData = response.data
      
      localSettings.value = {
        server_ip: settingsData.server_ip || '192.168.1.115',
        server_port: parseInt(settingsData.server_port) || 8080
      }
    } else {
      // Use default values if no data returned
      localSettings.value = {
        server_ip: '192.168.1.115',
        server_port: 8080
      }
    }
  } catch (err) {
    console.error('Failed to load settings:', err)
    error.value = 'Failed to load settings. Using default values.'
    // Use default values
    localSettings.value = {
      server_ip: '192.168.1.115',
      server_port: 8080
    }
  } finally {
    loading.value = false
  }
}

const saveSettings = async () => {
  try {
    saving.value = true
    errorMessage.value = ''
    
    // Validate inputs
    if (!localSettings.value.server_ip) {
      errorMessage.value = 'Server IP is required'
      return
    }
    
    if (!localSettings.value.server_port || localSettings.value.server_port < 1 || localSettings.value.server_port > 65535) {
      errorMessage.value = 'Server port must be between 1 and 65535'
      return
    }
    
    // Prepare settings for API (match backend format)
    const settingsToSave = {
      serverIP: localSettings.value.server_ip,
      serverPort: parseInt(localSettings.value.server_port)
    }
    
    await apiService.saveSettings(settingsToSave)
    
    successMessage.value = 'Settings saved successfully!'
    hasChanges.value = false
    
    // Clear success message after 3 seconds
    setTimeout(() => {
      successMessage.value = ''
    }, 3000)
    
  } catch (err) {
    console.error('Failed to save settings:', err)
    errorMessage.value = err.response?.data?.error || 'Failed to save settings. Please try again.'
  } finally {
    saving.value = false
  }
}

const markAsChanged = () => {
  hasChanges.value = true
  errorMessage.value = ''
  successMessage.value = ''
}

const copyCommand = async () => {
  const command = `bin/agent -a <agent-id> -r ${websocketUrl.value}`
  try {
    await navigator.clipboard.writeText(command)
    successMessage.value = 'Agent command copied to clipboard!'
    setTimeout(() => {
      successMessage.value = ''
    }, 2000)
  } catch (err) {
    console.error('Failed to copy:', err)
    errorMessage.value = 'Failed to copy to clipboard'
  }
}

// Lifecycle
onMounted(() => {
  loadSettings()
})
</script>

<style scoped>
.settings-container {
  padding: 2rem;
  max-width: 1200px;
  margin: 0 auto;
}

.settings-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 2rem;
  padding-bottom: 1rem;
  border-bottom: 2px solid var(--border-color);
}

.settings-title {
  color: var(--text-primary);
  font-size: 1.75rem;
  font-weight: 600;
  margin: 0;
  flex-grow: 1;
}

.btn {
  padding: 0.75rem 1.5rem;
  border: none;
  border-radius: var(--radius-md);
  cursor: pointer;
  font-weight: 500;
  transition: all 0.2s ease;
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
}

.btn-primary {
  background: var(--color-primary);
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background: var(--color-primary-dark);
  transform: translateY(-1px);
}

.btn-primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.btn-secondary {
  background: var(--surface-color);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
}

.btn-secondary:hover {
  background: var(--surface-alt);
}

.loading-state,
.error-state {
  text-align: center;
  padding: 3rem 2rem;
  color: var(--text-secondary);
}

.error-state {
  color: var(--color-error);
}

.error-state p {
  margin-bottom: 1rem;
}

.settings-content {
  max-width: 100%;
}

.settings-section {
  background: var(--surface-color);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  padding: 1.5rem;
  margin-bottom: 2rem;
}

.settings-section h3 {
  color: var(--text-primary);
  font-size: 1.25rem;
  font-weight: 600;
  margin-bottom: 1.5rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.table-wrapper {
  overflow-x: auto;
}

.settings-table {
  width: 100%;
  border-collapse: collapse;
  background: var(--surface-color);
  border-radius: var(--radius-md);
  overflow: hidden;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.settings-table th {
  background: var(--surface-alt);
  color: var(--text-primary);
  font-weight: 600;
  padding: 1rem;
  text-align: left;
  border-bottom: 2px solid var(--border-color);
}

.settings-table td {
  padding: 1rem;
  border-bottom: 1px solid var(--border-color);
  vertical-align: middle;
  background: var(--surface-color);
}

.settings-table tbody tr:last-child td {
  border-bottom: none;
}

.settings-table tbody tr:hover {
  background: var(--surface-alt);
}

.setting-name {
  font-weight: 500;
  color: var(--text-primary);
}

.setting-value {
  min-width: 300px;
}

.setting-description {
  color: var(--text-secondary);
  font-size: 0.875rem;
  line-height: 1.5;
}

.form-input {
  width: 100%;
  padding: 0.75rem;
  border: 2px solid var(--border-color);
  border-radius: var(--radius-md);
  background: var(--surface-color);
  color: var(--text-primary);
  font-size: 0.875rem;
  transition: border-color 0.2s ease;
}

.form-input:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.websocket-url {
  background: var(--surface-alt);
  padding: 0.5rem 0.75rem;
  border-radius: var(--radius-sm);
  font-family: 'Fira Code', 'Consolas', 'Monaco', monospace;
  font-size: 0.875rem;
  color: var(--color-primary);
  border: 1px solid var(--border-color);
  display: inline-block;
  width: 100%;
}

.command-example {
  position: relative;
  background: var(--background-color);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  padding: 1rem;
  font-family: 'Fira Code', 'Consolas', 'Monaco', monospace;
  font-size: 0.875rem;
  color: var(--text-primary);
  overflow-x: auto;
}

.copy-btn {
  position: absolute;
  top: 0.5rem;
  right: 0.5rem;
  background: var(--surface-color);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  padding: 0.5rem;
  cursor: pointer;
  color: var(--text-secondary);
  transition: all 0.2s ease;
  font-size: 0.875rem;
}

.copy-btn:hover {
  background: var(--color-primary);
  color: white;
  border-color: var(--color-primary);
}

.alert {
  padding: 1rem;
  border-radius: var(--radius-md);
  margin-top: 1rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.alert-success {
  background: rgba(34, 197, 94, 0.1);
  color: var(--color-success);
  border: 1px solid rgba(34, 197, 94, 0.3);
}

.alert-error {
  background: rgba(239, 68, 68, 0.1);
  color: var(--color-error);
  border: 1px solid rgba(239, 68, 68, 0.3);
}

/* CSS Variables */
:root {
  --color-primary: #3b82f6;
  --color-primary-dark: #2563eb;
  --color-success: #22c55e;
  --color-error: #ef4444;
  --text-primary: #1f2937;
  --text-secondary: #6b7280;
  --background-color: #f9fafb;
  --surface-color: #ffffff;
  --surface-alt: #f3f4f6;
  --border-color: #e5e7eb;
  --radius-sm: 0.375rem;
  --radius-md: 0.5rem;
  --radius-lg: 0.75rem;
}

/* Dark mode support */
@media (prefers-color-scheme: dark) {
  :root {
    --color-primary: #60a5fa;
    --color-primary-dark: #3b82f6;
    --color-success: #4ade80;
    --color-error: #f87171;
    --text-primary: #f9fafb;
    --text-secondary: #d1d5db;
    --background-color: #111827;
    --surface-color: #1f2937;
    --surface-alt: #374151;
    --border-color: #4b5563;
  }

  .alert-success {
    background: rgba(74, 222, 128, 0.1);
    color: var(--color-success);
    border: 1px solid rgba(74, 222, 128, 0.3);
  }

  .alert-error {
    background: rgba(248, 113, 113, 0.1);
    color: var(--color-error);
    border: 1px solid rgba(248, 113, 113, 0.3);
  }

  .form-input {
    background: var(--surface-color) !important;
    color: var(--text-primary) !important;
    border-color: var(--border-color) !important;
  }

  .form-input::placeholder {
    color: var(--text-secondary) !important;
  }

  .settings-table th {
    background: var(--surface-alt) !important;
    color: var(--text-primary) !important;
  }

  .settings-table td {
    background: var(--surface-color) !important;
    color: var(--text-primary) !important;
  }
}

/* Force dark mode for systems with dark theme */
[data-theme="dark"] {
  --color-primary: #60a5fa;
  --color-primary-dark: #3b82f6;
  --color-success: #4ade80;
  --color-error: #f87171;
  --text-primary: #f9fafb;
  --text-secondary: #d1d5db;
  --background-color: #111827;
  --surface-color: #1f2937;
  --surface-alt: #374151;
  --border-color: #4b5563;
}
</style>