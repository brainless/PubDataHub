import type { PathValidationResult } from "@/components/settings/PathValidation"

// Types for storage configuration
export interface StorageConfig {
  defaultPath: string
  maxStoragePerDataset: number // in GB
  totalStorageLimit: number // in GB
  autoDeleteAfterDays: number
  enableCompression: boolean
  lastUpdated: string
}

export interface StorageStats {
  usedSpace: string
  freeSpace: string
  totalSpace: string
  numberOfDatasets: number
  oldestDownload?: string
}

// Mock API functions - these would typically call actual backend endpoints
export const configApi = {
  /**
   * Get current storage configuration
   */
  async getStorageConfig(): Promise<StorageConfig> {
    // Simulate API call
    await new Promise(resolve => setTimeout(resolve, 500))
    
    return {
      defaultPath: "~/Downloads/PubDataHub",
      maxStoragePerDataset: 1,
      totalStorageLimit: 10,
      autoDeleteAfterDays: 30,
      enableCompression: false,
      lastUpdated: new Date().toISOString()
    }
  },

  /**
   * Update storage configuration
   */
  async updateStorageConfig(config: Partial<StorageConfig>): Promise<StorageConfig> {
    // Simulate API call
    await new Promise(resolve => setTimeout(resolve, 1000))
    
    // In a real app, this would send the config to the backend
    console.log("Updating storage config:", config)
    
    // Return updated config (mock)
    return {
      defaultPath: config.defaultPath || "~/Downloads/PubDataHub",
      maxStoragePerDataset: config.maxStoragePerDataset || 1,
      totalStorageLimit: config.totalStorageLimit || 10,
      autoDeleteAfterDays: config.autoDeleteAfterDays || 30,
      enableCompression: config.enableCompression || false,
      lastUpdated: new Date().toISOString()
    }
  },

  /**
   * Validate a storage path
   */
  async validatePath(path: string): Promise<PathValidationResult> {
    // Simulate API call
    await new Promise(resolve => setTimeout(resolve, 800))
    
    // Mock validation logic
    const issues: string[] = []
    const warnings: string[] = []
    
    // Basic validation checks
    if (!path.trim()) {
      issues.push("Path cannot be empty")
    }
    
    if (path.includes("..")) {
      issues.push("Path cannot contain relative directories (..)")
    }
    
    if (path.startsWith("/tmp") || path.startsWith("/temp")) {
      warnings.push("Temporary directories may not persist between system restarts")
    }
    
    if (path.includes(" ")) {
      warnings.push("Paths with spaces may cause issues with some tools")
    }
    
    // Mock results based on path content
    const exists = !path.includes("nonexistent")
    const isDirectory = exists && !path.includes("file.")
    const isWritable = exists && !path.includes("readonly")
    
    const isValid = issues.length === 0 && exists && isDirectory && isWritable
    
    return {
      isValid,
      exists,
      isWritable,
      isDirectory,
      freeSpace: exists ? "15.2 GB" : undefined,
      issues,
      warnings
    }
  },

  /**
   * Get storage statistics
   */
  async getStorageStats(): Promise<StorageStats> {
    // Simulate API call
    await new Promise(resolve => setTimeout(resolve, 300))
    
    return {
      usedSpace: "2.4 GB",
      freeSpace: "15.2 GB", 
      totalSpace: "17.6 GB",
      numberOfDatasets: 12,
      oldestDownload: "2024-06-15"
    }
  },

  /**
   * Browse for directory (this would typically trigger a native file dialog)
   */
  async browseForDirectory(): Promise<string | null> {
    // Simulate file dialog
    await new Promise(resolve => setTimeout(resolve, 500))
    
    // In a real implementation, this would:
    // 1. Trigger a native file dialog via backend API
    // 2. Return the selected path or null if cancelled
    
    // Mock: return a sample path for demo purposes
    const mockPaths = [
      "/Users/john/Documents/DataStorage",
      "/home/user/data",
      "~/Desktop/MyData",
      null // cancelled
    ]
    
    return mockPaths[Math.floor(Math.random() * mockPaths.length)]
  },

  /**
   * Move existing data to new location
   */
  async moveData(fromPath: string, toPath: string): Promise<void> {
    // Simulate data migration
    await new Promise(resolve => setTimeout(resolve, 2000))
    
    console.log(`Moving data from ${fromPath} to ${toPath}`)
    
    // In a real implementation, this would handle:
    // 1. Copying existing files to new location
    // 2. Updating database references
    // 3. Cleaning up old location
    // 4. Progress reporting
  }
}