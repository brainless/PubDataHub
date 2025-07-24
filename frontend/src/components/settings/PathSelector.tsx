import { useState } from "react"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"

interface PathSelectorProps {
  label: string
  description?: string
  currentPath: string
  onPathChange: (path: string) => void
  onBrowse?: () => void
  isLoading?: boolean
  error?: string
}

export function PathSelector({
  label,
  description,
  currentPath,
  onPathChange,
  onBrowse,
  isLoading = false,
  error
}: PathSelectorProps) {
  const [inputPath, setInputPath] = useState(currentPath)
  const [isEditing, setIsEditing] = useState(false)

  const handleSave = () => {
    onPathChange(inputPath)
    setIsEditing(false)
  }

  const handleCancel = () => {
    setInputPath(currentPath)
    setIsEditing(false)
  }

  const handleBrowse = () => {
    onBrowse?.()
  }

  return (
    <div className="space-y-3">
      <div>
        <h3 className="text-lg font-semibold">{label}</h3>
        {description && (
          <p className="text-sm text-muted-foreground">{description}</p>
        )}
      </div>
      
      <div className="space-y-2">
        <div className="flex items-center gap-2">
          {isEditing ? (
            <>
              <Input
                value={inputPath}
                onChange={(e) => setInputPath(e.target.value)}
                placeholder="Enter path..."
                className={error ? "border-destructive" : ""}
                disabled={isLoading}
              />
              <Button
                onClick={handleSave}
                disabled={isLoading || inputPath === currentPath || !inputPath.trim()}
                size="sm"
              >
                Save
              </Button>
              <Button
                onClick={handleCancel}
                variant="outline"
                size="sm"
                disabled={isLoading}
              >
                Cancel
              </Button>
            </>
          ) : (
            <>
              <code className="flex-1 px-3 py-2 bg-muted rounded-md text-sm break-all">
                {currentPath || "No path selected"}
              </code>
              <Button
                onClick={() => setIsEditing(true)}
                variant="outline"
                size="sm"
                disabled={isLoading}
              >
                Change
              </Button>
              {onBrowse && (
                <Button
                  onClick={handleBrowse}
                  variant="outline"
                  size="sm"
                  disabled={isLoading}
                >
                  Browse
                </Button>
              )}
            </>
          )}
        </div>
        
        {error && (
          <p className="text-sm text-destructive">{error}</p>
        )}
        
        {isLoading && (
          <p className="text-sm text-muted-foreground">Processing...</p>
        )}
      </div>
    </div>
  )
}