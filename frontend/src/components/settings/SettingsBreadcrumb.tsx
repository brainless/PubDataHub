import { Link, useLocation } from "react-router-dom"

const settingsBreadcrumbMap = {
  "/settings": "General",
  "/settings/data-storage": "Data Storage",
  "/settings/data-sources": "Data Sources", 
  "/settings/downloads": "Downloads",
  "/settings/security": "Security"
}

export function SettingsBreadcrumb() {
  const location = useLocation()
  
  const breadcrumbItems = [
    { label: "Home", to: "/" },
    { label: "Settings", to: "/settings" }
  ]
  
  // Add current settings page if not general settings
  if (location.pathname !== "/settings") {
    const currentPageLabel = settingsBreadcrumbMap[location.pathname as keyof typeof settingsBreadcrumbMap]
    if (currentPageLabel) {
      breadcrumbItems.push({
        label: currentPageLabel,
        to: location.pathname
      })
    }
  }
  
  return (
    <nav className="flex items-center gap-2 text-sm text-muted-foreground">
      {breadcrumbItems.map((item, index) => (
        <div key={item.to} className="flex items-center gap-2">
          {index < breadcrumbItems.length - 1 ? (
            <Link 
              to={item.to} 
              className="hover:text-foreground transition-colors"
            >
              {item.label}
            </Link>
          ) : (
            <span className="text-foreground font-medium">{item.label}</span>
          )}
          {index < breadcrumbItems.length - 1 && (
            <span className="text-muted-foreground">/</span>
          )}
        </div>
      ))}
    </nav>
  )
}