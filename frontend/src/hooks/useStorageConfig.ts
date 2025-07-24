import { useState, useEffect, useCallback } from "react"
import { configApi } from "@/services/configApi"
import type { StorageConfig, StorageStats } from "@/services/configApi"
import type { PathValidationResult } from "@/components/settings/PathValidation"

interface UseStorageConfigReturn {
  // Config state
  config: StorageConfig | null
  stats: StorageStats | null
  isLoading: boolean
  error: string | null
  
  // Path validation state
  pathValidation: PathValidationResult | null
  isValidatingPath: boolean
  
  // Actions
  updateConfig: (updates: Partial<StorageConfig>) => Promise<void>
  validatePath: (path: string) => Promise<void>
  browseForPath: () => Promise<string | null>
  moveData: (fromPath: string, toPath: string) => Promise<void>
  refreshConfig: () => Promise<void>
  refreshStats: () => Promise<void>
}

export function useStorageConfig(): UseStorageConfigReturn {
  const [config, setConfig] = useState<StorageConfig | null>(null)
  const [stats, setStats] = useState<StorageStats | null>(null)
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  
  const [pathValidation, setPathValidation] = useState<PathValidationResult | null>(null)
  const [isValidatingPath, setIsValidatingPath] = useState(false)
  
  // Load initial config and stats
  const refreshConfig = useCallback(async () => {
    try {
      setIsLoading(true)
      setError(null)
      const configData = await configApi.getStorageConfig()
      setConfig(configData)
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to load configuration")
    } finally {
      setIsLoading(false)
    }
  }, [])
  
  const refreshStats = useCallback(async () => {
    try {
      const statsData = await configApi.getStorageStats()
      setStats(statsData)
    } catch (err) {
      console.error("Failed to load storage stats:", err)
      // Don't set error for stats failure as it's not critical
    }
  }, [])
  
  // Update configuration
  const updateConfig = useCallback(async (updates: Partial<StorageConfig>) => {
    if (!config) return
    
    try {
      setIsLoading(true)
      setError(null)
      const updatedConfig = await configApi.updateStorageConfig(updates)
      setConfig(updatedConfig)
      
      // Refresh stats after config update
      await refreshStats()
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to update configuration")
      throw err
    } finally {
      setIsLoading(false)
    }
  }, [config, refreshStats])
  
  // Validate path
  const validatePath = useCallback(async (path: string) => {
    if (!path.trim()) {
      setPathValidation(null)
      return
    }
    
    try {
      setIsValidatingPath(true)
      const validation = await configApi.validatePath(path)
      setPathValidation(validation)
    } catch (err) {
      console.error("Path validation failed:", err)
      setPathValidation({
        isValid: false,
        exists: false,
        isWritable: false,
        isDirectory: false,
        issues: ["Failed to validate path"],
        warnings: []
      })
    } finally {
      setIsValidatingPath(false)
    }
  }, [])
  
  // Browse for path
  const browseForPath = useCallback(async (): Promise<string | null> => {
    try {
      const selectedPath = await configApi.browseForDirectory()
      
      // If a path was selected, validate it
      if (selectedPath) {
        await validatePath(selectedPath)
      }
      
      return selectedPath
    } catch (err) {
      console.error("Failed to browse for path:", err)
      return null
    }
  }, [validatePath])
  
  // Move data
  const moveData = useCallback(async (fromPath: string, toPath: string) => {
    try {
      setIsLoading(true)
      setError(null)
      await configApi.moveData(fromPath, toPath)
      
      // Refresh stats after moving data
      await refreshStats()
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to move data")
      throw err
    } finally {
      setIsLoading(false)
    }
  }, [refreshStats])
  
  // Load initial data
  useEffect(() => {
    refreshConfig()
    refreshStats()
  }, [refreshConfig, refreshStats])
  
  return {
    // State
    config,
    stats,
    isLoading,
    error,
    pathValidation,
    isValidatingPath,
    
    // Actions
    updateConfig,
    validatePath,
    browseForPath,
    moveData,
    refreshConfig,
    refreshStats
  }
}