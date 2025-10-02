<template>
  <div class="ssh-terminal-container">
    <div class="terminal-header">
      <h2>SSH Web Terminal</h2>
    </div>
    
    <!-- Hide connection form for secure auto-login -->
    <div class="connection-form" v-if="false" style="display: none;">
      <div class="form-group">
        <label for="host">Host:</label>
        <input type="text" id="host" v-model="connection.host" placeholder="example.com or IP">
      </div>
      <div class="form-group">
        <label for="port">Port:</label>
        <input type="number" id="port" v-model="connection.port" value="22">
      </div>
      <div class="form-group">
        <label for="username">Username:</label>
        <input type="text" id="username" v-model="connection.username" placeholder="username">
      </div>
      <div class="form-group">
        <label for="password">Password:</label>
        <input type="password" id="password" v-model="connection.password" placeholder="password">
      </div>
      <button class="connect-btn" @click="toggleConnection" :disabled="isConnecting">
        {{ isConnected ? 'Disconnect' : 'Connect' }}
      </button>
    </div>
    
    <div class="terminal-container">
      <div 
        id="terminal" 
        class="terminal" 
        ref="terminal"
        tabindex="0"
        @click="focusTerminal"
        @keydown="handleKeyDown"
      >{{ terminalContent }}</div>
    </div>
    
    <div class="status-bar">
      <span :class="['connection-status', connectionStatusClass]">{{ connectionStatusText }}</span>
      <span v-if="connection.host && connection.username" class="connection-info">
        {{ connection.username }}@{{ connection.host }}:{{ connection.port }}
      </span>
    </div>
  </div>
</template>

<script>
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { apiService } from '../config/api.js'

