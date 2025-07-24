import { Button } from "@/components/ui/button"

export function Home() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-3xl font-bold">Welcome to PubDataHub</h1>
        <p className="text-lg text-muted-foreground">
          Your gateway to discovering, downloading, and managing public datasets.
        </p>
      </div>
      
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        <div className="border rounded-lg p-6">
          <h2 className="text-xl font-semibold mb-2">📊 Data Sources</h2>
          <p className="text-muted-foreground mb-4">
            Connect to various public data APIs and repositories.
          </p>
          <Button>Explore Sources</Button>
        </div>
        
        <div className="border rounded-lg p-6">
          <h2 className="text-xl font-semibold mb-2">⬇️ Downloads</h2>
          <p className="text-muted-foreground mb-4">
            Manage your downloaded datasets and files.
          </p>
          <Button>View Downloads</Button>
        </div>
        
        <div className="border rounded-lg p-6">
          <h2 className="text-xl font-semibold mb-2">⚙️ Settings</h2>
          <p className="text-muted-foreground mb-4">
            Configure your preferences and API connections.
          </p>
          <Button>Open Settings</Button>
        </div>
      </div>
      
      <div className="border rounded-lg p-6">
        <h2 className="text-xl font-semibold mb-4">Recent Activity</h2>
        <p className="text-muted-foreground">
          No recent activity. Start by exploring available data sources.
        </p>
      </div>
    </div>
  )
}