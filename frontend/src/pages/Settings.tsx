import { Button } from "@/components/ui/button"

export function Settings() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Settings</h1>
        <p className="text-muted-foreground">
          Configure your PubDataHub preferences and settings.
        </p>
      </div>
      
      <div className="space-y-4">
        <div className="border rounded-lg p-4">
          <h2 className="text-lg font-semibold mb-2">General Settings</h2>
          <p className="text-sm text-muted-foreground mb-4">
            Manage your general application preferences.
          </p>
          <Button variant="outline">Configure</Button>
        </div>
        
        <div className="border rounded-lg p-4">
          <h2 className="text-lg font-semibold mb-2">Data Sources</h2>
          <p className="text-sm text-muted-foreground mb-4">
            Manage your connected data sources and API keys.
          </p>
          <Button variant="outline">Manage Sources</Button>
        </div>
        
        <div className="border rounded-lg p-4">
          <h2 className="text-lg font-semibold mb-2">Download Settings</h2>
          <p className="text-sm text-muted-foreground mb-4">
            Configure download preferences and storage options.
          </p>
          <Button variant="outline">Configure Downloads</Button>
        </div>
      </div>
    </div>
  )
}