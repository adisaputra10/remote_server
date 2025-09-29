<template>
  <div class="project-management-container">
    <!-- Project Management Header -->
    <div class="table-header">
      <h2 class="table-title">
        <span v-if="!showProjectDashboard">Project Management</span>
        <span v-else>
          <i class="fas fa-arrow-left" @click="closeProjectDashboard" style="cursor: pointer; margin-right: 0.5rem;"></i>
          {{ selectedProject?.project_name || 'Project' }} Dashboard
        </span>
      </h2>
      <div class="table-actions">
        <button 
          v-if="!showProjectDashboard && isAdmin"
          class="btn btn-success" 
          @click="openAddProjectModal">
          <i class="fas fa-plus"></i>
          Add Project
        </button>
        <button 
          class="btn btn-primary" 
          @click="refreshData">
          <i class="fas fa-sync-alt"></i>
          Refresh
        </button>
      </div>
    </div>
    
    <div v-if="loading" class="loading-state">
      <i class="fas fa-spinner fa-spin"></i>
      Loading projects...
    </div>
    
    <div v-else-if="error" class="error-state">
      <i class="fas fa-exclamation-triangle"></i>
      {{ error }}
    </div>
    
    <div v-else-if="allProjects.length === 0 && isAdmin" class="empty-state">
      <i class="fas fa-folder"></i>
      No projects found
    </div>
    
    <div v-else-if="allProjects.length === 0 && !isAdmin" class="no-projects-message">
      <i class="fas fa-info-circle"></i>
      You don't have any projects assigned yet.
    </div>
    
    <!-- Project Dashboard Section -->
    <div v-else-if="showProjectDashboard" class="project-dashboard">
      <div v-if="isAdmin" class="dashboard-section">
        <h3>Available Agents</h3>
        
        <div v-if="dashboardAgents.length === 0" class="empty-message">
          No agents available in this project
        </div>
        
        <div v-else class="dashboard-table-container">
          <table class="dashboard-table">
            <thead>
              <tr>
                <th><i class="fas fa-server"></i> Agent ID</th>
                <th><i class="fas fa-key"></i> Access Type</th>
                <th><i class="fas fa-cogs"></i> Actions</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="agent in dashboardAgents" :key="agent.agent_id" class="agent-row">
                <td class="agent-id-cell">
                  <div class="agent-id-info">
                    <i class="fas fa-server agent-icon"></i>
                    <span class="agent-name">{{ agent.agent_id }}</span>
                  </div>
                </td>
                <td class="access-type-cell">
                  <span :class="['access-badge', 'access-' + getAccessTypeBadgeClass(agent.access_type)]">
                    <i :class="getAccessTypeIcon(agent.access_type)"></i>
                    {{ formatAccessType(agent.access_type) }}
                  </span>
                </td>
                <td class="actions-cell">
                  <div class="action-buttons">
                    <button 
                      v-if="agent.access_type === 'ssh' || agent.access_type === 'both'"
                      class="action-btn ssh-btn"
                      @click="connectSSH(agent.agent_id)"
                      :disabled="!isAgentOnline(agent.agent_id)"
                      title="SSH Connect">
                      <i class="fas fa-terminal"></i>
                      <span>SSH</span>
                    </button>
                    
                    <button 
                      v-if="agent.access_type === 'database' || agent.access_type === 'both'"
                      class="action-btn tunnel-btn"
                      @click="openDatabaseTunnel(agent.agent_id)"
                      :disabled="!isAgentOnline(agent.agent_id)"
                      title="Database Tunnel">
                      <i class="fas fa-database"></i>
                      <span>Tunnel</span>
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
      
      <!-- User Access Section -->
      <div v-if="!isAdmin" class="dashboard-section">
        <h3>{{ selectedProject.project_name }} Project</h3>
        
        <div v-if="dashboardAgents.length === 0" class="user-project-message">
          <i class="fas fa-info-circle"></i>
          <p>You have access to this project, but no agents are currently available. Contact your administrator for agent access.</p>
        </div>
        
        <div v-else class="available-agents">
          <h4>Available Connections</h4>
          <div class="agents-grid">
            <div v-for="agent in dashboardAgents" :key="agent.agent_id" class="agent-card">
              <div class="agent-header">
                <h5>{{ agent.agent_id }}</h5>
                <span class="status" :class="agent.status">{{ agent.status }}</span>
              </div>
              <div class="agent-details">
                <p><strong>Access Type:</strong> {{ agent.access_type }}</p>
                <div v-if="agent.status === 'connected'" class="agent-actions">
                  <button v-if="agent.access_type === 'ssh' || agent.access_type === 'both'" 
                          class="btn-action ssh" 
                          @click="connectSSH(agent.agent_id)">
                    <i class="fas fa-terminal"></i> SSH
                  </button>
                  <button v-if="agent.access_type === 'database' || agent.access_type === 'both'" 
                          class="btn-action tunnel" 
                          @click="openDatabaseTunnel(agent.agent_id)">
                    <i class="fas fa-database"></i> Database
                  </button>
                </div>
                <div v-else class="agent-offline">
                  <i class="fas fa-exclamation-triangle"></i> Agent offline
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
      
      <!-- Active Connections Section -->
      <div v-if="isAdmin" class="dashboard-section">
        <h3>Active Connections</h3>
        
        <div v-if="activeConnections.length === 0" class="empty-message">
          No active connections
        </div>
        
        <div v-else class="dashboard-table-container">
          <table class="dashboard-table">
            <thead>
              <tr>
                <th><i class="fas fa-server"></i> Agent ID</th>
                <th><i class="fas fa-plug"></i> Connection Type</th>
                <th><i class="fas fa-clock"></i> Duration</th>
                <th><i class="fas fa-cogs"></i> Actions</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="connection in activeConnections" :key="connection.id" class="connection-row">
                <td class="agent-id-cell">
                  <div class="agent-id-info">
                    <i class="fas fa-server agent-icon"></i>
                    <span class="agent-name">{{ connection.agent_id }}</span>
                  </div>
                </td>
                <td class="connection-type-cell">
                  <span :class="['connection-type-badge', connection.type === 'ssh' ? 'type-ssh' : 'type-database']">
                    <i :class="['fas', connection.type === 'ssh' ? 'fa-terminal' : 'fa-database']"></i>
                    {{ connection.type.toUpperCase() }}
                  </span>
                </td>
                <td class="duration-cell">
                  <span class="duration-text">
                    <i class="fas fa-clock"></i>
                    {{ formatConnectionTime(connection.started_at) }}
                  </span>
                </td>
                <td class="actions-cell">
                  <button 
                    class="action-btn disconnect-btn"
                    @click="disconnectSession(connection.id)"
                    title="Disconnect">
                    <i class="fas fa-times"></i>
                    Disconnect
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
    
    <div v-else-if="allProjects.length > 0 || isAdmin" class="table-wrapper">
      <div class="table-container">
        <table class="table">
          <thead>
            <tr>
              <th>ID</th>
              <th>PROJECT NAME</th>
              <th>DESCRIPTION</th>
              <th>STATUS</th>
              <th v-if="isAdmin">USERS</th>
              <th v-if="isAdmin">AGENTS</th>
              <th>CREATED</th>
              <th>ACTIONS</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="project in paginatedProjects" :key="project.id">
              <td>{{ project.id }}</td>
              <td>
                <div class="project-name">
                  <i class="fas fa-folder"></i>
                  {{ project.project_name }}
                </div>
              </td>
              <td>{{ project.description || '-' }}</td>
              <td>
                <span :class="['badge', 'badge-' + getStatusBadgeClass(project.status)]">
                  {{ project.status?.toUpperCase() || 'UNKNOWN' }}
                </span>
              </td>
              <td v-if="isAdmin">
                <div class="counter-cell">
                  <span class="counter">{{ project.user_count || 0 }}</span>
                  <button class="btn-mini" @click="openManageUsersModal(project)">
                    <i class="fas fa-users"></i>
                  </button>
                </div>
              </td>
              <td v-if="isAdmin">
                <div class="counter-cell">
                  <span class="counter">{{ project.agent_count || 0 }}</span>
                  <button class="btn-mini" @click="openManageAgentsModal(project)">
                    <i class="fas fa-server"></i>
                  </button>
                </div>
              </td>
              <td>{{ formatDateTime(project.created_at) }}</td>
              <td>
                <div class="action-buttons">
                  <button 
                    class="action-btn enter-btn" 
                    @click="enterProject(project)"
                    title="Enter Project">
                    <i class="fas fa-sign-in-alt"></i>
                  </button>
                  <button 
                    v-if="isAdmin"
                    class="action-btn edit-btn" 
                    @click="openEditProjectModal(project)"
                    title="Edit Project">
                    <i class="fas fa-edit"></i>
                  </button>
                  <button 
                    v-if="isAdmin"
                    class="action-btn users-btn" 
                    @click="openManageUsersModal(project)"
                    title="Manage Users">
                    <i class="fas fa-users"></i>
                  </button>
                  <button 
                    v-if="isAdmin"
                    class="action-btn agents-btn" 
                    @click="openManageAgentsModal(project)"
                    title="Manage Agents">
                    <i class="fas fa-server"></i>
                  </button>
                  <button 
                    v-if="isAdmin && project.project_name !== 'Default Project'"
                    class="action-btn delete-btn" 
                    @click="showDelete(project.id, project.project_name)"
                    title="Delete Project">
                    <i class="fas fa-trash"></i>
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      
      <Pagination
        :current-page="currentPage"
        :total-items="allProjects.length"
        :items-per-page="itemsPerPage"
        @page-changed="handlePageChange"
      />
    </div>

    <!-- Add Project Modal -->
    <div v-if="showAddModal" class="modal-overlay" @click="closeAddProjectModal">
      <div class="modal" @click.stop>
        <div class="modal-header">
          <h3>
            <i class="fas fa-folder-plus"></i>
            Add New Project
          </h3>
          <button class="btn-close" @click="closeAddProjectModal">
            <i class="fas fa-times"></i>
          </button>
        </div>
        
        <div class="modal-body">
          <div class="form-group">
            <label for="project-name">Project Name *</label>
            <input
              id="project-name"
              v-model="newProject.project_name"
              type="text"
              class="form-input"
              placeholder="Enter project name"
              required
            />
            <small class="form-help">Choose a unique and descriptive name</small>
          </div>

          <div class="form-group">
            <label for="project-description">Description</label>
            <textarea
              id="project-description"
              v-model="newProject.description"
              class="form-input"
              rows="3"
              placeholder="Enter project description"
            ></textarea>
            <small class="form-help">Optional description for the project</small>
          </div>

          <div class="form-group">
            <label for="project-status">Status *</label>
            <select
              id="project-status"
              v-model="newProject.status"
              class="form-input"
              required
            >
              <option value="active">Active</option>
              <option value="inactive">Inactive</option>
            </select>
            <small class="form-help">Project status determines accessibility</small>
          </div>
        </div>
        
        <div class="modal-footer">
          <button type="button" class="btn btn-secondary" @click="closeAddProjectModal">
            Cancel
          </button>
          <button 
            type="button" 
            class="btn btn-success" 
            @click="submitProject"
            :disabled="submitting || !newProject.project_name">
            <i class="fas fa-folder-plus" v-if="!submitting"></i>
            <i class="fas fa-spinner fa-spin" v-else></i>
            {{ submitting ? 'Creating...' : 'Create Project' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Edit Project Modal -->
    <div v-if="showEditModal" class="modal-overlay" @click="closeEditProjectModal">
      <div class="modal" @click.stop>
        <div class="modal-header">
          <h3>
            <i class="fas fa-folder-open"></i>
            Edit Project
          </h3>
          <button class="btn-close" @click="closeEditProjectModal">
            <i class="fas fa-times"></i>
          </button>
        </div>
        
        <div class="modal-body">
          <div class="form-group">
            <label for="edit-project-name">Project Name *</label>
            <input
              id="edit-project-name"
              v-model="editProject.project_name"
              type="text"
              class="form-input"
              placeholder="Enter project name"
              required
            />
          </div>

          <div class="form-group">
            <label for="edit-project-description">Description</label>
            <textarea
              id="edit-project-description"
              v-model="editProject.description"
              class="form-input"
              rows="3"
              placeholder="Enter project description"
            ></textarea>
          </div>

          <div class="form-group">
            <label for="edit-project-status">Status *</label>
            <select
              id="edit-project-status"
              v-model="editProject.status"
              class="form-input"
              required
            >
              <option value="active">Active</option>
              <option value="inactive">Inactive</option>
            </select>
          </div>
        </div>
        
        <div class="modal-footer">
          <button type="button" class="btn btn-secondary" @click="closeEditProjectModal">
            Cancel
          </button>
          <button 
            type="button" 
            class="btn btn-primary" 
            @click="submitEditProject"
            :disabled="submitting || !editProject.project_name">
            <i class="fas fa-save" v-if="!submitting"></i>
            <i class="fas fa-spinner fa-spin" v-else></i>
            {{ submitting ? 'Saving...' : 'Save Changes' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Manage Users Modal -->
    <div v-if="showUsersModal" class="modal-overlay" @click="closeManageUsersModal">
      <div class="modal modal-large" @click.stop>
        <div class="modal-header">
          <h3>
            <i class="fas fa-users"></i>
            Manage Users - {{ selectedProject?.project_name }}
          </h3>
          <button class="btn-close" @click="closeManageUsersModal">
            <i class="fas fa-times"></i>
          </button>
        </div>
        
        <div class="modal-body">
          <!-- Add User Section -->
          <div class="section">
            <h4>Add User to Project</h4>
            <div class="add-user-form">
              <div class="form-row">
                <select v-model="selectedUserId" class="form-input">
                  <option value="">Select User</option>
                  <option 
                    v-for="user in availableUsers" 
                    :key="user.id" 
                    :value="user.id">
                    {{ user.username }} ({{ user.role }})
                  </option>
                </select>
                <select v-model="selectedUserRole" class="form-input">
                  <option value="admin">Admin</option>
                  <option value="user">User</option>
                </select>
                <button 
                  class="btn btn-success" 
                  @click="addUserToProject"
                  :disabled="!selectedUserId">
                  <i class="fas fa-plus"></i>
                  Add
                </button>
              </div>
            </div>
          </div>

          <!-- Current Users Section -->
          <div class="section">
            <h4>Current Users</h4>
            <div class="users-list">
              <div 
                v-for="projectUser in currentProjectUsers" 
                :key="projectUser.user_id"
                class="user-item">
                <div class="user-info">
                  <i class="fas fa-user"></i>
                  <span class="username">{{ projectUser.username }}</span>
                  <span :class="['badge', 'badge-' + getRoleBadgeClass(projectUser.role)]">
                    {{ projectUser.role?.toUpperCase() }}
                  </span>
                </div>
                <div class="user-actions">
                  <select 
                    v-model="projectUser.role" 
                    @change="updateUserRole(projectUser.user_id, projectUser.role)"
                    class="form-input-small">
                    <option value="admin">Admin</option>
                    <option value="user">User</option>
                  </select>
                  <button 
                    class="btn-danger-small" 
                    @click="removeUserFromProject(projectUser.user_id)"
                    title="Remove User">
                    <i class="fas fa-times"></i>
                  </button>
                </div>
              </div>
              <div v-if="currentProjectUsers.length === 0" class="empty-message">
                No users assigned to this project
              </div>
            </div>
          </div>
        </div>
        
        <div class="modal-footer">
          <button type="button" class="btn btn-secondary" @click="closeManageUsersModal">
            Close
          </button>
        </div>
      </div>
    </div>

    <!-- Manage Agents Modal -->
    <div v-if="showAgentsModal" class="modal-overlay" @click="closeManageAgentsModal">
      <div class="modal modal-large" @click.stop>
        <div class="modal-header">
          <h3>
            <i class="fas fa-server"></i>
            Manage Agents - {{ selectedProject?.project_name }}
          </h3>
          <button class="btn-close" @click="closeManageAgentsModal">
            <i class="fas fa-times"></i>
          </button>
        </div>
        
        <div class="modal-body">
          <!-- Add Agent Section -->
          <div class="section">
            <h4>Add Agent to Project</h4>
            <div class="add-agent-form">
              <div class="form-row">
                <select v-model="selectedAgentId" class="form-input">
                  <option value="">Select Agent</option>
                  <option 
                    v-for="agent in availableAgents" 
                    :key="agent.agent_id" 
                    :value="agent.agent_id">
                    {{ agent.agent_id }} - {{ agent.name || 'Unknown' }}
                  </option>
                </select>
                <select v-model="selectedAccessType" class="form-input">
                  <option value="both">SSH + Database</option>
                  <option value="ssh">SSH Only</option>
                  <option value="database">Database Only</option>
                </select>
                <button 
                  class="btn btn-success" 
                  @click="addAgentToProject"
                  :disabled="!selectedAgentId">
                  <i class="fas fa-plus"></i>
                  Add
                </button>
              </div>
            </div>
          </div>

          <!-- Current Agents Section -->
          <div class="section">
            <h4>Current Agents</h4>
            <div class="agents-list">
              <div 
                v-for="projectAgent in currentProjectAgents" 
                :key="projectAgent.agent_id"
                class="agent-item">
                <div class="agent-info">
                  <i class="fas fa-server"></i>
                  <span class="agent-name">{{ projectAgent.agent_id }}</span>
                  <span :class="['badge', 'badge-' + getAccessTypeBadgeClass(projectAgent.access_type)]">
                    {{ formatAccessType(projectAgent.access_type) }}
                  </span>
                </div>
                <div class="agent-actions">
                  <select 
                    v-model="projectAgent.access_type" 
                    @change="updateAgentAccess(projectAgent.agent_id, projectAgent.access_type)"
                    class="form-input-small">
                    <option value="both">SSH + Database</option>
                    <option value="ssh">SSH Only</option>
                    <option value="database">Database Only</option>
                  </select>
                  <button 
                    class="btn-danger-small" 
                    @click="removeAgentFromProject(projectAgent.agent_id)"
                    title="Remove Agent">
                    <i class="fas fa-times"></i>
                  </button>
                </div>
              </div>
              <div v-if="currentProjectAgents.length === 0" class="empty-message">
                No agents assigned to this project
              </div>
            </div>
          </div>
        </div>
        
        <div class="modal-footer">
          <button type="button" class="btn btn-secondary" @click="closeManageAgentsModal">
            Close
          </button>
        </div>
      </div>
    </div>

    <!-- Delete Project Confirmation Modal -->
    <div v-if="showDeleteModal" class="modal-overlay" @click="closeDeleteModal">
      <div class="modal delete-modal" @click.stop>
        <div class="modal-header">
          <h3>
            <i class="fas fa-exclamation-triangle text-warning"></i>
            Confirm Delete
          </h3>
          <button class="btn-close" @click="closeDeleteModal">
            <i class="fas fa-times"></i>
          </button>
        </div>
        
        <div class="modal-body">
          <p class="delete-warning">
            Are you sure you want to delete project <strong>{{ projectToDelete.name }}</strong>?
          </p>
          <p class="delete-note">
            This action cannot be undone. All user and agent assignments will be removed.
          </p>
        </div>
        
        <div class="modal-footer">
          <button type="button" class="btn btn-secondary" @click="closeDeleteModal">
            Cancel
          </button>
          <button type="button" class="btn btn-danger" @click="deleteProject">
            <i class="fas fa-trash"></i>
            Delete Project
          </button>
        </div>
      </div>
    </div>
  </div>

  <!-- Access Options Modal -->
  <div v-if="showAccessOptions" class="modal-overlay" @click="closeAccessOptions">
    <div class="modal modal-large access-options-modal" @click.stop>
      <div class="modal-header">
        <h3>
          <i :class="selectedModalAccessType === 'ssh' ? 'fas fa-terminal' : 'fas fa-database'"></i>
          {{ modalTitle }} for Agent: {{ selectedAgentForAccess }}
        </h3>
        <button class="btn-close" @click="closeAccessOptions">×</button>
      </div>

      <div class="modal-body">
        <div class="access-options-table">
          <div class="access-option-header">
            <div class="header-col service-col">SERVICE TYPE</div>
            <div class="header-col command-col">COMMAND</div>
            <div class="header-col action-col">ACTION</div>
          </div>

          <!-- SSH Access - Only show if SSH is selected -->
          <div v-if="selectedModalAccessType === 'ssh'" class="access-option-row">
            <div class="service-type">
              <i class="fas fa-terminal ssh-icon"></i>
              <span>SSH Access</span>
            </div>
            <div class="command-text">
              {{ getAccessCommands(selectedAgentForAccess).ssh }}
            </div>
            <div class="action-copy">
              <button 
                class="copy-btn"
                @click="copyToClipboard(getAccessCommands(selectedAgentForAccess).ssh)"
                title="Copy command">
                <i class="fas fa-copy"></i>
              </button>
            </div>
          </div>

          <!-- Tunnel Commands - Only show if Tunnel is selected -->
          <template v-if="selectedModalAccessType === 'tunnel'">
            <!-- MySQL Tunnel -->
            <div class="access-option-row">
              <div class="service-type">
                <i class="fas fa-database mysql-icon"></i>
                <span>MySQL Tunnel</span>
              </div>
              <div class="command-text">
                {{ getAccessCommands(selectedAgentForAccess).mysqlTunnel }}
              </div>
              <div class="action-copy">
                <button 
                  class="copy-btn"
                  @click="copyToClipboard(getAccessCommands(selectedAgentForAccess).mysqlTunnel)"
                  title="Copy command">
                  <i class="fas fa-copy"></i>
                </button>
              </div>
            </div>

            <!-- PostgreSQL Tunnel -->
            <div class="access-option-row">
              <div class="service-type">
                <i class="fas fa-database postgres-icon"></i>
                <span>PostgreSQL Tunnel</span>
              </div>
              <div class="command-text">
                {{ getAccessCommands(selectedAgentForAccess).postgresqlTunnel }}
              </div>
              <div class="action-copy">
                <button 
                  class="copy-btn"
                  @click="copyToClipboard(getAccessCommands(selectedAgentForAccess).postgresqlTunnel)"
                  title="Copy command">
                  <i class="fas fa-copy"></i>
                </button>
              </div>
            </div>

            <!-- MongoDB Tunnel -->
            <div class="access-option-row">
              <div class="service-type">
                <i class="fas fa-database mongodb-icon"></i>
                <span>MongoDB Tunnel</span>
              </div>
              <div class="command-text">
                {{ getAccessCommands(selectedAgentForAccess).mongodbTunnel }}
              </div>
              <div class="action-copy">
                <button 
                  class="copy-btn"
                  @click="copyToClipboard(getAccessCommands(selectedAgentForAccess).mongodbTunnel)"
                  title="Copy command">
                  <i class="fas fa-copy"></i>
                </button>
              </div>
            </div>

            <!-- Custom Tunnel -->
            <div class="access-option-row custom-row">
              <div class="service-type">
                <i class="fas fa-cog custom-icon"></i>
                <span>Custom Tunnel</span>
              </div>
              <div class="custom-inputs">
                <div class="input-group">
                  <input 
                    type="text" 
                    v-model="customLocalPort" 
                    placeholder="9999"
                    class="custom-input port-input"
                  />
                  <input 
                    type="text" 
                    v-model="customRemoteHost" 
                    placeholder="localhost:80"
                    class="custom-input host-input"
                  />
                </div>
                <div class="command-text">
                  {{ getAccessCommands(selectedAgentForAccess).customTunnel(customLocalPort, customRemoteHost) }}
                </div>
              </div>
              <div class="action-copy">
                <button 
                  class="copy-btn"
                  @click="copyToClipboard(getAccessCommands(selectedAgentForAccess).customTunnel(customLocalPort, customRemoteHost))"
                  title="Copy command">
                  <i class="fas fa-copy"></i>
                </button>
              </div>
            </div>
          </template>
        </div>

        <!-- Note -->
        <div class="access-note">
          <i class="fas fa-info-circle"></i>
          <div class="note-content">
            <strong>Note:</strong> Make sure to use your user token (-T parameter) when executing these commands. 
            Your token: <code>{{ isAdmin ? 'admin_token_2025_secure' : 'user_token_2025_access' }}</code>
          </div>
        </div>
      </div>

      <div class="modal-footer">
        <button type="button" class="btn btn-secondary" @click="closeAccessOptions">
          Close
        </button>
      </div>
    </div>
  </div>
</template>

<script>
import { ref, onMounted, computed } from 'vue'
import { apiService } from '../config/api.js'
import { currentUser, isAdmin, isUser } from '../utils/auth.js'
import Pagination from './Pagination.vue'

export default {
  name: 'ProjectManagement',
  components: {
    Pagination
  },
  setup() {
    // Main data
    const allProjects = ref([])
    const allUsers = ref([])
    const allAgents = ref([])
    const projectUsers = ref([])
    const projectAgents = ref([])
    const loading = ref(false)
    const error = ref(null)
    const currentPage = ref(1)
    const itemsPerPage = ref(10)

    // Modal states
    const showAddModal = ref(false)
    const showEditModal = ref(false)
    const showUsersModal = ref(false)
    const showAgentsModal = ref(false)
    const showDeleteModal = ref(false)
    const submitting = ref(false)

    // Form data
    const newProject = ref({
      project_name: '',
      description: '',
      status: 'active'
    })

    const editProject = ref({
      id: null,
      project_name: '',
      description: '',
      status: 'active'
    })

    // Selected items
    const selectedProject = ref(null)
    const selectedUserId = ref('')
    const selectedUserRole = ref('user')
    const selectedAgentId = ref('')
    const selectedAccessType = ref('both')
    const projectToDelete = ref({})

    // Project Dashboard data
    const showProjectDashboard = ref(false)
    const showAccessOptions = ref(false)
    const selectedAgentForAccess = ref(null)
    const selectedModalAccessType = ref(null) // 'ssh' or 'tunnel'
    const customLocalPort = ref('9999')
    const customRemoteHost = ref('localhost:80')
    const dashboardAgents = ref([])
    const activeConnections = ref([])

    // Computed properties
    const paginatedProjects = computed(() => {
      const start = (currentPage.value - 1) * itemsPerPage.value
      const end = start + itemsPerPage.value
      return allProjects.value.slice(start, end)
    })

    // Project-specific data (loaded when managing specific project)
    const currentProjectUsers = ref([])
    const currentProjectAgents = ref([])

    const availableUsers = computed(() => {
      const assignedUserIds = currentProjectUsers.value.map(pu => pu.user_id)
      return allUsers.value.filter(user => !assignedUserIds.includes(user.id))
    })

    const availableAgents = computed(() => {
      const assignedAgentIds = currentProjectAgents.value.map(pa => pa.agent_id)
      return allAgents.value.filter(agent => !assignedAgentIds.includes(agent.agent_id))
    })

    // Data fetching functions
    const fetchProjects = async () => {
      try {
        loading.value = true
        error.value = null
        
        console.log('=== FETCHING PROJECTS FROM API ===')
        console.log('Current user:', currentUser.value)
        console.log('Is Admin:', isAdmin.value)
        console.log('User role from currentUser:', currentUser.value?.role)
        console.log('Auth token from localStorage:', localStorage.getItem('auth_token'))
        
        let response
        if (isAdmin.value) {
          // Admin can see all projects
          console.log('Admin user - fetching all projects')
          response = await apiService.getProjects()
        } else {
          // Regular user - only see assigned projects
          console.log('Regular user - fetching user-assigned projects')
          console.log('Calling apiService.getUserProjects()...')
          response = await apiService.getUserProjects()
          console.log('Raw response from getUserProjects:', response)
        }
        
        console.log('Projects API response:', response.data)
        console.log('Setting allProjects to:', response.data || [])
        allProjects.value = response.data || []
        console.log('allProjects.value after assignment:', allProjects.value)
        console.log('allProjects.value.length:', allProjects.value.length)
        
      } catch (err) {
        console.error('Error fetching projects:', err)
        console.error('Error details:', {
          message: err.message,
          response: err.response,
          status: err.response?.status,
          data: err.response?.data
        })
        error.value = 'Failed to load projects data'
        allProjects.value = []
      } finally {
        loading.value = false
      }
    }

    const fetchUsers = async () => {
      try {
        const response = await apiService.getUsers()
        allUsers.value = response.data || []
      } catch (err) {
        console.error('Error fetching users:', err)
      }
    }

    const fetchAgents = async () => {
      try {
        const response = await apiService.getAgents()
        allAgents.value = response.data || []
      } catch (err) {
        console.error('Error fetching agents:', err)
      }
    }

    const fetchProjectUsers = async (projectId = null) => {
      if (!projectId) return
      try {
        const response = await apiService.getProjectUsers(projectId)
        currentProjectUsers.value = response.data || []
      } catch (err) {
        console.error('Error fetching project users:', err)
        currentProjectUsers.value = []
      }
    }

    const fetchProjectAgents = async (projectId = null) => {
      if (!projectId) return
      try {
        const response = await apiService.getProjectAgents(projectId)
        currentProjectAgents.value = response.data || []
      } catch (err) {
        console.error('Error fetching project agents:', err)
        currentProjectAgents.value = []
      }
    }

    const refreshData = async () => {
      console.log('Refreshing all project data...')
      await Promise.all([
        fetchProjects(),
        fetchUsers(),
        fetchAgents()
      ])
      
      // Reset project-specific data when refreshing general data
      currentProjectUsers.value = []
      currentProjectAgents.value = []
    }

    // Utility functions
    const handlePageChange = (page) => {
      currentPage.value = page
    }

    const getStatusBadgeClass = (status) => {
      switch (status) {
        case 'active': return 'success'
        case 'inactive': return 'warning'
        default: return 'secondary'
      }
    }

    const getRoleBadgeClass = (role) => {
      switch (role) {
        case 'admin': return 'danger'
        case 'member': return 'primary'
        case 'viewer': return 'secondary'
        default: return 'secondary'
      }
    }

    const formatDateTime = (dateString) => {
      if (!dateString) return 'Unknown'
      const date = new Date(dateString)
      return date.toLocaleString('en-US', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit'
      })
    }

    const getProjectUserCount = (projectId) => {
      return projectUsers.value.filter(pu => pu.project_id === projectId).length
    }

    const getProjectAgentCount = (projectId) => {
      return projectAgents.value.filter(pa => pa.project_id === projectId).length
    }

    const getUsernameById = (userId) => {
      const user = allUsers.value.find(u => u.id === userId)
      return user ? user.username : 'Unknown'
    }

    // Project CRUD functions
    const openAddProjectModal = () => {
      newProject.value = {
        project_name: '',
        description: '',
        status: 'active'
      }
      showAddModal.value = true
    }

    const closeAddProjectModal = () => {
      showAddModal.value = false
      submitting.value = false
    }

    const submitProject = async () => {
      if (!newProject.value.project_name) {
        alert('Project name is required')
        return
      }
      
      submitting.value = true
      
      try {
        const response = await apiService.addProject(newProject.value)
        console.log('Project added successfully:', response.data)
        
        alert(`Project "${newProject.value.project_name}" created successfully!`)
        closeAddProjectModal()
        refreshData()
        
      } catch (error) {
        console.error('Error adding project:', error)
        alert('Failed to add project: ' + (error.response?.data?.error || error.message))
      } finally {
        submitting.value = false
      }
    }

    const openEditProjectModal = (project) => {
      editProject.value = {
        id: project.id,
        project_name: project.name || project.project_name || '',
        description: project.description || '',
        status: project.status || 'active'
      }
      showEditModal.value = true
    }

    const closeEditProjectModal = () => {
      showEditModal.value = false
      submitting.value = false
    }

    const submitEditProject = async () => {
      if (!editProject.value.project_name) {
        alert('Project name is required')
        return
      }
      
      submitting.value = true
      
      try {
        // Transform data to match backend API expectations
        const updateData = {
          name: editProject.value.project_name,
          description: editProject.value.description,
          status: editProject.value.status
        }
        
        const response = await apiService.updateProject(editProject.value.id, updateData)
        console.log('Project updated successfully:', response.data)
        
        alert(`Project "${editProject.value.project_name}" updated successfully!`)
        closeEditProjectModal()
        refreshData()
        
      } catch (error) {
        console.error('Error updating project:', error)
        alert('Failed to update project: ' + (error.response?.data?.error || error.message))
      } finally {
        submitting.value = false
      }
    }

    // User management functions
    const openManageUsersModal = async (project) => {
      selectedProject.value = project
      selectedUserId.value = ''
      selectedUserRole.value = 'user'
      showUsersModal.value = true
      
      // Load users for this specific project
      await fetchProjectUsers(project.id)
    }

    const closeManageUsersModal = () => {
      showUsersModal.value = false
      selectedProject.value = null
    }

    const addUserToProject = async () => {
      if (!selectedUserId.value) return
      
      try {
        // Find the username from the selected user ID
        const selectedUser = availableUsers.value.find(u => u.id === parseInt(selectedUserId.value))
        if (!selectedUser) {
          alert('Selected user not found')
          return
        }
        
        await apiService.addUserToProject(selectedProject.value.id, {
          username: selectedUser.username,
          role: selectedUserRole.value
        })
        
        selectedUserId.value = ''
        selectedUserRole.value = 'user'
        await fetchProjectUsers(selectedProject.value.id)
        
        alert('User added to project successfully!')
        
      } catch (error) {
        console.error('Error adding user to project:', error)
        alert('Failed to add user: ' + (error.response?.data?.error || error.message))
      }
    }

    const updateUserRole = async (userId, newRole) => {
      try {
        await apiService.updateProjectUserRole(selectedProject.value.id, userId, newRole)
        
        await fetchProjectUsers(selectedProject.value.id)
        alert('User role updated successfully!')
        
      } catch (error) {
        console.error('Error updating user role:', error)
        alert('Failed to update user role: ' + (error.response?.data?.error || error.message))
      }
    }

    const removeUserFromProject = async (userId) => {
      if (confirm('Are you sure you want to remove this user from the project?')) {
        try {
          await apiService.removeUserFromProject(selectedProject.value.id, userId)
          await fetchProjectUsers(selectedProject.value.id)
          alert('User removed from project successfully!')
          
        } catch (error) {
          console.error('Error removing user from project:', error)
          alert('Failed to remove user: ' + (error.response?.data?.error || error.message))
        }
      }
    }

    // Agent management functions
    const openManageAgentsModal = async (project) => {
      selectedProject.value = project
      selectedAgentId.value = ''
      selectedAccessType.value = 'both'
      showAgentsModal.value = true
      
      // Load agents for this specific project
      await fetchProjectAgents(project.id)
    }

    const closeManageAgentsModal = () => {
      showAgentsModal.value = false
      selectedProject.value = null
    }

    const addAgentToProject = async () => {
      if (!selectedAgentId.value) return
      
      try {
        await apiService.addAgentToProject(selectedProject.value.id, {
          agent_id: selectedAgentId.value
        })
        
        selectedAgentId.value = ''
        selectedAccessType.value = 'both'
        await fetchProjectAgents(selectedProject.value.id)
        
        alert('Agent added to project successfully!')
        
      } catch (error) {
        console.error('Error adding agent to project:', error)
        alert('Failed to add agent: ' + (error.response?.data?.error || error.message))
      }
    }

    const updateAgentAccess = async (agentId, newAccessType) => {
      try {
        await apiService.updateProjectAgent(selectedProject.value.id, agentId, {
          access_type: newAccessType
        })
        
        await fetchProjectAgents(selectedProject.value.id)
        alert('Agent access updated successfully!')
        
      } catch (error) {
        console.error('Error updating agent access:', error)
        alert('Failed to update agent access: ' + (error.response?.data?.error || error.message))
      }
    }

    const removeAgentFromProject = async (agentId) => {
      if (confirm('Are you sure you want to remove this agent from the project?')) {
        try {
          await apiService.removeAgentFromProject(selectedProject.value.id, agentId)
          await fetchProjectAgents(selectedProject.value.id)
          alert('Agent removed from project successfully!')
          
        } catch (error) {
          console.error('Error removing agent from project:', error)
          alert('Failed to remove agent: ' + (error.response?.data?.error || error.message))
        }
      }
    }

    // Project Dashboard functions
    const enterProject = async (project) => {
      selectedProject.value = project
      showProjectDashboard.value = true
      await loadProjectDashboard(project.id)
    }

    const closeProjectDashboard = () => {
      showProjectDashboard.value = false
      selectedProject.value = null
      dashboardAgents.value = []
      activeConnections.value = []
    }

    const loadProjectDashboard = async (projectId) => {
      try {
        loading.value = true
        
        // Load project agents based on user role
        if (isAdmin.value) {
          const agentsResponse = await apiService.getProjectAgents(projectId)
          dashboardAgents.value = agentsResponse.data || []
        } else {
          // For regular users, load agents they have access to
          const agentsResponse = await apiService.getUserProjectAgents(projectId)
          dashboardAgents.value = agentsResponse.data || []
        }
        
        // Load active connections (this would come from a separate API)
        // For now, we'll use mock data
        activeConnections.value = []
        
      } catch (error) {
        console.error('Error loading project dashboard:', error)
        if (!isAdmin.value) {
          // For users, show a simpler error message
          console.log('User accessed project dashboard - some features may be limited')
          dashboardAgents.value = []
        } else {
          alert('Failed to load project dashboard')
        }
      } finally {
        loading.value = false
      }
    }

    const isAgentOnline = (agentId) => {
      // This should check real agent status
      // For now, we'll return true for demonstration
      return allAgents.value.some(agent => agent.agent_id === agentId)
    }

    const getAccessTypeBadgeClass = (accessType) => {
      switch (accessType) {
        case 'ssh': return 'primary'
        case 'database': return 'success'
        case 'both': return 'info'
        default: return 'secondary'
      }
    }

    const getAccessTypeIcon = (accessType) => {
      switch (accessType) {
        case 'ssh': return 'fas fa-terminal'
        case 'database': return 'fas fa-database'
        case 'both': return 'fas fa-server'
        default: return 'fas fa-question'
      }
    }

    const formatAccessType = (accessType) => {
      switch (accessType) {
        case 'ssh': return 'SSH Only'
        case 'database': return 'Database Only'
        case 'both': return 'SSH + Database'
        default: return 'Unknown'
      }
    }

    const connectSSH = async (agentId) => {
      selectedAgentForAccess.value = agentId
      selectedModalAccessType.value = 'ssh'
      showAccessOptions.value = true
    }

    const openDatabaseTunnel = async (agentId) => {
      selectedAgentForAccess.value = agentId
      selectedModalAccessType.value = 'tunnel'
      showAccessOptions.value = true
    }

    const closeAccessOptions = () => {
      showAccessOptions.value = false
      selectedAgentForAccess.value = null
      selectedModalAccessType.value = null
    }

    const copyToClipboard = async (text) => {
      try {
        await navigator.clipboard.writeText(text)
        alert('Command copied to clipboard!')
      } catch (error) {
        console.error('Failed to copy:', error)
        // Fallback for older browsers
        const textArea = document.createElement('textarea')
        textArea.value = text
        document.body.appendChild(textArea)
        textArea.select()
        document.execCommand('copy')
        document.body.removeChild(textArea)
        alert('Command copied to clipboard!')
      }
    }

    const getAccessCommands = (agentId) => {
      // Use appropriate token based on user role
      const token = isAdmin.value ? 'admin_token_2025_secure' : 'user_token_2025_access'
      
      return {
        ssh: `.\\bin\\universal-client.exe -T ${token} -u username -H target-server -a ${agentId}`,
        mysqlTunnel: `.\\bin\\universal-client.exe -T ${token} -L :3306 -t localhost:3306 -a ${agentId}`,
        postgresqlTunnel: `.\\bin\\universal-client.exe -T ${token} -L :5432 -t localhost:5432 -a ${agentId}`,
        mongodbTunnel: `.\\bin\\universal-client.exe -T ${token} -L :27017 -t localhost:27017 -a ${agentId}`,
        customTunnel: (localPort, remoteHost) => `.\\bin\\universal-client.exe -T ${token} -L :${localPort} -t ${remoteHost} -a ${agentId}`
      }
    }

    const disconnectSession = async (connectionId) => {
      try {
        // This would disconnect the session
        activeConnections.value = activeConnections.value.filter(conn => conn.id !== connectionId)
        alert('Session disconnected successfully')
      } catch (error) {
        console.error('Error disconnecting session:', error)
        alert('Failed to disconnect session')
      }
    }

    const formatConnectionTime = (startedAt) => {
      const now = new Date()
      const diff = now - new Date(startedAt)
      const minutes = Math.floor(diff / 60000)
      const seconds = Math.floor((diff % 60000) / 1000)
      return `${minutes}m ${seconds}s`
    }

    // Delete functions
    const showDelete = (projectId, projectName) => {
      projectToDelete.value = { id: projectId, name: projectName }
      showDeleteModal.value = true
    }

    const closeDeleteModal = () => {
      showDeleteModal.value = false
      projectToDelete.value = {}
    }

    const deleteProject = async () => {
      if (!projectToDelete.value.id) return
      
      try {
        await apiService.deleteProject(projectToDelete.value.id)
        await refreshData()
        closeDeleteModal()
        
        alert(`Project ${projectToDelete.value.name} deleted successfully`)
      } catch (error) {
        console.error('Error deleting project:', error)
        alert('Failed to delete project: ' + (error.response?.data?.error || error.message))
      }
    }

    // Initialize data on mount
    onMounted(() => {
      refreshData()
    })

    const modalTitle = computed(() => {
      if (selectedModalAccessType.value === 'ssh') {
        return 'SSH Access Commands'
      } else if (selectedModalAccessType.value === 'tunnel') {
        return 'Database Tunnel Commands'
      }
      return 'Access Options'
    })

    return {
      // Data
      allProjects,
      paginatedProjects,
      loading,
      error,
      currentPage,
      itemsPerPage,
      
      // Modal states
      showAddModal,
      showEditModal,
      showUsersModal,
      showAgentsModal,
      showDeleteModal,
      submitting,
      
      // Form data
      newProject,
      editProject,
      selectedProject,
      selectedUserId,
      selectedUserRole,
      selectedAgentId,
      selectedAccessType,
      projectToDelete,
      
      // Project Dashboard
      showProjectDashboard,
      dashboardAgents,
      activeConnections,
      
      // Computed
      currentProjectUsers,
      currentProjectAgents,
      availableUsers,
      availableAgents,
      
      // Functions
      refreshData,
      handlePageChange,
      getStatusBadgeClass,
      getRoleBadgeClass,
      getAccessTypeBadgeClass,
      getAccessTypeIcon,
      
      // Access Options Modal
      showAccessOptions,
      selectedAgentForAccess,
      selectedModalAccessType,
      modalTitle,
      customLocalPort,
      customRemoteHost,
      closeAccessOptions,
      copyToClipboard,
      getAccessCommands,
      formatAccessType,
      formatDateTime,
      getProjectUserCount,
      getProjectAgentCount,
      getUsernameById,
      
      // Project CRUD
      openAddProjectModal,
      closeAddProjectModal,
      submitProject,
      openEditProjectModal,
      closeEditProjectModal,
      submitEditProject,
      
      // User management
      openManageUsersModal,
      closeManageUsersModal,
      addUserToProject,
      updateUserRole,
      removeUserFromProject,
      
      // Agent management
      openManageAgentsModal,
      closeManageAgentsModal,
      addAgentToProject,
      updateAgentAccess,
      removeAgentFromProject,
      
      // Project Dashboard
      enterProject,
      closeProjectDashboard,
      loadProjectDashboard,
      isAgentOnline,
      connectSSH,
      openDatabaseTunnel,
      disconnectSession,
      formatConnectionTime,
      
      // Delete
      showDelete,
      closeDeleteModal,
      deleteProject,
      
      // Auth
      currentUser,
      isAdmin,
      isUser
    }
  }
}
</script>

<style scoped>
.project-management-container {
  padding: 1.5rem;
  height: 100%;
  display: flex;
  flex-direction: column;
}

.table-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 1.5rem;
}

.table-title {
  font-size: 1.25rem;
  font-weight: 600;
  color: var(--text-primary);
  margin: 0;
}

.table-actions {
  display: flex;
  gap: 0.75rem;
  align-items: center;
}

.loading-state, .error-state, .empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 12px;
  height: 300px;
  color: var(--text-secondary);
  font-size: 16px;
  flex: 1;
}

.error-state {
  color: var(--color-danger);
}

.loading-state i {
  font-size: 20px;
}

/* No projects message for users */
.no-projects-message {
  display: flex;
  align-items: center;
  justify-content: center;
  flex: 1;
  padding: 60px 20px;
}

.message-content {
  text-align: center;
  max-width: 400px;
}

.message-content i {
  font-size: 48px;
  color: var(--color-info);
  margin-bottom: 20px;
}

.message-content h3 {
  font-size: 24px;
  font-weight: 600;
  color: var(--text-primary);
  margin-bottom: 12px;
}

.message-content p {
  font-size: 16px;
  color: var(--text-secondary);
  line-height: 1.6;
}

.table-wrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
}

