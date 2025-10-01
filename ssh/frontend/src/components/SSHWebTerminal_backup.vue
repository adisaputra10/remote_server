<template>
  <div class="ssh-terminal-container">
    <div class="terminal-header">
      <div class="header-info">
        <h2>SSH Web Terminal</h2>
        <div v-if="connectionInfo.host" class="connection-details">
          Connecting to: {{ connectionInfo.username }}@{{ connectionInfo.host }}:{{ connectionInfo.port }}
        </div>
      </div>
      <div class="connection-controls">
        <div class="connection-status" :class="connectionStatusClass">
          {{ connectionStatus }}
        </div>
        <button 
          v-if="!isConnected" 
          @click="connect" 
          :disabled="isConnecting || !canConnect"
          class="connect-btn">
          {{ isConnecting ? 'Connecting...' : 'Connect' }}
        </button>
        <button 
          v-else 
          @click="disconnect" 
          class="disconnect-btn">
          Disconnect
        </button>
      </div>
    </div>

    <!-- Connection Form (shown when not auto-connecting) -->
    <div v-if="!hasAutoConnectParams && !isConnected" class="connection-form">
      <div class="form-row">
        <div class="form-group">
          <label for="host">Host:</label>
          <input 
            type="text" 
            id="host" 
            v-model="connectionForm.host" 
            placeholder="example.com or IP"
            @keyup.enter="connect">
        </div>
        <div class="form-group">
          <label for="port">Port:</label>
          <input 
            type="number" 
            id="port" 
            v-model.number="connectionForm.port" 
            placeholder="22"
            @keyup.enter="connect">
        </div>
      </div>
      <div class="form-row">
        <div class="form-group">
          <label for="username">Username:</label>
          <input 
            type="text" 
            id="username" 
            v-model="connectionForm.username" 
            placeholder="username"
            @keyup.enter="connect">
        </div>
        <div class="form-group">
          <label for="password">Password:</label>
          <input 
            type="password" 
            id="password" 
            v-model="connectionForm.password" 
            placeholder="password"
            @keyup.enter="connect">
        </div>
      </div>
    </div>

    <!-- Password Form (shown when auto-connecting) -->
    <div v-if="hasAutoConnectParams && !isConnected" class="password-form">
      <div class="form-group">
        <label for="auto-password">Password for {{ connectionInfo.username }}@{{ connectionInfo.host }}:</label>
        <input 
          type="password" 
          id="auto-password" 
          v-model="connectionForm.password" 
          placeholder="Enter password"
          @keyup.enter="connect"
          ref="passwordInput">
      </div>
    </div>

    <!-- Terminal -->
    <div class="terminal-wrapper" @click="focusTerminal">
      <div 
        ref="terminal" 
        class="terminal" 
        tabindex="0"
        @keydown="handleKeyDown"
        @keypress="handleKeyPress"
        @click="focusTerminal"
        @focus="() => console.log('Terminal focused')"
        @blur="() => console.log('Terminal blurred')">
        {{ terminalContent }}
      </div>
    </div>
  </div>
</template>

<script>
import { ref, onMounted, onUnmounted, nextTick, computed } from 'vue'
import { useRoute } from 'vue-router'

