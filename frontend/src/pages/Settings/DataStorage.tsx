import { useState, useEffect } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { PathSelector } from "@/components/settings/PathSelector"
import { PathValidation } from "@/components/settings/PathValidation"
import { ConfirmationDialog } from "@/components/settings/ConfirmationDialog"
import { useStorageConfig } from "@/hooks/useStorageConfig"
import { Loader2, HardDrive, FolderOpen, Trash2 } from "lucide-react"

export function DataStorage() {
  const {
    config,
    stats,
    isLoading,
    error,
    pathValidation,
    isValidatingPath,
    updateConfig,
    validatePath,
    browseForPath,
    moveData,
    refreshStats
  } = useStorageConfig()

  const [showMoveConfirmation, setShowMoveConfirmation] = useState(false)
  const [pendingPath, setPendingPath] = useState<string>("")
  const [isMovingData, setIsMovingData] = useState(false)

  // Local form state
  const [maxPerDataset, setMaxPerDataset] = useState(1)
  const [totalLimit, setTotalLimit] = useState(10)
  const [autoDeleteDays, setAutoDeleteDays] = useState(30)
  const [enableCompression, setEnableCompression] = useState(false)

  // Update local state when config loads
  useEffect(() => {
    if (config) {
      setMaxPerDataset(config.maxStoragePerDataset)
      setTotalLimit(config.totalStorageLimit)
      setAutoDeleteDays(config.autoDeleteAfterDays)
      setEnableCompression(config.enableCompression)
    }
  }, [config])

  const handlePathChange = (newPath: string) => {
    if (!config) return

    // If path is different and there might be existing data, show confirmation
    if (newPath !== config.defaultPath && (stats?.numberOfDatasets || 0) > 0) {
      setPendingPath(newPath)
      setShowMoveConfirmation(true)
    } else {
      // No existing data, just update the path
      updateConfig({ defaultPath: newPath })
    }
  }

  const handleConfirmMove = async () => {
    if (!config) return

    try {
      setIsMovingData(true)
      
      // First move the data
      await moveData(config.defaultPath, pendingPath)
      
      // Then update the config
      await updateConfig({ defaultPath: pendingPath })
      
      setShowMoveConfirmation(false)
      setPendingPath("")
    } catch (err) {
      console.error("Failed to move data:", err)
    } finally {
      setIsMovingData(false)
    }
  }

  const handleBrowseForPath = async () => {
    const selectedPath = await browseForPath()
    if (selectedPath) {
      handlePathChange(selectedPath)
    }
  }

  const handleSaveSettings = async () => {
    if (!config) return

    await updateConfig({
      maxStoragePerDataset: maxPerDataset,
      totalStorageLimit: totalLimit,
      autoDeleteAfterDays: autoDeleteDays,
      enableCompression
    })
  }

  const handleCleanupNow = async () => {
    // In a real app, this would trigger cleanup API
    console.log("Triggering cleanup...")
    await refreshStats()
  }

  if (isLoading && !config) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader2 className="h-6 w-6 animate-spin mr-2" />
        <span>Loading storage configuration...</span>
      </div>
    )
  }

  if (error) {
    return (
      <div className="space-y-4">
        <div className="text-destructive">
          Error loading storage configuration: {error}
        </div>
        <Button onClick={() => window.location.reload()}>
          Retry
        </Button>
      </div>
    )
  }

  if (!config) {
    return (
      <div className="text-muted-foreground">
        No configuration available
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-bold">Data Storage</h2>
        <p className="text-muted-foreground">
          Configure where your downloaded data is stored and manage storage preferences.
        </p>
      </div>

      {/* Storage Statistics */}
      {stats && (
        <div className="border rounded-lg p-4 bg-muted/30">
          <h3 className="text-lg font-semibold mb-3 flex items-center gap-2">
            <HardDrive className="h-5 w-5" />
            Storage Statistics
          </h3>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4 text-sm">
            <div>
              <div className="text-muted-foreground">Used Space</div>
              <div className="font-medium">{stats.usedSpace}</div>
            </div>
            <div>
              <div className="text-muted-foreground">Free Space</div>
              <div className="font-medium">{stats.freeSpace}</div>
            </div>
            <div>
              <div className="text-muted-foreground">Datasets</div>
              <div className="font-medium">{stats.numberOfDatasets}</div>
            </div>
            <div>
              <div className="text-muted-foreground">Oldest Download</div>
              <div className="font-medium">{stats.oldestDownload || "None"}</div>
            </div>
          </div>
        </div>
      )}
      
      <div className="space-y-6">
        {/* Default Download Location */}
        <div className="border rounded-lg p-4">
          <PathSelector
            label="Default Download Location"
            description="Choose where downloaded files will be saved by default."
            currentPath={config.defaultPath}
            onPathChange={handlePathChange}
            onBrowse={handleBrowseForPath}
            isLoading={isLoading}
          />
          
          {/* Path Validation */}
          <div className="mt-4">
            <PathValidation
              validation={pathValidation}
              isValidating={isValidatingPath}
              path={config.defaultPath}
            />
          </div>
          
          {config.defaultPath && (
            <div className="mt-3">
              <Button
                variant="outline"
                size="sm"
                onClick={() => validatePath(config.defaultPath)}
                disabled={isValidatingPath}
                className="flex items-center gap-2"
              >
                <FolderOpen className="h-4 w-4" />
                {isValidatingPath ? "Validating..." : "Validate Path"}
              </Button>
            </div>
          )}
        </div>
        
        {/* Storage Limits */}
        <div className="border rounded-lg p-4">
          <h3 className="text-lg font-semibold mb-2">Storage Limits</h3>
          <p className="text-sm text-muted-foreground mb-4">
            Set maximum storage limits for downloaded data.
          </p>
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <label className="text-sm font-medium">Maximum storage per dataset</label>
              <div className="flex items-center gap-2">
                <Input
                  type="number"
                  value={maxPerDataset}
                  onChange={(e) => setMaxPerDataset(Number(e.target.value))}
                  className="w-20 text-sm"
                  min="0.1"
                  step="0.1"
                />
                <span className="text-sm">GB</span>
              </div>
            </div>
            <div className="flex items-center justify-between">
              <label className="text-sm font-medium">Total storage limit</label>
              <div className="flex items-center gap-2">
                <Input
                  type="number"
                  value={totalLimit}
                  onChange={(e) => setTotalLimit(Number(e.target.value))}
                  className="w-20 text-sm"
                  min="1"
                  step="1"
                />
                <span className="text-sm">GB</span>
              </div>
            </div>
          </div>
        </div>
        
        {/* Cleanup Options */}
        <div className="border rounded-lg p-4">
          <h3 className="text-lg font-semibold mb-2">Cleanup Options</h3>
          <p className="text-sm text-muted-foreground mb-4">
            Automatically manage storage by cleaning up old downloads.
          </p>
          <div className="space-y-4">
            <div className="flex items-center justify-between">
              <label className="text-sm font-medium">Auto-delete downloads after</label>
              <div className="flex items-center gap-2">
                <Input
                  type="number"
                  value={autoDeleteDays}
                  onChange={(e) => setAutoDeleteDays(Number(e.target.value))}
                  className="w-20 text-sm"
                  min="1"
                />
                <span className="text-sm">days</span>
              </div>
            </div>
            
            <label className="flex items-center gap-3">
              <input
                type="checkbox"
                checked={enableCompression}
                onChange={(e) => setEnableCompression(e.target.checked)}
                className="rounded"
              />
              <span className="text-sm">Compress old downloads to save space</span>
            </label>
            
            <div className="pt-2">
              <Button
                variant="outline"
                size="sm"
                onClick={handleCleanupNow}
                className="flex items-center gap-2"
              >
                <Trash2 className="h-4 w-4" />
                Clean up now
              </Button>
            </div>
          </div>
        </div>
      </div>
      
      {/* Save/Reset Actions */}
      <div className="flex gap-2">
        <Button onClick={handleSaveSettings} disabled={isLoading}>
          {isLoading ? (
            <>
              <Loader2 className="h-4 w-4 animate-spin mr-2" />
              Saving...
            </>
          ) : (
            "Save Changes"
          )}
        </Button>
        <Button
          variant="outline"
          onClick={() => {
            if (config) {
              setMaxPerDataset(config.maxStoragePerDataset)
              setTotalLimit(config.totalStorageLimit)
              setAutoDeleteDays(config.autoDeleteAfterDays)
              setEnableCompression(config.enableCompression)
            }
          }}
        >
          Reset to Current
        </Button>
      </div>

      {/* Move Data Confirmation Dialog */}
      <ConfirmationDialog
        isOpen={showMoveConfirmation}
        onClose={() => {
          setShowMoveConfirmation(false)
          setPendingPath("")
        }}
        onConfirm={handleConfirmMove}
        title="Move Existing Data?"
        description={`You have ${stats?.numberOfDatasets || 0} datasets in the current location. Do you want to move them to the new path?`}
        confirmText="Move Data"
        isDestructive={false}
        isLoading={isMovingData}
      >
        <div className="space-y-2 text-sm">
          <div>
            <strong>Current path:</strong> {config.defaultPath}
          </div>
          <div>
            <strong>New path:</strong> {pendingPath}
          </div>
          <div className="text-amber-600">
            ⚠️ This operation may take several minutes depending on the amount of data.
          </div>
        </div>
      </ConfirmationDialog>
    </div>
  )
}