export default {
  name: 'SSHWebTerminal',
  setup() {
    const route = useRoute()
    const terminal = ref(null)
    const terminalContent = ref('SSH Web Terminal\r\nPlease enter connection details and click Connect.\r\n\r\n')
    const isConnected = ref(false)
    const isConnecting = ref(false)
    const socket = ref(null)
    const agentId = ref(null)
    
    const connection = reactive({
      host: '',
      port: 22,
      username: '',
      password: ''
    })
    
    const connectionStatusText = computed(() => {
      if (isConnecting.value) return 'Connecting...'
      if (isConnected.value) return `Connected to ${connection.host}`
      return 'Disconnected'
    })
    
    const connectionStatusClass = computed(() => {
      if (isConnecting.value) return 'connecting'
      if (isConnected.value) return 'connected'
      return ''
    })
    
    const focusTerminal = () => {
      terminal.value?.focus()
    }
    
    const appendToTerminal = (text) => {
      terminalContent.value += text
      // Auto-scroll to bottom with minimal delay
      setTimeout(() => {
        if (terminal.value) {
          terminal.value.scrollTop = terminal.value.scrollHeight
        }
      }, 0)
    }
    
    const toggleConnection = () => {
      if (isConnected.value) {
        disconnect()
      } else {
        connect()
      }
    }
    
    const connect = () => {
      if (!connection.host || !connection.port || !connection.username || !connection.password) {
        appendToTerminal('Error: Please fill in all connection details.\r\n')
        return
      }
      
      isConnecting.value = true
      
      // Determine connection target based on agent mode
      let connectHost = connection.host
      let connectPort = connection.port
      let connectionText = `${connection.host}:${connection.port}`
      
      if (agentId.value) {
        // For agent mode, connection goes through agent tunnel
        // Agent should handle the SSH forwarding
        connectionText = `${connection.host}:${connection.port} via Agent ${agentId.value}`
      }
      
      appendToTerminal(`Connecting to ${connectionText}...\r\n`)
      
      // Create WebSocket connection to relay server
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      // Use the new secure SSH WebSocket endpoint
      const wsUrl = `${protocol}//${window.location.hostname}:8080/ws/ssh`
      
      try {
        socket.value = new WebSocket(wsUrl)
        
        socket.value.onopen = () => {
          // Send connection details to relay server
          const connectData = {
            type: 'connect',
            host: connectHost,
            port: parseInt(connectPort),
            username: connection.username,
            password: connection.password
          }
          
          // Add agent information if available
          if (agentId.value) {
            connectData.agentId = agentId.value
          }
          
          socket.value.send(JSON.stringify(connectData))
        }
        
        socket.value.onmessage = (event) => {
          try {
            const data = JSON.parse(event.data)
            
            if (data.type === 'connected') {
              isConnected.value = true
              isConnecting.value = false
              appendToTerminal(`Successfully connected to ${connection.host}\r\n`)
              focusTerminal()
            } else if (data.type === 'data') {
              // Direct append for INSTANT display - no additional processing
              terminalContent.value += data.data
              // Immediate scroll update
              if (terminal.value) {
                terminal.value.scrollTop = terminal.value.scrollHeight
              }
            } else if (data.type === 'error') {
              appendToTerminal(`Error: ${data.message}\r\n`)
              disconnect()
            } else if (data.type === 'disconnected') {
              appendToTerminal('Connection closed by server\r\n')
              disconnect()
            }
          } catch (e) {
            console.error('Error parsing WebSocket message:', e)
          }
        }
        
        socket.value.onclose = () => {
          if (isConnected.value) {
            appendToTerminal('Connection lost\r\n')
            disconnect()
          }
        }
        
        socket.value.onerror = (error) => {
          appendToTerminal('WebSocket connection error\r\n')
          console.error('WebSocket error:', error)
          disconnect()
        }
      } catch (error) {
        appendToTerminal('Failed to create WebSocket connection\r\n')
        console.error('WebSocket creation error:', error)
        disconnect()
      }
    }
    
    const disconnect = () => {
      if (socket.value) {
        socket.value.close()
        socket.value = null
      }
      
      isConnected.value = false
      isConnecting.value = false
    }
    
    // Ultra-optimized keyboard handling for INSTANT response
    const keyMap = {
      'Enter': '\r',
      'Backspace': '\x08',
      'Tab': '\t',
      'Escape': '\x1b',
      'ArrowUp': '\x1b[A',
      'ArrowDown': '\x1b[B',
      'ArrowRight': '\x1b[C',
      'ArrowLeft': '\x1b[D'
    }

    const handleKeyDown = (e) => {
      if (!isConnected.value || !socket.value) return
      
      e.preventDefault()
      
      let keyData = ''
      
      // Ultra-fast key mapping
      if (keyMap[e.key]) {
        keyData = keyMap[e.key]
      } else if (e.ctrlKey) {
        // Control key combinations
        const ctrlKeys = {
          'c': '\x03', 'd': '\x04', 'z': '\x1a'
        }
        keyData = ctrlKeys[e.key] || ''
      } else if (e.key.length === 1) {
        keyData = e.key
      }
      
      // Send IMMEDIATELY for fastest possible response
      if (keyData && socket.value.readyState === WebSocket.OPEN) {
        socket.value.send(JSON.stringify({ type: 'data', data: keyData }))
      }
    }
    
    onMounted(async () => {
      // Ensure terminal is focusable
      if (terminal.value) {
        terminal.value.setAttribute('tabindex', '0')
        focusTerminal()
      }
      
      // Check for secure connection data from session storage (preferred method)
      const sessionData = sessionStorage.getItem('ssh_connection_data')
      if (sessionData) {
        try {
          const connectionData = JSON.parse(sessionData)
          sessionStorage.removeItem('ssh_connection_data') // Clean up for security
          
          // Pre-fill connection details
          connection.host = connectionData.host
          connection.port = connectionData.port || 22
          connection.username = connectionData.username
          connection.password = connectionData.password || ''
          
          appendToTerminal(`🔒 Secure SSH Terminal - Auto-Login Mode\r\n`)
          appendToTerminal(`Target: ${connectionData.host}:${connectionData.port}\r\n`)
          appendToTerminal(`User: ${connectionData.username}\r\n`)
          appendToTerminal(`Establishing secure connection...\r\n\r\n`)
          
          // Auto-connect if password is provided
          if (connectionData.password) {
            setTimeout(() => {
              connect()
            }, 1000) // Small delay to show the connection info
          } else {
            appendToTerminal(`Please enter password and click Connect.\r\n\r\n`)
          }
          
          return // Exit early if session data was found
        } catch (error) {
          console.error('Error parsing session connection data:', error)
          appendToTerminal(`Error parsing connection data: ${error.message}\r\n\r\n`)
        }
      }
      
      // Fallback: Check URL parameters (less secure)
      const urlParams = new URLSearchParams(window.location.search)
      const hostParam = urlParams.get('host')
      const usernameParam = urlParams.get('username')
      const passwordParam = urlParams.get('password')
      const portParam = urlParams.get('port')
      
      if (hostParam && usernameParam) {
        connection.host = hostParam
        connection.port = portParam || 22
        connection.username = usernameParam
        connection.password = passwordParam || ''
        
        appendToTerminal(`SSH Web Terminal - URL Parameters Mode\r\n`)
        appendToTerminal(`Target: ${hostParam}:${portParam || 22}\r\n`)
        appendToTerminal(`User: ${usernameParam}\r\n\r\n`)
        
        // Auto-connect if password is provided
        if (passwordParam) {
          appendToTerminal(`Auto-connecting with provided credentials...\r\n\r\n`)
          setTimeout(() => {
            connect()
          }, 1000)
        } else {
          appendToTerminal(`Please enter password and click Connect.\r\n\r\n`)
        }
        
        return
      }
      
      // Check if agentId is provided in query parameter
      const queryAgentId = route.query.agentId
      if (queryAgentId) {
        agentId.value = queryAgentId
        terminalContent.value = `SSH Web Terminal - Agent Mode\r\nConnecting via Agent: ${queryAgentId}\r\n\r\n`
        
        try {
          // Get agent details from API
          const response = await apiService.getAgents()
          const agent = response.data.find(a => a.id === queryAgentId)
          
          if (agent) {
            // Pre-fill connection details for agent tunnel
            connection.host = 'localhost'  // Agent akan handle tunneling
            connection.port = 22           // Default SSH port
            connection.username = 'root'   // Default username, user bisa ubah
            connection.password = ''       // User harus input password
            
            appendToTerminal(`Agent found: ${agent.id}\r\nPlease enter credentials and click Connect.\r\n\r\n`)
          } else {
            appendToTerminal(`Agent ${queryAgentId} not found. Please enter connection details manually.\r\n\r\n`)
          }
        } catch (error) {
          appendToTerminal(`Error fetching agent details: ${error.message}\r\n\r\n`)
        }
      }
    })
    
    onUnmounted(() => {
      disconnect()
    })
    
    return {
      terminal,
      terminalContent,
      isConnected,
      isConnecting,
      connection,
      connectionStatusText,
      connectionStatusClass,
      focusTerminal,
      toggleConnection,
      handleKeyDown
    }
  }
}
</script>