.table-container {
  flex: 1;
  overflow: auto;
}

/* Project name styling */
.project-name {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-weight: 500;
}

.project-name i {
  color: var(--color-primary);
}

/* Counter cells */
.counter-cell {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.counter {
  background: var(--surface-alt);
  color: var(--text-secondary);
  padding: 0.25rem 0.5rem;
  border-radius: var(--radius-sm);
  font-size: 0.75rem;
  font-weight: 600;
  min-width: 20px;
  text-align: center;
}

.btn-mini {
  background: none;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  padding: 0.25rem;
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s ease;
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.75rem;
}

.btn-mini:hover {
  background: var(--color-primary);
  color: white;
  border-color: var(--color-primary);
  transform: translateY(-1px);
}

/* Badge styles for status and roles */
.badge {
  padding: 0.25rem 0.5rem;
  border-radius: var(--radius-sm);
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
}

.badge-success {
  background: var(--color-success);
  color: white;
}

.badge-warning {
  background: var(--color-warning);
  color: white;
}

.badge-danger {
  background: var(--color-danger);
  color: white;
}

.badge-primary {
  background: var(--color-primary);
  color: white;
}

.badge-secondary {
  background: var(--surface-alt);
  color: var(--text-secondary);
}

/* Action buttons */
.action-buttons {
  display: flex;
  gap: 0.75rem;
  justify-content: center;
  align-items: center;
}

.action-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  padding: 0.75rem 1rem;
  border: none;
  border-radius: var(--radius-lg);
  font-size: 0.875rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.3s ease;
  text-transform: capitalize;
  letter-spacing: 0.025em;
  min-width: 90px;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.1);
  position: relative;
  overflow: hidden;
}

