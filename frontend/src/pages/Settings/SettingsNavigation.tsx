import { Link, useLocation } from "react-router-dom"

const settingsNavItems = [
  {
    to: "/settings",
    label: "General",
    icon: "⚙️",
    description: "General application preferences"
  },
  {
    to: "/settings/data-storage",
    label: "Data Storage",
    icon: "💾",
    description: "Storage configuration and paths"
  },
  {
    to: "/settings/data-sources",
    label: "Data Sources",
    icon: "📊",
    description: "API connections and credentials"
  },
  {
    to: "/settings/downloads",
    label: "Downloads",
    icon: "⬇️",
    description: "Download preferences and history"
  },
  {
    to: "/settings/security",
    label: "Security",
    icon: "🔒",
    description: "Security and privacy settings"
  }
]

export function SettingsNavigation() {
  const location = useLocation()

  return (
    <nav className="space-y-1">
      {settingsNavItems.map((item) => {
        const isActive = location.pathname === item.to
        
        return (
          <Link
            key={item.to}
            to={item.to}
            className={`block px-3 py-2 rounded-md text-sm transition-colors ${
              isActive
                ? "bg-accent text-accent-foreground"
                : "text-muted-foreground hover:bg-accent hover:text-accent-foreground"
            }`}
          >
            <div className="flex items-center gap-3">
              <span className="text-base">{item.icon}</span>
              <div className="flex-1">
                <div className="font-medium">{item.label}</div>
                <div className="text-xs text-muted-foreground">
                  {item.description}
                </div>
              </div>
            </div>
          </Link>
        )
      })}
    </nav>
  )
}