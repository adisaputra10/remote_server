<template>
  <div>
    <div class="flex justify-between items-center mb-4">
      <h3 class="text-lg font-medium text-gray-900">Connection Logs</h3>
      <button
        @click="$emit('refresh')"
        class="bg-blue-600 hover:bg-blue-700 text-white px-3 py-1 rounded text-sm"
      >
        Refresh
      </button>
    </div>
    
    <div class="overflow-hidden shadow ring-1 ring-black ring-opacity-5 md:rounded-lg">
      <table class="min-w-full divide-y divide-gray-300">
        <thead class="bg-gray-50">
          <tr>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Timestamp
            </th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Type
            </th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Event
            </th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Agent ID
            </th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Client ID
            </th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Details
            </th>
          </tr>
        </thead>
        <tbody class="bg-white divide-y divide-gray-200">
          <tr v-for="log in logs" :key="log.id">
            <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
              {{ formatDate(log.timestamp) }}
            </td>
            <td class="px-6 py-4 whitespace-nowrap">
              <span
                :class="[
                  getTypeColor(log.type),
                  'inline-flex px-2 py-1 text-xs font-semibold rounded-full'
                ]"
              >
                {{ log.type }}
              </span>
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
              {{ log.event }}
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
              {{ log.agent_id || '-' }}
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
              {{ log.client_id || '-' }}
            </td>
            <td class="px-6 py-4 text-sm text-gray-500 max-w-xs truncate">
              {{ log.details || '-' }}
            </td>
          </tr>
          <tr v-if="logs.length === 0">
            <td colspan="6" class="px-6 py-4 whitespace-nowrap text-sm text-gray-500 text-center">
              No logs available
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script>
export default {
  name: 'LogsTable',
  props: {
    logs: {
      type: Array,
      default: () => []
    }
  },
  emits: ['refresh'],
  methods: {
    formatDate(dateString) {
      if (!dateString) return 'N/A'
      return new Date(dateString).toLocaleString()
    },
    getTypeColor(type) {
      const colors = {
        'agent': 'bg-blue-100 text-blue-800',
        'client': 'bg-green-100 text-green-800',
        'database': 'bg-purple-100 text-purple-800',
        'error': 'bg-red-100 text-red-800'
      }
      return colors[type] || 'bg-gray-100 text-gray-800'
    }
  }
}
</script>