<style scoped>
.ssh-terminal-container {
  max-width: 900px;
  margin: 0 auto;
  background: linear-gradient(135deg, #1e293b, #334155);
  border-radius: 8px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.4);
  overflow: hidden;
  border: 1px solid #475569;
}

.terminal-header {
  background: linear-gradient(135deg, #0f172a, #1e293b);
  padding: 15px 20px;
  border-bottom: 1px solid #475569;
}

.terminal-header h2 {
  font-size: 1.4rem;
  color: #22c55e;
  margin: 0;
  text-shadow: 0 0 8px rgba(34, 197, 94, 0.3);
}

.connection-form {
  padding: 20px;
  background-color: #0d0d0d;
  border-bottom: 1px solid #2a6a2a;
  display: flex;
  flex-wrap: wrap;
  gap: 15px;
}

.form-group {
  flex: 1;
  min-width: 180px;
}

.form-group label {
  display: block;
  margin-bottom: 5px;
  font-size: 0.9rem;
  color: #22c55e;
  font-weight: 500;
}

.form-group input {
  width: 100%;
  padding: 8px 12px;
  background-color: #0f172a;
  border: 1px solid #475569;
  border-radius: 4px;
  color: #e2e8f0;
  font-size: 0.9rem;
  font-family: 'Courier New', monospace;
  transition: border-color 0.2s, box-shadow 0.2s;
}

.form-group input:focus {
  outline: none;
  border-color: #22c55e;
  box-shadow: 0 0 8px rgba(34, 197, 94, 0.3);
}

.connect-btn {
  padding: 8px 20px;
  background: linear-gradient(135deg, #16a34a, #22c55e);
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 0.9rem;
  font-weight: 500;
  transition: all 0.2s;
  align-self: flex-end;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.3);
}

.connect-btn:hover {
  background: linear-gradient(135deg, #22c55e, #4ade80);
  box-shadow: 0 2px 8px rgba(74, 222, 128, 0.3);
}

.connect-btn:disabled {
  background: #404040;
  cursor: not-allowed;
  box-shadow: none;
}

.terminal-container {
  height: calc(100vh - 120px); /* Expand terminal height since form is hidden */
  overflow-y: auto;
  background: linear-gradient(135deg, #0f172a, #1e293b);
}

.terminal {
  padding: 15px;
  height: 100%;
  overflow-y: auto;
  white-space: pre-wrap;
  word-wrap: break-word;
  font-size: 14px;
  line-height: 1.2;
  color: #22c55e;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  cursor: text;
  background: linear-gradient(135deg, #0f172a, #1e293b);
  outline: none;
  /* Enhanced terminal display properties */
  text-rendering: optimizeSpeed;
  font-variant-ligatures: none;
  font-feature-settings: "liga" 0;
  letter-spacing: 0;
  word-spacing: 0;
  /* Better text selection */
  user-select: text;
  -webkit-user-select: text;
  -moz-user-select: text;
  -ms-user-select: text;
  /* Prevent font smoothing issues */
  -webkit-font-smoothing: auto;
  -moz-osx-font-smoothing: auto;
}

.terminal:focus {
  background: linear-gradient(135deg, #0f172a, #1e293b);
  outline: none;
  border: 1px solid #22c55e;
}

.status-bar {
  padding: 8px 15px;
  background: linear-gradient(135deg, #0f172a, #1e293b);
  border-top: 1px solid #475569;
  font-size: 0.8rem;
  display: flex;
  justify-content: space-between;
}

.connection-status {
  color: #ef4444;
  font-weight: 500;
}

.connection-status.connected {
  color: #22c55e;
}

.connection-status.connecting {
  color: #eab308;
}

.connection-info {
  color: #94a3b8;
  font-size: 0.75rem;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
}

/* Custom scrollbar for terminal */
.terminal::-webkit-scrollbar {
  width: 8px;
}

.terminal::-webkit-scrollbar-track {
  background: #0f172a;
}

.terminal::-webkit-scrollbar-thumb {
  background: #475569;
  border-radius: 4px;
}

.terminal::-webkit-scrollbar-thumb:hover {
  background: #22c55e;
}

/* Responsive design */
@media (max-width: 768px) {
  .connection-form {
    flex-direction: column;
  }
  
  .form-group {
    min-width: 100%;
  }
  
  .terminal-container {
    height: 300px;
  }
}
</style>