import { Outlet } from "react-router-dom"
import { TopNavigation } from "./TopNavigation"
import { LeftNavigation } from "./LeftNavigation"

export function AppLayout() {
  return (
    <div className="h-screen flex flex-col">
      <TopNavigation />
      <div className="flex flex-1 overflow-hidden">
        <LeftNavigation />
        <main className="flex-1 overflow-auto p-6">
          <Outlet />
        </main>
      </div>
    </div>
  )
}