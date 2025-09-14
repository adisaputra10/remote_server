<template>
  <div>
    <div class="flex justify-between items-center mb-4">
      <h3 class="text-lg font-medium text-gray-900">Connected Agents</h3>
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
              Agent ID
            </th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Status
            </th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Connected At
            </th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Last Ping
            </th>
          </tr>
        </thead>
        <tbody class="bg-white divide-y divide-gray-200">
          <tr v-for="agent in agents" :key="agent.id">
            <td class="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
              {{ agent.id }}
            </td>
            <td class="px-6 py-4 whitespace-nowrap">
              <span
                :class="[
                  agent.status === 'connected'
                    ? 'bg-green-100 text-green-800'
                    : 'bg-red-100 text-red-800',
                  'inline-flex px-2 py-1 text-xs font-semibold rounded-full'
                ]"
              >
                {{ agent.status }}
              </span>
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
              {{ formatDate(agent.connected_at) }}
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
              {{ formatDate(agent.last_ping) }}
            </td>
          </tr>
          <tr v-if="agents.length === 0">
            <td colspan="4" class="px-6 py-4 whitespace-nowrap text-sm text-gray-500 text-center">
              No agents connected
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script>
export default {
  name: 'AgentsTable',
  props: {
    agents: {
      type: Array,
      default: () => []
    }
  },
  emits: ['refresh'],
  methods: {
    formatDate(dateString) {
      if (!dateString) return 'N/A'
      return new Date(dateString).toLocaleString()
    }
  }
}
</script>