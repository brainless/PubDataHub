import { Button } from "@/components/ui/button"

export function DataSources() {
  const dataSources = [
    {
      name: "Open Data Portal",
      description: "Government and public sector datasets",
      status: "Available",
      icon: "🏛️"
    },
    {
      name: "Research Datasets",
      description: "Academic and scientific research data",
      status: "Available", 
      icon: "🔬"
    },
    {
      name: "Financial Data",
      description: "Market and economic indicators",
      status: "Coming Soon",
      icon: "💰"
    }
  ]

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Data Sources</h1>
        <p className="text-muted-foreground">
          Browse and connect to available public data sources.
        </p>
      </div>
      
      <div className="grid gap-4">
        {dataSources.map((source, index) => (
          <div key={index} className="border rounded-lg p-4 flex items-center justify-between">
            <div className="flex items-center gap-4">
              <span className="text-2xl">{source.icon}</span>
              <div>
                <h3 className="font-semibold">{source.name}</h3>
                <p className="text-sm text-muted-foreground">{source.description}</p>
                <span className={`text-xs px-2 py-1 rounded-full ${
                  source.status === "Available" 
                    ? "bg-green-100 text-green-800" 
                    : "bg-yellow-100 text-yellow-800"
                }`}>
                  {source.status}
                </span>
              </div>
            </div>
            <Button 
              variant={source.status === "Available" ? "default" : "outline"}
              disabled={source.status !== "Available"}
            >
              {source.status === "Available" ? "Connect" : "Coming Soon"}
            </Button>
          </div>
        ))}
      </div>
    </div>
  )
}