# SSH Logs Table Component

## Overview

The `SSHLogsTable.vue` component is a Vue 3 component that displays SSH command logs in a real-time table format. It fetches SSH logs from the API and presents them in a paginated table with automatic refresh functionality.

## Features

- **Real-time SSH log monitoring** with auto-refresh every 15 seconds
- **Paginated table display** with customizable items per page
- **Responsive table layout** with optimized column widths
- **Error handling** for API failures and authentication issues
- **Loading states** with spinner indicators
- **Manual refresh** functionality
- **Enhanced DATA column** for better command/output visibility

## Component Structure

### Template Section

The template consists of several main sections:

1. **Header Section**
   - Title: "SSH Commands"
   - Refresh button for manual data updates

2. **State Management**
   - Loading state with spinner
   - Error state with error messages
   - Empty state when no logs are available

3. **Table Display**
   - 5 columns: TIMESTAMP, AGENT, CLIENT, DIRECTION, DATA
   - Responsive table with scroll functionality
   - Pagination component at the bottom

### Script Section (Vue 3 Composition API)

#### Reactive Variables
- `allSSHLogs`: Array containing all SSH logs from API
- `loading`: Boolean indicating fetch status
- `error`: String containing error messages
- `currentPage`: Current pagination page number
- `itemsPerPage`: Number of items to display per page (default: 20)

#### Computed Properties
- `paginatedSSHLogs`: Filtered logs for current page
- `apiBaseUrl`: Base URL for API calls from environment variables

#### Core Functions

##### `fetchSSHLogs()`
Main function that fetches SSH logs from the API:
- Makes authenticated API call to `/api/ssh-logs`
- Validates response data structure
- Transforms API data into component-friendly format
- Handles various error scenarios (401, 404, network errors)
- Ensures no dummy/fallback data is used

**Data Transformation Process:**
1. Parses JSON if response is string format
2. Extracts fields from various possible API field names
3. Builds user@host:port format from connection details
4. Creates standardized log objects with consistent field names

##### `refreshData()`
Manual refresh function that:
- Clears existing data
- Triggers fresh API fetch
- Provides user-initiated data updates

##### `handlePageChange(page)`
Pagination handler that updates current page number

#### Lifecycle Hooks
- `onMounted()`: Initializes component, fetches initial data, sets up auto-refresh interval

## Table Column Configuration

The table uses optimized column widths for better data visibility:

| Column | Width | Min-Width | Purpose |
|--------|-------|-----------|---------|
| TIMESTAMP | 15% | 140px | When the command was executed |
| AGENT | 8% | 80px | Agent identifier (with green badge) |
| CLIENT | 8% | 80px | Client identifier (with blue badge) |
| DIRECTION | 8% | 70px | INPUT/OUTPUT indicator (with orange badge) |
| **DATA** | **61%** | **400px** | **SSH command or output content** |

### Special DATA Column Features

The DATA column has been specially optimized:
- **Largest column width** (61% of table width)
- **No content truncation** - displays full command/output text
- **Monospace font** (Courier New) for better code readability
- **Enhanced styling** with background highlighting
- **Pre-formatted text** with proper line breaks
- **Word wrapping** for long commands

## Styling Features

### CSS Classes

- `.ssh-logs-table-container`: Main container with flexbox layout
- `.table-header`: Header section with title and refresh button
- `.loading-state`, `.error-state`, `.empty-state`: Different UI states
- `.table-wrapper`: Table container with pagination
- `.data-cell`: Special styling for DATA column
- `.data-content`: Content wrapper with text formatting

### Responsive Design
- Flexible table layout that adapts to screen size
- Minimum widths prevent columns from becoming too narrow
- Scroll functionality for table overflow

## API Integration

### Endpoint
- **URL**: `${VITE_API_BASE_URL}/api/ssh-logs`
- **Method**: GET
- **Authentication**: Uses JWT token from localStorage

### Data Format Expected
The component expects SSH log objects with fields like:
```javascript
{
  id: "unique-identifier",
  timestamp: "2025-09-21T10:16:16Z",
  agent: "AGENT1",
  client: "E8A580BCFA35558E", 
  direction: "INPUT|OUTPUT",
  data: "command or output content"
}
```

### Error Handling
- **401 Unauthorized**: Shows authentication required message
- **404 Not Found**: Indicates API endpoint configuration issue
- **Network Errors**: Connection failure to relay server
- **Invalid Data**: Handles malformed API responses

## Auto-Refresh Mechanism

- **Interval**: 15 seconds
- **Behavior**: Automatically fetches fresh data from API
- **Console Logging**: Detailed logs for debugging API calls
- **Error Resilient**: Continues auto-refresh even after errors

## Usage Example

```vue
<template>
  <SSHLogsTable />
</template>

<script>
import SSHLogsTable from './components/SSHLogsTable.vue'

export default {
  components: {
    SSHLogsTable
  }
}
</script>
```

## Dependencies

- **Vue 3**: Composition API with reactive references
- **API Service**: Custom API service for HTTP calls (`../config/api.js`)
- **Pagination Component**: Custom pagination component (`./Pagination.vue`)
- **Font Awesome**: Icons for UI elements

## Environment Variables

- `VITE_API_BASE_URL`: Base URL for API calls (defaults to `http://localhost:8080`)

## Development Notes

### Debugging
The component includes extensive console logging for debugging:
- API call details and responses
- Data transformation process
- Error scenarios and handling
- Auto-refresh cycles

### Data Integrity
- **No fallback data**: Component only shows real API data
- **Validation**: Strict data type and structure validation
- **Clear error states**: Explicit error messages for different failure scenarios

### Performance Considerations
- **Pagination**: Limits DOM elements for large datasets
- **Efficient updates**: Only re-renders changed data
- **Memory management**: Clears data before fresh fetches

## Future Enhancements

Potential improvements could include:
- Real-time WebSocket updates instead of polling
- Advanced filtering and search capabilities
- Export functionality for log data
- Detailed log view modal
- Column sorting and customization
- Dark/light theme support