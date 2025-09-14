<template>
  <div>
    <div class="flex justify-between items-center mb-4">
      <h3 class="text-lg font-medium text-gray-900">Database Queries</h3>
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
              Agent ID
            </th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Client ID
            </th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Protocol
            </th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Operation
            </th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Table
            </th>
            <th class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
              Query
            </th>
          </tr>
        </thead>
        <tbody class="bg-white divide-y divide-gray-200">
          <tr v-for="query in queries" :key="query.id">
            <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
              {{ formatDate(query.timestamp) }}
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
              {{ query.agent_id || '-' }}
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
              {{ query.client_id || '-' }}
            </td>
            <td class="px-6 py-4 whitespace-nowrap">
              <span
                :class="[
                  getProtocolColor(query.protocol),
                  'inline-flex px-2 py-1 text-xs font-semibold rounded-full'
                ]"
              >
                {{ query.protocol }}
              </span>
            </td>
            <td class="px-6 py-4 whitespace-nowrap">
              <span
                :class="[
                  getOperationColor(query.operation),
                  'inline-flex px-2 py-1 text-xs font-semibold rounded-full'
                ]"
              >
                {{ query.operation }}
              </span>
            </td>
            <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
              {{ query.table_name || '-' }}
            </td>
            <td class="px-6 py-4 text-sm text-gray-500 max-w-md">
              <div class="truncate" :title="query.query_text">
                {{ truncateQuery(query.query_text) }}
              </div>
            </td>
          </tr>
          <tr v-if="queries.length === 0">
            <td colspan="7" class="px-6 py-4 whitespace-nowrap text-sm text-gray-500 text-center">
              No database queries logged
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script>
export default {
  name: 'QueriesTable',
  props: {
    queries: {
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
    getProtocolColor(protocol) {
      const colors = {
        'mysql': 'bg-blue-100 text-blue-800',
        'postgresql': 'bg-indigo-100 text-indigo-800',
        'mongodb': 'bg-green-100 text-green-800',
        'redis': 'bg-red-100 text-red-800'
      }
      return colors[protocol] || 'bg-gray-100 text-gray-800'
    },
    getOperationColor(operation) {
      const colors = {
        'SELECT': 'bg-green-100 text-green-800',
        'INSERT': 'bg-blue-100 text-blue-800',
        'UPDATE': 'bg-yellow-100 text-yellow-800',
        'DELETE': 'bg-red-100 text-red-800',
        'CREATE': 'bg-purple-100 text-purple-800',
        'DROP': 'bg-red-100 text-red-800'
      }
      return colors[operation] || 'bg-gray-100 text-gray-800'
    },
    truncateQuery(query) {
      if (!query) return '-'
      return query.length > 100 ? query.substring(0, 100) + '...' : query
    }
  }
}
</script>