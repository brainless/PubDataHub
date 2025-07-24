import { CheckCircle, XCircle, AlertCircle, Loader2 } from "lucide-react"

export interface PathValidationResult {
  isValid: boolean
  exists: boolean
  isWritable: boolean
  isDirectory: boolean
  size?: string
  freeSpace?: string
  issues: string[]
  warnings: string[]
}

interface PathValidationProps {
  validation: PathValidationResult | null
  isValidating: boolean
  path: string
}

export function PathValidation({ validation, isValidating, path }: PathValidationProps) {
  if (!path.trim()) {
    return null
  }

  if (isValidating) {
    return (
      <div className="flex items-center gap-2 text-sm text-muted-foreground">
        <Loader2 className="h-4 w-4 animate-spin" />
        <span>Validating path...</span>
      </div>
    )
  }

  if (!validation) {
    return null
  }

  const getStatusIcon = () => {
    if (validation.isValid) {
      return <CheckCircle className="h-4 w-4 text-green-600" />
    } else {
      return <XCircle className="h-4 w-4 text-destructive" />
    }
  }

  return (
    <div className="space-y-3 p-3 border rounded-md bg-muted/30">
      <div className="flex items-center gap-2">
        {getStatusIcon()}
        <span className="text-sm font-medium">
          {validation.isValid ? "Path is valid" : "Path validation failed"}
        </span>
      </div>

      {/* Path details */}
      <div className="grid grid-cols-2 gap-4 text-sm">
        <div className="space-y-1">
          <div className="flex items-center gap-2">
            {validation.exists ? (
              <CheckCircle className="h-3 w-3 text-green-600" />
            ) : (
              <XCircle className="h-3 w-3 text-destructive" />
            )}
            <span>{validation.exists ? "Path exists" : "Path does not exist"}</span>
          </div>
          
          <div className="flex items-center gap-2">
            {validation.isDirectory ? (
              <CheckCircle className="h-3 w-3 text-green-600" />
            ) : (
              <XCircle className="h-3 w-3 text-destructive" />
            )}
            <span>{validation.isDirectory ? "Is a directory" : "Not a directory"}</span>
          </div>
        </div>

        <div className="space-y-1">
          <div className="flex items-center gap-2">
            {validation.isWritable ? (
              <CheckCircle className="h-3 w-3 text-green-600" />
            ) : (
              <XCircle className="h-3 w-3 text-destructive" />
            )}
            <span>{validation.isWritable ? "Writable" : "Not writable"}</span>
          </div>
          
          {validation.freeSpace && (
            <div className="text-muted-foreground">
              Free space: {validation.freeSpace}
            </div>
          )}
        </div>
      </div>

      {/* Issues */}
      {validation.issues.length > 0 && (
        <div className="space-y-1">
          <div className="flex items-center gap-2 text-destructive">
            <XCircle className="h-4 w-4" />
            <span className="font-medium">Issues:</span>
          </div>
          <ul className="list-disc list-inside text-sm text-destructive space-y-1 ml-6">
            {validation.issues.map((issue, index) => (
              <li key={index}>{issue}</li>
            ))}
          </ul>
        </div>
      )}

      {/* Warnings */}
      {validation.warnings.length > 0 && (
        <div className="space-y-1">
          <div className="flex items-center gap-2 text-amber-600">
            <AlertCircle className="h-4 w-4" />
            <span className="font-medium">Warnings:</span>
          </div>
          <ul className="list-disc list-inside text-sm text-amber-600 space-y-1 ml-6">
            {validation.warnings.map((warning, index) => (
              <li key={index}>{warning}</li>
            ))}
          </ul>
        </div>
      )}
    </div>
  )
}