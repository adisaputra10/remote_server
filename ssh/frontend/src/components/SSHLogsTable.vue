<template>
  <div class="bg-white shadow overflow-hidden sm:rounded-md">
    <div class="px-4 py-5 sm:px-6 flex justify-between items-center">
      <div>
        <h3 class="text-lg leading-6 font-medium text-gray-900">SSH Commands</h3>
        <p class="mt-1 max-w-2xl text-sm text-gray-500">
          Real-time SSH command logging and monitoring
        </p>
      </div>
      <button
        @click="refresh"
        class="bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-md text-sm font-medium"
      >
        Refresh
      </button>
    </div>
    
    <div class="border-t border-gray-200">
      <div v-if="sshLogs.length === 0" class="p-4 text-center text-gray-500">
        No SSH commands logged yet
      </div>
      
      <div v-else class="overflow-x-auto">
        <table class="min-w-full divide-y divide-gray-200">
          <thead class="bg-gray-50">
            <tr>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Timestamp
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Session
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Agent
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Client
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                User@Host:Port
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Direction
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Command
              </th>
              <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                Size
              </th>
            </tr>
          </thead>
          <tbody class="bg-white divide-y divide-gray-200">
            <tr v-for="log in sshLogs" :key="log.id" class="hover:bg-gray-50">
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                {{ formatTimestamp(log.timestamp) }}
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                <span class="font-mono text-xs bg-gray-100 px-2 py-1 rounded">
                  {{ log.session_id.substring(0, 8) }}...
                </span>
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                  {{ log.agent_id || 'N/A' }}
                </span>
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                <span class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-blue-100 text-blue-800">
                  {{ log.client_id || 'N/A' }}
                </span>
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                <span class="font-mono">{{ log.ssh_user }}@{{ log.ssh_host }}:{{ log.ssh_port }}</span>
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm">
                <span 
                  :class="getDirectionClass(log.direction)"
                  class="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium"
                >
                  {{ log.direction }}
                </span>
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                <span 
                  class="font-mono text-sm bg-gray-100 px-2 py-1 rounded max-w-xs truncate inline-block"
                  :title="log.command"
                >
                  {{ log.command || 'N/A' }}
                </span>
              </td>
              <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                {{ formatDataSize(log.data_size) }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      
      <!-- Pagination info -->
      <div v-if="sshLogs.length > 0" class="bg-white px-4 py-3 border-t border-gray-200 sm:px-6">
        <div class="flex-1 flex justify-between sm:hidden">
          <p class="text-sm text-gray-700">
            Showing {{ sshLogs.length }} SSH commands
          </p>
        </div>
        <div class="hidden sm:flex-1 sm:flex sm:items-center sm:justify-between">
          <div>
            <p class="text-sm text-gray-700">
              Showing latest <span class="font-medium">{{ sshLogs.length }}</span> SSH commands
            </p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'SSHLogsTable',
  props: {
    sshLogs: {
      type: Array,
      default: () => []
    }
  },
  emits: ['refresh'],
  methods: {
    refresh() {
      this.$emit('refresh')
    },
    formatTimestamp(timestamp) {
      return new Date(timestamp).toLocaleString()
    },
    formatDataSize(size) {
      if (!size || size === 0) return '0 B'
      
      const units = ['B', 'KB', 'MB', 'GB']
      let unitIndex = 0
      let sizeValue = size
      
      while (sizeValue >= 1024 && unitIndex < units.length - 1) {
        sizeValue /= 1024
        unitIndex++
      }
      
      return `${sizeValue.toFixed(1)} ${units[unitIndex]}`
    },
    getDirectionClass(direction) {
      switch (direction) {
        case 'inbound':
          return 'bg-green-100 text-green-800'
        case 'outbound':
          return 'bg-yellow-100 text-yellow-800'
        default:
          return 'bg-gray-100 text-gray-800'
      }
    }
  }
}
</script>

<style scoped>
.truncate {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>