.action-btn::before {
  content: '';
  position: absolute;
  top: 0;
  left: -100%;
  width: 100%;
  height: 100%;
  background: linear-gradient(90deg, transparent, rgba(255, 255, 255, 0.2), transparent);
  transition: left 0.5s;
}

.action-btn:hover::before {
  left: 100%;
}

.ssh-btn {
  background: linear-gradient(135deg, var(--color-primary) 0%, #2563eb 100%);
  color: white;
  border: 1px solid transparent;
}

.ssh-btn:hover:not(:disabled) {
  background: linear-gradient(135deg, #2563eb 0%, #1d4ed8 100%);
  transform: translateY(-2px);
  box-shadow: 0 8px 16px rgba(59, 130, 246, 0.3);
}

.tunnel-btn {
  background: linear-gradient(135deg, var(--color-success) 0%, #059669 100%);
  color: white;
  border: 1px solid transparent;
}

.tunnel-btn:hover:not(:disabled) {
  background: linear-gradient(135deg, #059669 0%, #047857 100%);
  transform: translateY(-2px);
  box-shadow: 0 8px 16px rgba(16, 185, 129, 0.3);
}

.action-btn:active:not(:disabled) {
  transform: translateY(0);
  box-shadow: 0 4px 8px rgba(0, 0, 0, 0.15);
}

.action-btn:disabled {
  background: linear-gradient(135deg, #9ca3af 0%, #6b7280 100%);
  color: #d1d5db;
  cursor: not-allowed;
  transform: none;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
}

.action-btn:disabled::before {
  display: none;
}

.action-btn i {
  font-size: 1rem;
  opacity: 0.9;
}

.action-btn span {
  font-weight: 600;
  text-shadow: 0 1px 2px rgba(0, 0, 0, 0.1);
}

.disconnect-btn {
  background: linear-gradient(135deg, var(--color-danger) 0%, #dc2626 100%);
  color: white;
  border: 1px solid transparent;
}

.disconnect-btn:hover:not(:disabled) {
  background: linear-gradient(135deg, #dc2626 0%, #b91c1c 100%);
  transform: translateY(-2px);
  box-shadow: 0 8px 16px rgba(239, 68, 68, 0.3);
}

/* Access Options Modal Styles */
.modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal {
  background: var(--surface-color);
  border-radius: var(--radius-lg);
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.15);
  width: 90%;
  max-width: 500px;
  max-height: 90vh;
  overflow: hidden;
}

.modal-large {
  max-width: 900px;
}

.access-options-modal {
  max-width: 1200px;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1.5rem;
  border-bottom: 1px solid var(--border-color);
  background: var(--surface-alt);
}

.modal-header h3 {
  margin: 0;
  color: var(--text-primary);
  font-size: 1.25rem;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.btn-close {
  background: none;
  border: none;
  font-size: 1.25rem;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 0.25rem;
  border-radius: var(--radius-sm);
  transition: all 0.2s ease;
}

.btn-close:hover {
  background: var(--border-color);
  color: var(--text-primary);
}

.modal-body {
  padding: 1.5rem;
  max-height: 60vh;
  overflow-y: auto;
}

.access-options-table {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.access-option-header {
  display: grid;
  grid-template-columns: 200px 1fr 80px;
  gap: 1rem;
  padding: 0.75rem;
  background: var(--surface-alt);
  border-radius: var(--radius-md);
  font-weight: 600;
  font-size: 0.875rem;
  color: var(--text-primary);
  text-transform: uppercase;
  letter-spacing: 0.025em;
}

.access-option-row {
  display: grid;
  grid-template-columns: 200px 1fr 80px;
  gap: 1rem;
  padding: 1rem;
  background: var(--surface-color);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  align-items: center;
  transition: all 0.2s ease;
}

.access-option-row:hover {
  background: var(--surface-alt);
  border-color: var(--color-primary);
}

.custom-row {
  grid-template-columns: 200px 1fr 80px;
}

.service-type {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  font-weight: 600;
  color: var(--text-primary);
}

.ssh-icon {
  color: var(--color-primary);
}

.mysql-icon {
  color: #00758f;
}

.postgres-icon {
  color: #336791;
}

.mongodb-icon {
  color: #4db33d;
}

.custom-icon {
  color: var(--color-warning);
}

.command-text {
  font-family: 'Courier New', monospace;
  font-size: 0.8rem;
  background: var(--surface-alt);
  padding: 0.75rem;
  border-radius: var(--radius-md);
  border: 1px solid var(--border-color);
  color: var(--text-primary);
  word-break: break-all;
  line-height: 1.4;
}

.custom-inputs {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.input-group {
  display: flex;
  gap: 0.5rem;
}

.custom-input {
  padding: 0.5rem;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  background: var(--surface-color);
  color: var(--text-primary);
  font-size: 0.875rem;
  transition: all 0.2s ease;
}

.custom-input:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.port-input {
  width: 100px;
}

.host-input {
  flex: 1;
}

.action-copy {
  display: flex;
  justify-content: center;
}

.copy-btn {
  background: var(--color-primary);
  color: white;
  border: none;
  border-radius: var(--radius-md);
  padding: 0.75rem;
  cursor: pointer;
  transition: all 0.2s ease;
  font-size: 1rem;
}

.copy-btn:hover {
  background: var(--color-primary-dark);
  transform: translateY(-1px);
}

.access-note {
  display: flex;
  gap: 0.75rem;
  margin-top: 2rem;
  padding: 1rem;
  background: rgba(59, 130, 246, 0.1);
  border: 1px solid rgba(59, 130, 246, 0.2);
  border-radius: var(--radius-md);
  color: var(--color-primary);
}

.access-note i {
  color: var(--color-primary);
  margin-top: 0.125rem;
}

.note-content {
  flex: 1;
  font-size: 0.875rem;
  line-height: 1.5;
}

.note-content code {
  background: rgba(59, 130, 246, 0.1);
  padding: 0.125rem 0.25rem;
  border-radius: var(--radius-sm);
  font-family: 'Courier New', monospace;
  font-size: 0.8rem;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
  padding: 1.5rem;
  border-top: 1px solid var(--border-color);
  background: var(--surface-alt);
}

.modal-footer .btn {
  padding: 0.75rem 1.5rem;
  border: none;
  border-radius: var(--radius-md);
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
}

.modal-footer .btn-secondary {
  background: var(--surface-color);
  color: var(--text-secondary);
  border: 1px solid var(--border-color);
}

.modal-footer .btn-secondary:hover {
  background: var(--border-color);
  color: var(--text-primary);
}

/* Form Elements */
.form-group {
  margin-bottom: 1.5rem;
}

.form-group label {
  display: block;
  margin-bottom: 0.5rem;
  color: var(--text-primary);
  font-weight: 500;
  font-size: 0.875rem;
}

.form-input {
  width: 100%;
  padding: 0.75rem;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  background: var(--background-color);
  color: var(--text-primary);
  font-size: 0.875rem;
  transition: all 0.2s ease;
  box-sizing: border-box;
}

.form-input:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
}

.form-input::placeholder {
  color: var(--text-secondary);
}

/* Form row for inline elements */
.form-row {
  display: flex;
  gap: 0.75rem;
  align-items: center;
  margin-bottom: 1rem;
}

.form-row .form-input {
  flex: 1;
}

.form-row .btn {
  white-space: nowrap;
  padding: 0.75rem 1rem;
}

/* User and agent lists */
.users-list, .agents-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.user-item, .agent-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem;
  background: var(--surface-alt);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
}

.user-info, .agent-info {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex: 1;
}

.user-info i, .agent-info i {
  color: var(--color-primary);
  width: 16px;
  text-align: center;
}

.username, .agent-name {
  font-weight: 500;
  color: var(--text-primary);
}

.user-actions, .agent-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.form-input-small {
  padding: 0.5rem;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  background: var(--background-color);
  color: var(--text-primary);
  font-size: 0.75rem;
  min-width: 100px;
}

.btn-danger-small {
  background: var(--color-danger);
  color: white;
  border: none;
  border-radius: var(--radius-sm);
  padding: 0.375rem 0.5rem;
  cursor: pointer;
  transition: all 0.2s ease;
  font-size: 0.75rem;
}

.btn-danger-small:hover {
  background: var(--color-danger-dark);
  transform: translateY(-1px);
}

.empty-message {
  text-align: center;
  color: var(--text-secondary);
  font-style: italic;
  padding: 2rem;
}

/* CSS Custom Properties for consistent theming */
:root {
  --color-primary: #3b82f6;
  --color-primary-dark: #2563eb;
  --color-success: #10b981;
  --color-success-dark: #059669;
  --color-warning: #f59e0b;
  --color-warning-dark: #d97706;
  --color-danger: #ef4444;
  --color-danger-dark: #dc2626;
  --color-info: #8b5cf6;
  --color-info-dark: #7c3aed;
  
  --text-primary: #1f2937;
  --text-secondary: #6b7280;
  --background-color: #ffffff;
  --surface-color: #ffffff;
  --surface-alt: #f9fafb;
  --border-color: #e5e7eb;
  
  --radius-sm: 0.25rem;
  --radius-md: 0.375rem;
  --radius-lg: 0.5rem;
}

/* Dark theme support */
@media (prefers-color-scheme: dark) {
  :root {
    --text-primary: #f9fafb;
    --text-secondary: #d1d5db;
    --background-color: #111827;
    --surface-color: #1f2937;
    --surface-alt: #374151;
    --border-color: #4b5563;
  }
}

/* Project Dashboard Styles */
.project-dashboard {
  padding: 2rem;
  background: var(--surface-alt);
  border-radius: var(--radius-lg);
  margin-top: 1rem;
  border: 1px solid var(--border-color);
}

.dashboard-section {
  margin-bottom: 2rem;
}

.dashboard-section:last-child {
  margin-bottom: 0;
}

.dashboard-section h3 {
  color: var(--text-primary);
  margin-bottom: 1rem;
  font-size: 1.25rem;
  font-weight: 600;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.dashboard-section h3::before {
  content: "";
  display: block;
  width: 4px;
  height: 1.5rem;
  background: var(--color-primary);
  border-radius: 2px;
}

.user-project-message {
  padding: 1.5rem;
  background: var(--color-info-light);
  border: 1px solid var(--color-info);
  border-radius: 8px;
  display: flex;
  align-items: flex-start;
  gap: 1rem;
}

.user-project-message i {
  color: var(--color-info);
  font-size: 1.25rem;
  margin-top: 0.125rem;
}

.user-project-message p {
  margin: 0;
  color: var(--text-secondary);
  line-height: 1.5;
}

.available-agents {
  margin-top: 1rem;
}

.available-agents h4 {
  color: var(--text-primary);
  margin-bottom: 1rem;
  font-size: 1.1rem;
}

.agents-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 1rem;
}

.agent-card {
  background: var(--bg-secondary);
  border: 1px solid var(--border-light);
  border-radius: 8px;
  padding: 1rem;
  transition: all 0.2s ease;
}

.agent-card:hover {
  border-color: var(--color-primary);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
}

.agent-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.75rem;
}

.agent-header h5 {
  margin: 0;
  color: var(--text-primary);
  font-size: 1rem;
}

.agent-header .status {
  padding: 0.25rem 0.5rem;
  border-radius: 12px;
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
}

.agent-header .status.connected {
  background: var(--color-success-light);
  color: var(--color-success);
}

.agent-header .status.disconnected {
  background: var(--color-error-light);
  color: var(--color-error);
}

.agent-details p {
  margin: 0 0 1rem 0;
  color: var(--text-secondary);
  font-size: 0.9rem;
}

.agent-actions {
  display: flex;
  gap: 0.5rem;
}

.btn-action {
  padding: 0.5rem 1rem;
  border: none;
  border-radius: 6px;
  font-size: 0.85rem;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.btn-action.ssh {
  background: var(--color-primary);
  color: white;
}

.btn-action.ssh:hover {
  background: var(--color-primary-dark);
}

.btn-action.tunnel {
  background: var(--color-success);
  color: white;
}

.btn-action.tunnel:hover {
  background: var(--color-success-dark);
}

.agent-offline {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  color: var(--color-warning);
  font-size: 0.85rem;
}

.action-btn.enter-btn:hover {
  background: var(--color-info);
  color: white;
  border-color: var(--color-info);
}

.agents-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 1.5rem;
  margin-top: 1rem;
}

.agent-card {
  background: var(--surface-alt);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-lg);
  padding: 1.5rem;
  transition: all 0.3s ease;
}

.agent-card:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

/* Dashboard Table Styles */
.dashboard-table-container {
  background: var(--surface-color);
  border-radius: var(--radius-lg);
  overflow: hidden;
  border: 1px solid var(--border-color);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);
}

.dashboard-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.875rem;
}

.dashboard-table thead {
  background: var(--surface-alt);
  border-bottom: 2px solid var(--border-color);
}

.dashboard-table thead th {
  padding: 1rem 1.25rem;
  text-align: left;
  font-weight: 600;
  color: var(--text-primary);
  font-size: 0.875rem;
  border-bottom: 1px solid var(--border-color);
}

.dashboard-table thead th i {
  margin-right: 0.5rem;
  color: var(--color-primary);
}

.dashboard-table tbody tr {
  border-bottom: 1px solid var(--border-color);
  transition: background-color 0.2s ease;
}

.dashboard-table tbody tr:hover {
  background: var(--surface-alt);
}

.dashboard-table tbody tr:last-child {
  border-bottom: none;
}

.dashboard-table td {
  padding: 1rem 1.25rem;
  vertical-align: middle;
}

.agent-id-cell {
  font-weight: 500;
}

.agent-id-info {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.agent-icon {
  color: var(--color-primary);
  font-size: 1.1rem;
}

.agent-name {
  font-weight: 600;
  color: var(--text-primary);
}

.access-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.375rem 0.75rem;
  border-radius: var(--radius-md);
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.025em;
}

.access-primary {
  background: rgba(59, 130, 246, 0.1);
  color: var(--color-primary);
  border: 1px solid rgba(59, 130, 246, 0.2);
}

.access-success {
  background: rgba(16, 185, 129, 0.1);
  color: var(--color-success);
  border: 1px solid rgba(16, 185, 129, 0.2);
}

.access-info {
  background: rgba(139, 92, 246, 0.1);
  color: var(--color-info);
  border: 1px solid rgba(139, 92, 246, 0.2);
}

.connection-type-badge {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  padding: 0.375rem 0.75rem;
  border-radius: var(--radius-md);
  font-size: 0.75rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.025em;
}

.type-ssh {
  background: rgba(59, 130, 246, 0.1);
  color: var(--color-primary);
  border: 1px solid rgba(59, 130, 246, 0.2);
}

.type-database {
  background: rgba(16, 185, 129, 0.1);
  color: var(--color-success);
  border: 1px solid rgba(16, 185, 129, 0.2);
}

.duration-text {
  display: flex;
  align-items: center;
  gap: 0.375rem;
  color: var(--text-secondary);
  font-size: 0.875rem;
}

.duration-text i {
  color: var(--color-info);
}

.agent-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 1rem;
}

.agent-info {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}
</style>
