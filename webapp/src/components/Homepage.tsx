import { createResource, For } from 'solid-js'

interface DataSource {
  id: string
  name: string
  description: string
  status: 'available' | 'connected' | 'error'
  lastUpdated?: string
  recordCount?: number
}

const fetchDataSources = async (): Promise<DataSource[]> => {
  try {
    const response = await fetch('/api/sources')
    if (!response.ok) {
      throw new Error(`HTTP ${response.status}: Failed to fetch data sources`)
    }
    
    const contentType = response.headers.get('content-type')
    if (!contentType || !contentType.includes('application/json')) {
      throw new Error('Response is not JSON')
    }
    
    return await response.json()
  } catch (error) {
    console.warn('API not available, using mock data:', error instanceof Error ? error.message : String(error))
    return [
      {
        id: 'hackernews',
        name: 'Hacker News',
        description: 'Tech news and discussions from the Hacker News community',
        status: 'connected',
        lastUpdated: '2025-08-12T05:30:00Z',
        recordCount: 15420
      },
      {
        id: 'reddit',
        name: 'Reddit',
        description: 'Social news aggregation, web content rating, and discussion',
        status: 'available',
        recordCount: 0
      },
      {
        id: 'twitter',
        name: 'Twitter',
        description: 'Social media platform for real-time news and updates',
        status: 'error',
        recordCount: 0
      }
    ]
  }
}

function DataSourceCard({ source }: { source: DataSource }) {
  const formatDate = (dateStr?: string) => {
    if (!dateStr) return 'Never'
    const date = new Date(dateStr)
    return date.toLocaleDateString('en-US', { 
      month: 'short', 
      day: 'numeric',
      hour: '2-digit',
      minute: '2-digit'
    })
  }

  const formatCount = (count?: number) => {
    if (!count || count === 0) return '0'
    if (count >= 1000000) return `${(count / 1000000).toFixed(1)}M`
    if (count >= 1000) return `${(count / 1000).toFixed(1)}K`
    return count.toString()
  }

  return (
    <div class="bg-white rounded-lg border border-gray-200 p-6 hover:shadow-md transition-shadow">
      <div class="flex items-start justify-between">
        <div class="flex-1">
          <div class="flex items-center space-x-2 mb-2">
            <h3 class="text-lg font-semibold text-gray-900">{source.name}</h3>
            <div class={`w-2 h-2 rounded-full ${
              source.status === 'connected' ? 'bg-green-400' :
              source.status === 'error' ? 'bg-red-400' : 
              'bg-gray-400'
            }`} />
          </div>
          <p class="text-gray-600 text-sm mb-4">{source.description}</p>
          
          <div class="flex items-center justify-between text-xs text-gray-500">
            <span>Last updated: {formatDate(source.lastUpdated)}</span>
            <span>{formatCount(source.recordCount)} records</span>
          </div>
        </div>
      </div>

      <div class="mt-4 flex space-x-2">
        <a
          href={`/sources/${source.id}`}
          class={`px-3 py-1 text-xs font-medium rounded-md transition-colors ${
            source.status === 'connected' 
              ? 'bg-blue-50 text-blue-700 hover:bg-blue-100' 
              : 'bg-green-50 text-green-700 hover:bg-green-100'
          }`}
        >
          {source.status === 'connected' ? 'View Data' : 'Connect'}
        </a>
        
        {source.status === 'connected' && (
          <button 
            class="px-3 py-1 text-xs font-medium text-gray-700 bg-gray-50 hover:bg-gray-100 rounded-md transition-colors"
            onClick={() => console.log(`Refresh ${source.name}`)}
          >
            Refresh
          </button>
        )}
      </div>
    </div>
  )
}

export default function Homepage() {
  const [dataSources] = createResource(fetchDataSources)

  return (
    <div class="bg-gray-50">
      <div class="max-w-7xl mx-auto px-4 py-8">
        <div class="mb-8">
          <h1 class="text-2xl font-bold text-gray-900 mb-2">Data Sources</h1>
          <p class="text-gray-600">Connect to and query data from various public sources</p>
        </div>

        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          <For each={dataSources()} fallback={
            <div class="col-span-full flex items-center justify-center py-12">
              <div class="text-gray-500">Loading data sources...</div>
            </div>
          }>
            {(source) => <DataSourceCard source={source} />}
          </For>
        </div>

        {dataSources()?.length === 0 && (
          <div class="text-center py-12">
            <div class="text-gray-500 text-lg mb-2">No data sources available</div>
            <p class="text-gray-400">Check your configuration or try refreshing the page</p>
          </div>
        )}
      </div>
    </div>
  )
}