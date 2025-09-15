<template>
  <div class="min-h-screen bg-gray-50">
    <!-- Navigation -->
    <nav class="bg-white shadow">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="flex justify-between h-16">
          <div class="flex items-center">
            <h1 class="text-xl font-semibold text-gray-900">SSH Tunnel Dashboard</h1>
          </div>
          <div class="flex items-center space-x-4">
            <span class="text-sm text-gray-700">Welcome, {{ username }}</span>
            <button
              @click="logout"
              class="bg-red-600 hover:bg-red-700 text-white px-3 py-2 rounded-md text-sm font-medium"
            >
              Logout
            </button>
          </div>
        </div>
      </div>
    </nav>

    <!-- Main Content -->
    <div class="max-w-7xl mx-auto py-6 sm:px-6 lg:px-8">
      <!-- Stats Cards -->
      <div class="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
        <div class="bg-white overflow-hidden shadow rounded-lg">
          <div class="p-5">
            <div class="flex items-center">
              <div class="flex-shrink-0">
                <div class="w-8 h-8 bg-blue-500 rounded-full flex items-center justify-center">
                  <span class="text-white text-sm font-medium">A</span>
                </div>
              </div>
              <div class="ml-5 w-0 flex-1">
                <dl>
                  <dt class="text-sm font-medium text-gray-500 truncate">Connected Agents</dt>
                  <dd class="text-lg font-medium text-gray-900">{{ agents.length }}</dd>
                </dl>
              </div>
            </div>
          </div>
        </div>

        <div class="bg-white overflow-hidden shadow rounded-lg">
          <div class="p-5">
            <div class="flex items-center">
              <div class="flex-shrink-0">
                <div class="w-8 h-8 bg-green-500 rounded-full flex items-center justify-center">
                  <span class="text-white text-sm font-medium">C</span>
                </div>
              </div>
              <div class="ml-5 w-0 flex-1">
                <dl>
                  <dt class="text-sm font-medium text-gray-500 truncate">Active Clients</dt>
                  <dd class="text-lg font-medium text-gray-900">{{ clients.length }}</dd>
                </dl>
              </div>
            </div>
          </div>
        </div>

        <div class="bg-white overflow-hidden shadow rounded-lg">
          <div class="p-5">
            <div class="flex items-center">
              <div class="flex-shrink-0">
                <div class="w-8 h-8 bg-purple-500 rounded-full flex items-center justify-center">
                  <span class="text-white text-sm font-medium">L</span>
                </div>
              </div>
              <div class="ml-5 w-0 flex-1">
                <dl>
                  <dt class="text-sm font-medium text-gray-500 truncate">Total Logs</dt>
                  <dd class="text-lg font-medium text-gray-900">{{ logs.length }}</dd>
                </dl>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- Tabs -->
      <div class="bg-white shadow rounded-lg">
        <div class="border-b border-gray-200">
          <nav class="-mb-px flex space-x-8 px-6" aria-label="Tabs">
            <button
              v-for="tab in tabs"
              :key="tab.id"
              @click="activeTab = tab.id"
              :class="[
                activeTab === tab.id
                  ? 'border-indigo-500 text-indigo-600'
                  : 'border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300',
                'whitespace-nowrap py-4 px-1 border-b-2 font-medium text-sm'
              ]"
            >
              {{ tab.name }}
            </button>
          </nav>
        </div>

        <!-- Tab Content -->
        <div class="p-6">
          <!-- Agents Tab -->
          <div v-if="activeTab === 'agents'">
            <AgentsTable :agents="agents" @refresh="fetchAgents" />
          </div>

          <!-- Clients Tab -->
          <div v-if="activeTab === 'clients'">
            <ClientsTable :clients="clients" @refresh="fetchClients" />
          </div>

          <!-- Connection Logs Tab -->
          <div v-if="activeTab === 'logs'">
            <LogsTable :logs="logs" @refresh="fetchLogs" />
          </div>

          <!-- Database Queries Tab -->
          <div v-if="activeTab === 'queries'">
            <QueriesTable :queries="tunnelLogs" @refresh="fetchTunnelLogs" />
          </div>

          <!-- SSH Commands Tab -->
          <div v-if="activeTab === 'ssh'">
            <SSHLogsTable :sshLogs="sshLogs" @refresh="fetchSSHLogs" />
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import api from '../config/api.js'
import AgentsTable from '../components/AgentsTable.vue'
import ClientsTable from '../components/ClientsTable.vue'
import LogsTable from '../components/LogsTable.vue'
import QueriesTable from '../components/QueriesTable.vue'
import SSHLogsTable from '../components/SSHLogsTable.vue'

export default {
  name: 'Dashboard',
  components: {
    AgentsTable,
    ClientsTable,
    LogsTable,
    QueriesTable,
    SSHLogsTable
  },
  data() {
    return {
      username: '',
      activeTab: 'agents',
      agents: [],
      clients: [],
      logs: [],
      tunnelLogs: [],
      sshLogs: [],
      tabs: [
        { id: 'agents', name: 'Agents' },
        { id: 'clients', name: 'Clients' },
        { id: 'logs', name: 'Connection Logs' },
        { id: 'queries', name: 'Database Queries' },
        { id: 'ssh', name: 'SSH Commands' }
      ]
    }
  },
  mounted() {
    this.username = localStorage.getItem('username')
    this.fetchData()
    
    // Auto refresh every 30 seconds
    this.refreshInterval = setInterval(() => {
      this.fetchData()
    }, 30000)
  },
  beforeUnmount() {
    if (this.refreshInterval) {
      clearInterval(this.refreshInterval)
    }
  },
  methods: {
    async fetchData() {
      await Promise.all([
        this.fetchAgents(),
        this.fetchClients(),
        this.fetchLogs(),
        this.fetchTunnelLogs(),
        this.fetchSSHLogs()
      ])
    },
    async fetchAgents() {
      try {
        const response = await this.apiCall('/api/agents')
        this.agents = response.data || []
      } catch (error) {
        console.error('Error fetching agents:', error)
      }
    },
    async fetchClients() {
      try {
        const response = await this.apiCall('/api/clients')
        this.clients = response.data || []
      } catch (error) {
        console.error('Error fetching clients:', error)
      }
    },
    async fetchLogs() {
      try {
        const response = await this.apiCall('/api/logs')
        this.logs = response.data || []
      } catch (error) {
        console.error('Error fetching logs:', error)
      }
    },
    async fetchTunnelLogs() {
      try {
        const response = await this.apiCall('/api/tunnel-logs')
        this.tunnelLogs = response.data || []
      } catch (error) {
        console.error('Error fetching tunnel logs:', error)
      }
    },
    async fetchSSHLogs() {
      try {
        const response = await this.apiCall('/api/ssh-logs')
        this.sshLogs = response.data || []
      } catch (error) {
        console.error('Error fetching SSH logs:', error)
      }
    },
    async apiCall(url) {
      const auth = localStorage.getItem('auth_token')
      return api.get(url, {
        headers: {
          'Authorization': `Basic ${auth}`
        }
      })
    },
    logout() {
      localStorage.removeItem('auth_token')
      localStorage.removeItem('username')
      this.$router.push('/login')
    }
  }
}
</script>