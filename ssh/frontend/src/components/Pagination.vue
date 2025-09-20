<template>
  <div class="pagination-container" v-if="totalPages > 1">
    <div class="pagination-info">
      Showing {{ startItem }} to {{ endItem }} of {{ totalItems }} entries
    </div>
    
    <nav class="pagination">
      <!-- Previous Button -->
      <button 
        class="pagination-btn"
        :class="{ 'disabled': currentPage === 1 }"
        @click="goToPage(currentPage - 1)"
        :disabled="currentPage === 1"
      >
        <i class="fas fa-chevron-left"></i>
        Previous
      </button>

      <!-- Page Numbers -->
      <div class="pagination-numbers">
        <!-- First page if not in visible range -->
        <button 
          v-if="startPage > 1"
          class="pagination-number"
          @click="goToPage(1)"
        >
          1
        </button>
        
        <!-- Ellipsis if there's gap -->
        <span v-if="startPage > 2" class="pagination-ellipsis">...</span>
        
        <!-- Visible page numbers -->
        <button
          v-for="page in visiblePages"
          :key="page"
          class="pagination-number"
          :class="{ 'active': page === currentPage }"
          @click="goToPage(page)"
        >
          {{ page }}
        </button>
        
        <!-- Ellipsis if there's gap -->
        <span v-if="endPage < totalPages - 1" class="pagination-ellipsis">...</span>
        
        <!-- Last page if not in visible range -->
        <button 
          v-if="endPage < totalPages"
          class="pagination-number"
          @click="goToPage(totalPages)"
        >
          {{ totalPages }}
        </button>
      </div>

      <!-- Next Button -->
      <button 
        class="pagination-btn"
        :class="{ 'disabled': currentPage === totalPages }"
        @click="goToPage(currentPage + 1)"
        :disabled="currentPage === totalPages"
      >
        Next
        <i class="fas fa-chevron-right"></i>
      </button>
    </nav>
  </div>
</template>

<script>
import { defineComponent, computed } from 'vue'

export default defineComponent({
  name: 'Pagination',
  props: {
    currentPage: {
      type: Number,
      required: true,
      default: 1
    },
    totalItems: {
      type: Number,
      required: true,
      default: 0
    },
    itemsPerPage: {
      type: Number,
      required: true,
      default: 20
    },
    maxVisiblePages: {
      type: Number,
      default: 5
    }
  },
  emits: ['page-changed'],
  setup(props, { emit }) {
    const totalPages = computed(() => {
      return Math.ceil(props.totalItems / props.itemsPerPage)
    })

    const startItem = computed(() => {
      return (props.currentPage - 1) * props.itemsPerPage + 1
    })

    const endItem = computed(() => {
      const end = props.currentPage * props.itemsPerPage
      return end > props.totalItems ? props.totalItems : end
    })

    const startPage = computed(() => {
      const start = props.currentPage - Math.floor(props.maxVisiblePages / 2)
      return Math.max(1, start)
    })

    const endPage = computed(() => {
      const end = startPage.value + props.maxVisiblePages - 1
      return Math.min(totalPages.value, end)
    })

    const visiblePages = computed(() => {
      const pages = []
      for (let i = startPage.value; i <= endPage.value; i++) {
        pages.push(i)
      }
      return pages
    })

    const goToPage = (page) => {
      if (page >= 1 && page <= totalPages.value && page !== props.currentPage) {
        emit('page-changed', page)
      }
    }

    return {
      totalPages,
      startItem,
      endItem,
      startPage,
      endPage,
      visiblePages,
      goToPage
    }
  }
})
</script>

<style scoped>
.pagination-container {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 1rem;
  border-top: 1px solid var(--border-color);
  background: var(--surface-color);
}

.pagination-info {
  color: var(--text-secondary);
  font-size: 0.875rem;
}

.pagination {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.pagination-btn {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.5rem 1rem;
  background: var(--surface-color);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  color: var(--text-primary);
  cursor: pointer;
  transition: all 0.3s ease;
  font-size: 0.875rem;
}

.pagination-btn:hover:not(.disabled) {
  background: var(--primary-color);
  color: white;
  border-color: var(--primary-color);
}

.pagination-btn.disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.pagination-numbers {
  display: flex;
  align-items: center;
  gap: 0.25rem;
}

.pagination-number {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--surface-color);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  color: var(--text-primary);
  cursor: pointer;
  transition: all 0.3s ease;
  font-size: 0.875rem;
  font-weight: 500;
}

.pagination-number:hover {
  background: var(--primary-color);
  color: white;
  border-color: var(--primary-color);
}

.pagination-number.active {
  background: var(--primary-color);
  color: white;
  border-color: var(--primary-color);
  font-weight: 600;
}

.pagination-ellipsis {
  padding: 0.5rem;
  color: var(--text-secondary);
  font-weight: 600;
}

/* Mobile responsiveness */
@media (max-width: 768px) {
  .pagination-container {
    flex-direction: column;
    gap: 1rem;
    padding: 0.75rem;
  }

  .pagination-info {
    font-size: 0.8rem;
  }

  .pagination-btn {
    padding: 0.375rem 0.75rem;
    font-size: 0.8rem;
  }

  .pagination-number {
    width: 36px;
    height: 36px;
    font-size: 0.8rem;
  }

  .pagination-numbers {
    gap: 0.125rem;
  }

  .pagination {
    gap: 0.375rem;
  }
}

@media (max-width: 480px) {
  .pagination-btn span {
    display: none;
  }

  .pagination-number {
    width: 32px;
    height: 32px;
    font-size: 0.75rem;
  }
}
</style>