import { Button } from "@/components/ui/button"

export function DataStorage() {
  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-bold">Data Storage</h2>
        <p className="text-muted-foreground">
          Configure where your downloaded data is stored and manage storage preferences.
        </p>
      </div>
      
      <div className="space-y-4">
        <div className="border rounded-lg p-4">
          <h3 className="text-lg font-semibold mb-2">Default Download Location</h3>
          <p className="text-sm text-muted-foreground mb-4">
            Choose where downloaded files will be saved by default.
          </p>
          <div className="flex items-center gap-2">
            <code className="flex-1 px-3 py-2 bg-muted rounded-md text-sm">
              ~/Downloads/PubDataHub
            </code>
            <Button variant="outline">Change</Button>
          </div>
        </div>
        
        <div className="border rounded-lg p-4">
          <h3 className="text-lg font-semibold mb-2">Storage Limits</h3>
          <p className="text-sm text-muted-foreground mb-4">
            Set maximum storage limits for downloaded data.
          </p>
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <span className="text-sm">Maximum storage per dataset</span>
              <div className="flex items-center gap-2">
                <input 
                  type="number" 
                  defaultValue="1" 
                  className="w-20 px-2 py-1 border rounded text-sm"
                />
                <span className="text-sm">GB</span>
              </div>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm">Total storage limit</span>
              <div className="flex items-center gap-2">
                <input 
                  type="number" 
                  defaultValue="10" 
                  className="w-20 px-2 py-1 border rounded text-sm"
                />
                <span className="text-sm">GB</span>
              </div>
            </div>
          </div>
        </div>
        
        <div className="border rounded-lg p-4">
          <h3 className="text-lg font-semibold mb-2">Cleanup Options</h3>
          <p className="text-sm text-muted-foreground mb-4">
            Automatically manage storage by cleaning up old downloads.
          </p>
          <div className="space-y-3">
            <label className="flex items-center gap-2">
              <input type="checkbox" defaultChecked />
              <span className="text-sm">Auto-delete downloads older than 30 days</span>
            </label>
            <label className="flex items-center gap-2">
              <input type="checkbox" />
              <span className="text-sm">Compress old downloads to save space</span>
            </label>
            <Button variant="outline" size="sm">Clean up now</Button>
          </div>
        </div>
      </div>
      
      <div className="flex gap-2">
        <Button>Save Changes</Button>
        <Button variant="outline">Reset to Defaults</Button>
      </div>
    </div>
  )
}