export default {
  name: 'SSHWebTerminal',
  setup() {
    const route = useRoute()
    const terminal = ref(null)
    const passwordInput = ref(null)
    
    // Connection state
    const socket = ref(null)
    const isConnected = ref(false)
    const isConnecting = ref(false)
    const connectionStatus = ref('Disconnected')
    const terminalContent = ref('SSH Web Terminal\nPlease enter connection details and click Connect.\n\n')
    
    // Connection form
    const connectionForm = ref({
      host: '',
      port: 22,
      username: '',
      password: ''
    })
    
    // Connection info from URL params
    const connectionInfo = ref({
      host: '',
      port: 22,
      username: '',
      tunnel_id: ''
    })
    
    // Check if we have auto-connect parameters
    const hasAutoConnectParams = computed(() => {
      return connectionInfo.value.host && connectionInfo.value.username
    })
    
    const canConnect = computed(() => {
      if (hasAutoConnectParams.value) {
        return connectionForm.value.password.trim() !== ''
      }
      return connectionForm.value.host.trim() !== '' &&
             connectionForm.value.port > 0 &&
             connectionForm.value.username.trim() !== '' &&
             connectionForm.value.password.trim() !== ''
    })
    
    const connectionStatusClass = computed(() => {
      if (isConnected.value) return 'status-connected'
      if (isConnecting.value) return 'status-connecting'
      return 'status-disconnected'
    })
    
    // Initialize from URL parameters
    const initializeFromRoute = () => {
      const params = route.query
      if (params.host) {
        connectionInfo.value.host = params.host
        connectionForm.value.host = params.host
      }
      if (params.port) {
        connectionInfo.value.port = parseInt(params.port)
        connectionForm.value.port = parseInt(params.port)
      }
      if (params.username) {
        connectionInfo.value.username = params.username
        connectionForm.value.username = params.username
      }
      if (params.tunnel_id) {
        connectionInfo.value.tunnel_id = params.tunnel_id
      }
      
      // Update terminal content if we have auto-connect params
      if (hasAutoConnectParams.value) {
        terminalContent.value = `SSH Web Terminal\nReady to connect to ${connectionInfo.value.username}@${connectionInfo.value.host}:${connectionInfo.value.port}\nPlease enter your password and click Connect.\n\n`
        
        // Focus password input after component mounts
        nextTick(() => {
          if (passwordInput.value) {
            passwordInput.value.focus()
          }
        })
      }
    }
    
    // WebSocket connection
    const connect = () => {
      const host = connectionForm.value.host.trim()
      const port = connectionForm.value.port
      const username = connectionForm.value.username.trim()
      const password = connectionForm.value.password
      
      if (!canConnect.value) {
        appendToTerminal('Error: Please fill in all connection details.\n')
        return
      }
      
      isConnecting.value = true
      connectionStatus.value = 'Connecting...'
      
      // Create WebSocket connection to relay server SSH endpoint
      const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
      const wsUrl = `${protocol}//localhost:8080/ws/ssh`
      
      try {
        socket.value = new WebSocket(wsUrl)
        
        // Optimize WebSocket settings
        socket.value.binaryType = 'arraybuffer'
        
        socket.value.onopen = () => {
          // Send connection details to server immediately
          socket.value.send(JSON.stringify({
            type: 'connect',
            host: host,
            port: port,
            username: username,
            password: password
          }))
        }
        
        socket.value.onmessage = (event) => {
          try {
            const data = JSON.parse(event.data)
            
            switch (data.type) {
              case 'connected':
                isConnected.value = true
                isConnecting.value = false
                connectionStatus.value = 'Connected'
                appendToTerminal(`\nConnected to ${host}\n`)
                focusTerminal()
                break
                
              case 'data':
                appendToTerminal(data.data)
                break
                
              case 'disconnected':
                isConnected.value = false
                isConnecting.value = false
                connectionStatus.value = 'Disconnected'
                appendToTerminal('\nConnection closed.\n')
                break
                
              case 'error':
                isConnecting.value = false
                connectionStatus.value = 'Error'
                appendToTerminal(`\nError: ${data.message}\n`)
                break
            }
          } catch (err) {
            console.error('Error parsing message:', err)
          }
        }
        
        socket.value.onclose = () => {
          isConnected.value = false
          isConnecting.value = false
          connectionStatus.value = 'Disconnected'
          appendToTerminal('\nConnection closed.\n')
        }
        
        socket.value.onerror = (err) => {
          console.error('WebSocket error:', err)
          isConnecting.value = false
          connectionStatus.value = 'Error'
          appendToTerminal('\nConnection error occurred.\n')
        }
        
      } catch (err) {
        console.error('Failed to create WebSocket:', err)
        isConnecting.value = false
        connectionStatus.value = 'Error'
        appendToTerminal(`\nFailed to connect: ${err.message}\n`)
      }
    }
    
    const disconnect = () => {
      if (socket.value) {
        socket.value.close()
        socket.value = null
      }
      isConnected.value = false
      isConnecting.value = false
      connectionStatus.value = 'Disconnected'
      appendToTerminal('\nDisconnected.\n')
    }
    
    const appendToTerminal = (text) => {
      // Minimal ANSI cleanup for better performance
      const cleanText = text
        .replace(/\x1b\[2004[hl]/g, '') // Remove bracketed paste mode only
        .replace(/\x1b\[\?2004[hl]/g, '') // Remove bracketed paste mode only
      
      terminalContent.value += cleanText
      
      // Immediate scroll without debouncing for real-time feel
      nextTick(() => {
        if (terminal.value) {
          terminal.value.scrollTop = terminal.value.scrollHeight
        }
      })
    }
    
    const focusTerminal = () => {
      if (terminal.value) {
        terminal.value.focus()
      }
    }
    
    const handleKeyDown = (event) => {
      if (!isConnected.value || !socket.value) {
        console.log('Not connected or no socket')
        return
      }
      
      event.preventDefault()
      
      let data = ''
      
      // Handle special keys
      if (event.key === 'Enter') {
        data = '\r'
      } else if (event.key === 'Backspace') {
        data = '\b'
      } else if (event.key === 'Tab') {
        data = '\t'
      } else if (event.key === 'ArrowUp') {
        data = '\x1b[A'
      } else if (event.key === 'ArrowDown') {
        data = '\x1b[B'
      } else if (event.key === 'ArrowRight') {
        data = '\x1b[C'
      } else if (event.key === 'ArrowLeft') {
        data = '\x1b[D'
      } else if (event.key === 'Delete') {
        data = '\x1b[3~'
      } else if (event.key === 'Home') {
        data = '\x1b[H'
      } else if (event.key === 'End') {
        data = '\x1b[F'
      } else if (event.key === 'PageUp') {
        data = '\x1b[5~'
      } else if (event.key === 'PageDown') {
        data = '\x1b[6~'
      } else if (event.ctrlKey && event.key === 'c') {
        data = '\x03' // Ctrl+C
      } else if (event.ctrlKey && event.key === 'd') {
        data = '\x04' // Ctrl+D
      } else if (event.ctrlKey && event.key === 'z') {
        data = '\x1a' // Ctrl+Z
      } else if (event.key.length === 1) {
        data = event.key
      }
      
      if (data && socket.value) {
        console.log('Sending key:', event.key, 'as:', JSON.stringify(data))
        socket.value.send(JSON.stringify({
          type: 'data',
          data: data
        }))
      }
    }
    
    const handleKeyPress = (event) => {
      // Additional handler for character input
      if (!isConnected.value || !socket.value) return
      
      // Let keydown handle most keys, this is for fallback
      if (event.key.length === 1 && !event.ctrlKey && !event.altKey && !event.metaKey) {
        console.log('Key press:', event.key)
      }
    }
    
    onMounted(() => {
      initializeFromRoute()
      
      // Auto-focus terminal after mount
      nextTick(() => {
        focusTerminal()
        console.log('Terminal mounted and focused')
      })
    })
    
    onUnmounted(() => {
      if (socket.value) {
        socket.value.close()
      }
    })
    
    return {
      terminal,
      passwordInput,
      isConnected,
      isConnecting,
      connectionStatus,
      connectionStatusClass,
      terminalContent,
      connectionForm,
      connectionInfo,
      hasAutoConnectParams,
      canConnect,
      connect,
      disconnect,
      handleKeyDown,
      handleKeyPress,
      focusTerminal
    }
  }
}
</script>

<style scoped>
.ssh-terminal-container {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: #2a2a2a; /* Darker background like screenshot */
  color: var(--text-primary);
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
}

.terminal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  background: #1a1a1a; /* Very dark header */
  border-bottom: 1px solid #444;
}

.header-info h2 {
  margin: 0;
  color: #ffffff;
  font-size: 18px;
  font-weight: 400;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
}

.connection-details {
  color: #888;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 12px;
  margin-top: 4px;
}

.connection-controls {
  display: flex;
  align-items: center;
  gap: 16px;
}

.connection-status {
  padding: 4px 12px;
  border-radius: 12px;
  font-size: 11px;
  font-weight: 400;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  text-transform: uppercase;
}

.status-connected {
  background: #006600;
  color: #ffffff;
}

.status-connecting {
  background: #cc6600;
  color: #ffffff;
}

.status-disconnected {
  background: #cc0000;
  color: #ffffff;
}

.connect-btn, .disconnect-btn {
  padding: 8px 16px;
  border: none;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 400;
  cursor: pointer;
  transition: all 0.2s ease;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
}

.connect-btn {
  background: #0066cc;
  color: white;
}

.connect-btn:hover:not(:disabled) {
  background: #0052a3;
}

.connect-btn:disabled {
  background: #555;
  color: #888;
  cursor: not-allowed;
}

.disconnect-btn {
  background: #cc0000;
  color: white;
}

.disconnect-btn:hover {
  background: #a30000;
}

.connection-form, .password-form {
  padding: 16px 20px;
  background: #1a1a1a; /* Dark form background */
  border-bottom: 1px solid #444;
}

.form-row {
  display: flex;
  gap: 16px;
  margin-bottom: 12px;
}

.form-group {
  flex: 1;
}

.form-group label {
  display: block;
  margin-bottom: 6px;
  color: #ffffff;
  font-weight: 400;
  font-size: 12px;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
}

.form-group input {
  width: 100%;
  padding: 8px 12px;
  border: 1px solid #555;
  border-radius: 4px;
  background: #333;
  color: #ffffff;
  font-size: 12px;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
}

.form-group input:focus {
  outline: none;
  border-color: #0066cc;
  box-shadow: 0 0 0 2px rgba(0, 102, 204, 0.2);
}

.terminal-wrapper {
  flex: 1;
  padding: 0;
  overflow: hidden;
  background: #000000; /* Pure black like screenshot */
}

.terminal {
  width: 100%;
  height: 100%;
  background: #000000; /* Pure black background */
  color: #00ff00; /* Bright green text */
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.2;
  padding: 12px;
  border: none;
  white-space: pre;
  word-wrap: break-word;
  overflow-y: auto;
  overflow-x: auto;
  cursor: text;
  letter-spacing: 0;
  /* Crisp text rendering like real terminal */
  -webkit-font-smoothing: auto;
  -moz-osx-font-smoothing: auto;
  text-rendering: optimizeSpeed;
}

.terminal:focus {
  outline: none;
  box-shadow: inset 0 0 0 1px #00ff00;
}

/* Scrollbar styling for terminal */
.terminal::-webkit-scrollbar {
  width: 12px;
}

.terminal::-webkit-scrollbar-track {
  background: #000000;
}

.terminal::-webkit-scrollbar-thumb {
  background: #333333;
  border-radius: 6px;
}

.terminal::-webkit-scrollbar-thumb:hover {
  background: #555555;
}

/* Additional terminal text colors for better readability */
.terminal {
  /* Make sure text is always visible */
  text-shadow: none;
  /* Better contrast */
  background: #000000 !important;
  color: #00ff41 !important; /* Matrix green */
}
</style>