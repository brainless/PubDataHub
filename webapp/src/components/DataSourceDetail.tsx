import { createSignal, Show } from 'solid-js'
import { useLocation } from '@solidjs/router'

interface DataSourceItem {
  id: string
  title?: string
  url?: string
  author?: string
  points?: number
  comment_count?: number
  timestamp?: string
  type?: string
}

interface Pagination {
  currentPage: number
  totalPages: number
  totalItems: number
  itemsPerPage: number
}

export default function DataSourceDetail() {
  const location = useLocation()
  // Extract source name from URL path
  const sourceName = location.pathname.split('/')[3] || ''
  
  // State for pagination
  const [pagination, setPagination] = createSignal<Pagination>({
    currentPage: 1,
    totalPages: 1,
    totalItems: 0,
    itemsPerPage: 20
  })
  
  // State for loading and data
  const [loading, setLoading] = createSignal(true)
  const [error, setError] = createSignal<string | null>(null)
  const [data, setData] = createSignal<DataSourceItem[]>([])
  
  // Fetch data from API
  const fetchData = async (page: number = 1) => {
    setLoading(true)
    setError(null)
    
    try {
      const response = await fetch(`/api/sources/${sourceName}/data?page=${page}&limit=${pagination().itemsPerPage}`)
      
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`)
      }
      
      const contentType = response.headers.get('content-type')
      if (!contentType || !contentType.includes('application/json')) {
        throw new Error('Response is not JSON')
      }
      
      const result = await response.json()
      
      setData(result.data || [])
      setPagination({
        currentPage: result.currentPage || page,
        totalPages: result.totalPages || 1,
        totalItems: result.totalItems || 0,
        itemsPerPage: pagination().itemsPerPage
      })
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
      console.error('Error fetching data:', err)
    } finally {
      setLoading(false)
    }
  }
  
  // Load initial data
  fetchData()
  
  // Handle page change
  const handlePageChange = (newPage: number) => {
    if (newPage >= 1 && newPage <= pagination().totalPages && newPage !== pagination().currentPage) {
      fetchData(newPage)
    }
  }
  
  // Format timestamp for display
  const formatDate = (timestamp?: string) => {
    if (!timestamp) return 'Unknown'
    const date = new Date(timestamp)
    return date.toLocaleDateString('en-US', { 
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    })
  }
  
  // Render the data grid
  const renderGrid = () => {
    if (loading()) {
      return (
        <div class="flex justify-center items-center py-12">
          <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
        </div>
      )
    }
    
    if (error()) {
      return (
        <div class="bg-red-50 border border-red-200 rounded-lg p-6 text-center">
          <div class="text-red-800 font-medium mb-2">Error Loading Data</div>
          <div class="text-red-600 text-sm">{error()}</div>
        </div>
      )
    }
    
    if (data().length === 0) {
      return (
        <div class="bg-yellow-50 border border-yellow-200 rounded-lg p-6 text-center">
          <div class="text-yellow-800 font-medium mb-2">No Data Available</div>
          <div class="text-yellow-600 text-sm">There is no data for this source yet.</div>
        </div>
      )
    }
    
    return (
      <div class="overflow-x-auto">
        <table class="min-w-full divide-y divide-gray-200">
          <thead class="bg-gray-50">
            <tr>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">ID</th>
              <Show when={sourceName === 'hackernews'}>
                <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Title</th>
                <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Author</th>
                <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Points</th>
                <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Comments</th>
              </Show>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Timestamp</th>
            </tr>
          </thead>
          <tbody class="bg-white divide-y divide-gray-200">
            {data().map((item) => (
              <tr class="hover:bg-gray-50">
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">{item.id}</td>
                <Show when={sourceName === 'hackernews'}>
                  <td class="px-6 py-4 text-sm text-gray-900 max-w-xs truncate">
                    {item.title}
                  </td>
                  <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">{item.author}</td>
                  <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">{item.points}</td>
                  <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">{item.comment_count}</td>
                </Show>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{formatDate(item.timestamp)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    )
  }
  
  // Render pagination controls
  const renderPagination = () => {
    const pag = pagination()
    const pagesToShow = Math.min(5, pag.totalPages)
    let startPage = Math.max(1, pag.currentPage - Math.floor(pagesToShow / 2))
    let endPage = Math.min(pag.totalPages, startPage + pagesToShow - 1)
    
    if (endPage - startPage + 1 < pagesToShow) {
      startPage = Math.max(1, endPage - pagesToShow + 1)
    }
    
    const pageNumbers = []
    for (let i = startPage; i <= endPage; i++) {
      pageNumbers.push(i)
    }
    
    return (
      <div class="flex items-center justify-between border-t border-gray-200 bg-white px-4 py-3 sm:px-6">
        <div class="flex flex-1 justify-between sm:hidden">
          <button
            onClick={() => handlePageChange(pag.currentPage - 1)}
            disabled={pag.currentPage === 1}
            class="relative inline-flex items-center rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50"
          >
            Previous
          </button>
          <button
            onClick={() => handlePageChange(pag.currentPage + 1)}
            disabled={pag.currentPage === pag.totalPages}
            class="relative ml-3 inline-flex items-center rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50"
          >
            Next
          </button>
        </div>
        <div class="hidden sm:flex sm:flex-1 sm:items-center sm:justify-between">
          <div>
            <p class="text-sm text-gray-700">
              Showing <span class="font-medium">{(pag.currentPage - 1) * pag.itemsPerPage + 1}</span> to{' '}
              <span class="font-medium">
                {Math.min(pag.currentPage * pag.itemsPerPage, pag.totalItems)}
              </span> of{' '}
              <span class="font-medium">{pag.totalItems}</span> results
            </p>
          </div>
          <div>
            <nav class="isolate inline-flex -space-x-px rounded-md shadow-sm" aria-label="Pagination">
              <button
                onClick={() => handlePageChange(pag.currentPage - 1)}
                disabled={pag.currentPage === 1}
                class="relative inline-flex items-center rounded-l-md px-2 py-2 text-gray-400 ring-1 ring-inset ring-gray-300 hover:bg-gray-50 disabled:opacity-50"
              >
                <span class="sr-only">Previous</span>
                <svg class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                  <path fill-rule="evenodd" d="M12.79 5.23a.75.75 0 010 1.06L9.06 10l3.73 3.73a.75.75 0 11-1.06 1.06l-4.25-4.25a.75.75 0 010-1.06l4.25-4.25a.75.75 0 011.06 0z" clip-rule="evenodd" />
                </svg>
              </button>
              
              {pageNumbers.map((pageNum) => (
                <button
                  onClick={() => handlePageChange(pageNum)}
                  class={`relative inline-flex items-center px-4 py-2 text-sm font-semibold ${
                    pageNum === pag.currentPage
                      ? 'z-10 bg-blue-600 text-white focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-blue-600'
                      : 'text-gray-900 ring-1 ring-inset ring-gray-300 hover:bg-gray-50'
                  }`}
                >
                  {pageNum}
                </button>
              ))}
              
              <button
                onClick={() => handlePageChange(pag.currentPage + 1)}
                disabled={pag.currentPage === pag.totalPages}
                class="relative inline-flex items-center rounded-r-md px-2 py-2 text-gray-400 ring-1 ring-inset ring-gray-300 hover:bg-gray-50 disabled:opacity-50"
              >
                <span class="sr-only">Next</span>
                <svg class="h-5 w-5" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
                  <path fill-rule="evenodd" d="M7.21 14.77a.75.75 0 010-1.06L10.94 10 7.21 6.27a.75.75 0 111.06-1.06l4.25 4.25a.75.75 0 010 1.06l-4.25 4.25a.75.75 0 01-1.06 0z" clip-rule="evenodd" />
                </svg>
              </button>
            </nav>
          </div>
        </div>
      </div>
    )
  }
  
  return (
    <div class="max-w-7xl mx-auto px-4 py-8">
      <div class="mb-6">
        <h1 class="text-2xl font-bold text-gray-900 mb-2 capitalize">{sourceName} Data</h1>
        <p class="text-gray-600">Browse and explore data from {sourceName}</p>
      </div>
      
      <div class="bg-white shadow rounded-lg overflow-hidden">
        {renderGrid()}
        {renderPagination()}
      </div>
    </div>
  )
}