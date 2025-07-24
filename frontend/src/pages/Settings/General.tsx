import { Button } from "@/components/ui/button"

export function General() {
  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-bold">General Settings</h2>
        <p className="text-muted-foreground">
          Manage your general application preferences and behavior.
        </p>
      </div>
      
      <div className="space-y-4">
        <div className="border rounded-lg p-4">
          <h3 className="text-lg font-semibold mb-2">Appearance</h3>
          <p className="text-sm text-muted-foreground mb-4">
            Customize the look and feel of the application.
          </p>
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <span className="text-sm">Theme</span>
              <select className="px-3 py-1 border rounded text-sm">
                <option>Light</option>
                <option>Dark</option>
                <option>System</option>
              </select>
            </div>
            <div className="flex items-center justify-between">
              <span className="text-sm">Language</span>
              <select className="px-3 py-1 border rounded text-sm">
                <option>English</option>
                <option>Spanish</option>
                <option>French</option>
              </select>
            </div>
          </div>
        </div>
        
        <div className="border rounded-lg p-4">
          <h3 className="text-lg font-semibold mb-2">Notifications</h3>
          <p className="text-sm text-muted-foreground mb-4">
            Configure how you receive notifications about downloads and updates.
          </p>
          <div className="space-y-3">
            <label className="flex items-center gap-2">
              <input type="checkbox" defaultChecked />
              <span className="text-sm">Download completion notifications</span>
            </label>
            <label className="flex items-center gap-2">
              <input type="checkbox" defaultChecked />
              <span className="text-sm">Error notifications</span>
            </label>
            <label className="flex items-center gap-2">
              <input type="checkbox" />
              <span className="text-sm">Weekly data source updates</span>
            </label>
          </div>
        </div>
        
        <div className="border rounded-lg p-4">
          <h3 className="text-lg font-semibold mb-2">Performance</h3>
          <p className="text-sm text-muted-foreground mb-4">
            Adjust performance settings for better experience.
          </p>
          <div className="space-y-3">
            <div className="flex items-center justify-between">
              <span className="text-sm">Concurrent downloads</span>
              <select className="px-3 py-1 border rounded text-sm">
                <option>1</option>
                <option>2</option>
                <option>3</option>
                <option>5</option>
              </select>
            </div>
            <label className="flex items-center gap-2">
              <input type="checkbox" defaultChecked />
              <span className="text-sm">Enable download resume</span>
            </label>
          </div>
        </div>
      </div>
      
      <div className="flex gap-2">
        <Button>Save Changes</Button>
        <Button variant="outline">Reset to Defaults</Button>
      </div>
    </div>
  )
}