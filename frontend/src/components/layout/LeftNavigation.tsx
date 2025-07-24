import { useState } from "react"
import { Link, useLocation } from "react-router-dom"
import { Button } from "@/components/ui/button"
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible"

const navigationItems = [
  { to: "/", label: "Home", icon: "🏠" },
  { to: "/data-sources", label: "Data Sources", icon: "📊" },
  { to: "/downloads", label: "Downloads", icon: "⬇️" },
]

export function LeftNavigation() {
  const [isCollapsed, setIsCollapsed] = useState(false)
  const location = useLocation()

  return (
    <Collapsible
      open={!isCollapsed}
      onOpenChange={(open) => setIsCollapsed(!open)}
      className="border-r bg-card"
    >
      <div className="p-4">
        <CollapsibleTrigger asChild>
          <Button variant="ghost" size="sm" className="w-full justify-start">
            <span className="mr-2">☰</span>
            {!isCollapsed && "Menu"}
          </Button>
        </CollapsibleTrigger>
      </div>
      
      <CollapsibleContent className="space-y-2">
        <nav className="px-4 pb-4">
          <div className="space-y-2">
            {navigationItems.map((item) => (
              <Link
                key={item.to}
                to={item.to}
                className={`flex items-center gap-3 px-3 py-2 rounded-md text-sm transition-colors hover:bg-accent hover:text-accent-foreground ${
                  location.pathname === item.to
                    ? "bg-accent text-accent-foreground"
                    : "text-muted-foreground"
                }`}
              >
                <span>{item.icon}</span>
                {!isCollapsed && item.label}
              </Link>
            ))}
          </div>
          
          <div className="mt-8 pt-4 border-t">
            <Link
              to="/settings"
              className={`flex items-center gap-3 px-3 py-2 rounded-md text-sm transition-colors hover:bg-accent hover:text-accent-foreground ${
                location.pathname === "/settings"
                  ? "bg-accent text-accent-foreground"
                  : "text-muted-foreground"
              }`}
            >
              <span>⚙️</span>
              {!isCollapsed && "Settings"}
            </Link>
          </div>
        </nav>
      </CollapsibleContent>
      
      {isCollapsed && (
        <nav className="px-2 pb-4">
          <div className="space-y-2">
            {navigationItems.map((item) => (
              <Link
                key={item.to}
                to={item.to}
                className={`flex items-center justify-center p-2 rounded-md text-sm transition-colors hover:bg-accent hover:text-accent-foreground ${
                  location.pathname === item.to
                    ? "bg-accent text-accent-foreground"
                    : "text-muted-foreground"
                }`}
                title={item.label}
              >
                <span>{item.icon}</span>
              </Link>
            ))}
          </div>
          
          <div className="mt-8 pt-4 border-t">
            <Link
              to="/settings"
              className={`flex items-center justify-center p-2 rounded-md text-sm transition-colors hover:bg-accent hover:text-accent-foreground ${
                location.pathname === "/settings"
                  ? "bg-accent text-accent-foreground"
                  : "text-muted-foreground"
              }`}
              title="Settings"
            >
              <span>⚙️</span>
            </Link>
          </div>
        </nav>
      )}
    </Collapsible>
  )
}