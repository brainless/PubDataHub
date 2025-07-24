import { Button } from "@/components/ui/button"

export function Downloads() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Downloads</h1>
        <p className="text-muted-foreground">
          Manage your downloaded datasets and files.
        </p>
      </div>
      
      <div className="border rounded-lg p-8 text-center">
        <div className="text-4xl mb-4">📥</div>
        <h2 className="text-xl font-semibold mb-2">No downloads yet</h2>
        <p className="text-muted-foreground mb-4">
          Start by exploring data sources and downloading datasets.
        </p>
        <Button>Browse Data Sources</Button>
      </div>
      
      <div className="space-y-4">
        <h2 className="text-lg font-semibold">Download History</h2>
        <div className="border rounded-lg p-4 text-center text-muted-foreground">
          Your download history will appear here.
        </div>
      </div>
    </div>
  )
}