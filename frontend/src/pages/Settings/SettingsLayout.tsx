import { Outlet } from "react-router-dom"
import { SettingsNavigation } from "./SettingsNavigation"
import { SettingsBreadcrumb } from "@/components/settings/SettingsBreadcrumb"

export function SettingsLayout() {
  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-bold">Settings</h1>
        <p className="text-muted-foreground">
          Configure your PubDataHub preferences and application settings.
        </p>
      </div>
      
      <SettingsBreadcrumb />
      
      <div className="flex gap-6">
        <aside className="w-64 space-y-2">
          <SettingsNavigation />
        </aside>
        
        <main className="flex-1 space-y-6">
          <Outlet />
        </main>
      </div>
    </div>
  )
}