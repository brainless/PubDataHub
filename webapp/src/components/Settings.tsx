import { createResource, createSignal, onCleanup } from 'solid-js'
import { useNavigate } from '@solidjs/router'

interface JobInfo {
  id: string
  type: string
  state: string
  priority: number
  progress: {
    current: number
    total: number
    message: string
  }
  start_time: string
  end_time?: string
  error_message?: string
  retry_count: number
  max_retries: number
  created_by: string
  description: string
  metadata: {
    [key: string]: any
  }
}

export default function Settings() {
  const navigate = useNavigate()
  const [jobs, { refetch }] = createResource<JobInfo[]>(async () => {
    try {
      const response = await fetch('/api/jobs')
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`)
      }
      
      const contentType = response.headers.get('content-type')
      if (!contentType || !contentType.includes('application/json')) {
        throw new Error('Response is not JSON')
      }
      
      const data = await response.json()
      return data
    } catch (error) {
      console.error('Failed to fetch jobs:', error)
      return []
    }
  })
  
  // Set up periodic refresh of jobs data
  const intervalId = setInterval(() => {
    refetch()
  }, 5000) // Refresh every 5 seconds
  
  // Clean up interval on component unmount
  onCleanup(() => {
    clearInterval(intervalId)
  })
  
  const [loading, setLoading] = createSignal(false)
  const [error, setError] = createSignal<string | null>(null)
  
  // Format date for display
  const formatDate = (dateStr?: string) => {
    if (!dateStr) return 'Unknown'
    const date = new Date(dateStr)
    return date.toLocaleString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    })
  }
  
  // Format progress percentage
  const formatProgress = (progress: JobInfo['progress']) => {
    if (progress.total === 0) return '0%'
    return `${Math.round((progress.current / progress.total) * 100)}%`
  }
  
  // Format job state for display
  const formatState = (state: string) => {
    switch (state.toLowerCase()) {
      case 'queued':
        return 'Queued'
      case 'running':
        return 'Running'
      case 'paused':
        return 'Paused'
      case 'completed':
        return 'Completed'
      case 'failed':
        return 'Failed'
      default:
        return state.charAt(0).toUpperCase() + state.slice(1)
    }
  }
  
  // Handle pause job
  const handlePause = async (jobId: string) => {
    setLoading(true)
    setError(null)
    
    try {
      const response = await fetch(`/api/jobs/${jobId}/pause`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        }
      })
      
      if (!response.ok) {
        const errorText = await response.text()
        throw new Error(`HTTP ${response.status}: ${errorText || response.statusText}`)
      }
      
      // Refetch jobs to update the list
      await refetch()
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
      console.error('Failed to pause job:', err)
    } finally {
      setLoading(false)
    }
  }
  
  // Handle resume job
  const handleResume = async (jobId: string) => {
    setLoading(true)
    setError(null)
    
    try {
      const response = await fetch(`/api/jobs/${jobId}/resume`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        }
      })
      
      if (!response.ok) {
        const errorText = await response.text()
        throw new Error(`HTTP ${response.status}: ${errorText || response.statusText}`)
      }
      
      // Refetch jobs to update the list
      await refetch()
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err))
      console.error('Failed to resume job:', err)
    } finally {
      setLoading(false)
    }
  }
  
  // Render job actions
  const renderActions = (job: JobInfo) => {
    if (job.state.toLowerCase() === 'completed' || job.state.toLowerCase() === 'failed') {
      return (
        <span class="text-gray-500 text-sm">No actions available</span>
      )
    }
    
    if (job.state.toLowerCase() === 'paused') {
      return (
        <button
          onClick={() => handleResume(job.id)}
          class="px-3 py-1 text-xs font-medium text-green-700 bg-green-100 hover:bg-green-200 rounded-md transition-colors disabled:opacity-50"
          disabled={loading()}
        >
          Resume
        </button>
      )
    }
    
    if (job.state.toLowerCase() === 'running' || job.state.toLowerCase() === 'queued') {
      return (
        <button
          onClick={() => handlePause(job.id)}
          class="px-3 py-1 text-xs font-medium text-yellow-700 bg-yellow-100 hover:bg-yellow-200 rounded-md transition-colors disabled:opacity-50"
          disabled={loading()}
        >
          Pause
        </button>
      )
    }
    
    return null
  }
  
  // Render the jobs table
  const renderJobsTable = () => {
    if (jobs.loading) {
      return (
        <div class="flex justify-center items-center py-12">
          <div class="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-500"></div>
        </div>
      )
    }
    
    if (error()) {
      return (
        <div class="bg-red-50 border border-red-200 rounded-lg p-6 text-center">
          <div class="text-red-800 font-medium mb-2">Error Loading Jobs</div>
          <div class="text-red-600 text-sm">{error()}</div>
        </div>
      )
    }
    
    if (!jobs() || jobs()?.length === 0) {
      return (
        <div class="bg-gray-50 border border-gray-200 rounded-lg p-8 text-center">
          <div class="text-gray-500 font-medium mb-2">No Downloads</div>
          <p class="text-gray-400 text-sm mb-4">There are no active or completed downloads at this time.</p>
          <p class="text-gray-400 text-sm">Start a download from the Data Sources page to see it here.</p>
        </div>
      )
    }
    
    return (
      <div class="overflow-x-auto">
        <table class="min-w-full divide-y divide-gray-200">
          <thead class="bg-gray-50">
            <tr>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">ID</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Type</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">State</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Progress</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Started</th>
              <th scope="col" class="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">Actions</th>
            </tr>
          </thead>
          <tbody class="bg-white divide-y divide-gray-200">
            {jobs()?.map((job) => (
              <tr class="hover:bg-gray-50">
                <td class="px-6 py-4 whitespace-nowrap text-sm font-mono text-gray-900">{job.id.substring(0, 8)}...</td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900 capitalize">{job.type}</td>
                <td class="px-6 py-4 whitespace-nowrap">
                  <span class={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${
                    job.state.toLowerCase() === 'completed' ? 'bg-green-100 text-green-800' :
                    job.state.toLowerCase() === 'running' ? 'bg-blue-100 text-blue-800' :
                    job.state.toLowerCase() === 'paused' ? 'bg-yellow-100 text-yellow-800' :
                    job.state.toLowerCase() === 'failed' ? 'bg-red-100 text-red-800' :
                    'bg-gray-100 text-gray-800'
                  }`}>
                    {formatState(job.state)}
                  </span>
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-900">
                  <div class="flex items-center">
                    <div class="w-full bg-gray-200 rounded-full h-2 mr-2">
                      <div 
                        class="bg-blue-600 h-2 rounded-full" 
                        style={{ width: formatProgress(job.progress) }}
                      ></div>
                    </div>
                    <span>{formatProgress(job.progress)}</span>
                  </div>
                </td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">{formatDate(job.start_time)}</td>
                <td class="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                  {renderActions(job)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    )
  }
  
  return (
    <div class="max-w-7xl mx-auto px-4 py-8">
      <div class="mb-8">
        <h1 class="text-2xl font-bold text-gray-900 mb-2">Settings</h1>
        <p class="text-gray-600">Manage downloads and application settings</p>
      </div>
      
      <div class="bg-white shadow rounded-lg overflow-hidden">
        <div class="px-6 py-4 border-b border-gray-200 flex justify-between items-center">
          <div>
            <h2 class="text-lg font-medium text-gray-900">Downloads Manager</h2>
            <p class="text-sm text-gray-500 mt-1">Monitor and control your data downloads</p>
          </div>
          <button
            onClick={() => refetch()}
            class="px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 hover:bg-gray-200 rounded-md transition-colors disabled:opacity-50"
            disabled={jobs.loading || loading()}
          >
            Refresh
          </button>
        </div>
        
        <div class="p-6">
          {renderJobsTable()}
        </div>
      </div>
      
      <div class="mt-8 bg-white shadow rounded-lg overflow-hidden">
        <div class="px-6 py-4 border-b border-gray-200">
          <h2 class="text-lg font-medium text-gray-900">Application Settings</h2>
          <p class="text-sm text-gray-500 mt-1">Configure application preferences</p>
        </div>
        
        <div class="p-6">
          <div class="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div class="border border-gray-200 rounded-lg p-4">
              <h3 class="font-medium text-gray-900 mb-2">Storage Configuration</h3>
              <p class="text-sm text-gray-600 mb-4">Manage where data is stored on your system</p>
              <button 
                class="px-4 py-2 text-sm font-medium text-blue-700 bg-blue-50 hover:bg-blue-100 rounded-md transition-colors"
                onClick={() => navigate('/settings/storage')}
              >
                Configure Storage
              </button>
            </div>
            
            <div class="border border-gray-200 rounded-lg p-4">
              <h3 class="font-medium text-gray-900 mb-2">Data Sources</h3>
              <p class="text-sm text-gray-600 mb-4">Manage available data sources</p>
              <button 
                class="px-4 py-2 text-sm font-medium text-blue-700 bg-blue-50 hover:bg-blue-100 rounded-md transition-colors"
                onClick={() => navigate('/sources')}
              >
                Manage Sources
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  )
}