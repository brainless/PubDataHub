import { Button } from "@/components/ui/button"

export function TopNavigation() {
  return (
    <header className="h-16 border-b bg-card px-6 flex items-center justify-between">
      <div className="flex items-center gap-4">
        <h1 className="text-lg font-semibold">PubDataHub</h1>
      </div>
      <div className="flex items-center gap-2">
        <Button variant="outline" size="sm">
          Help
        </Button>
      </div>
    </header>
  )
}