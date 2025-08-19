import { createResource, For } from 'solid-js'
import * as DropdownMenu from '@kobalte/core/dropdown-menu'

interface DataSource {
  id: string
  name: string
  description: string
  status: 'available' | 'connected' | 'error'
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
        description: 'Tech news and discussions',
        status: 'available'
      },
      {
        id: 'reddit',
        name: 'Reddit',
        description: 'Social news aggregation',
        status: 'available'
      },
      {
        id: 'twitter',
        name: 'Twitter',
        description: 'Social media platform',
        status: 'available'
      }
    ]
  }
}

export default function Navigation() {
  const [dataSources] = createResource(fetchDataSources)

  return (
    <nav class="bg-white border-b border-gray-200 px-4 py-3 flex items-center justify-between">
      <div class="flex items-center space-x-6">
        <h1 class="text-xl font-bold text-gray-900">PubDataHub</h1>
        
        <div class="flex items-center space-x-4">
          <DropdownMenu.Root>
            <DropdownMenu.Trigger class="flex items-center space-x-1 px-3 py-2 text-sm font-medium text-gray-700 hover:text-gray-900 hover:bg-gray-50 rounded-md transition-colors">
              <span>Data Sources</span>
              <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width={2} d="M19 9l-7 7-7-7" />
              </svg>
            </DropdownMenu.Trigger>
            
            <DropdownMenu.Portal>
              <DropdownMenu.Content class="bg-white rounded-md shadow-lg ring-1 ring-black ring-opacity-5 focus:outline-none z-50 min-w-[200px] max-h-96 overflow-auto">
                <div class="py-1">
                  <For each={dataSources()} fallback={
                    <div class="px-4 py-2 text-sm text-gray-500">Loading...</div>
                  }>
                    {(source) => (
                      <DropdownMenu.Item class="flex items-center px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 cursor-pointer">
                <a href={`/sources/${source.id}`} class="flex items-center space-x-3 w-full">
                  <div class={`w-2 h-2 rounded-full ${
                    source.status === 'connected' ? 'bg-green-400' :
                    source.status === 'error' ? 'bg-red-400' : 
                    'bg-gray-400'
                  }`} />
                  <div class="flex-1">
                    <div class="font-medium">{source.name}</div>
                    <div class="text-xs text-gray-500">{source.description}</div>
                  </div>
                </a>
              </DropdownMenu.Item>
                    )}
                  </For>
                </div>
              </DropdownMenu.Content>
            </DropdownMenu.Portal>
          </DropdownMenu.Root>
        </div>
      </div>

      <div class="flex items-center space-x-4">
        <a 
          href="/settings" 
          class="text-sm font-medium text-gray-700 hover:text-gray-900 hover:bg-gray-50 px-3 py-2 rounded-md transition-colors"
        >
          Settings
        </a>
      </div>
    </nav>
  )
}