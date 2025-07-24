import { BrowserRouter, Routes, Route } from "react-router-dom"
import { AppLayout } from "./components/layout/AppLayout"
import { Home } from "./pages/Home"
import { DataSources } from "./pages/DataSources"
import { Downloads } from "./pages/Downloads"
import { SettingsLayout } from "./pages/Settings/SettingsLayout"
import { General } from "./pages/Settings/General"
import { DataStorage } from "./pages/Settings/DataStorage"

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<AppLayout />}>
          <Route index element={<Home />} />
          <Route path="data-sources" element={<DataSources />} />
          <Route path="downloads" element={<Downloads />} />
          <Route path="settings" element={<SettingsLayout />}>
            <Route index element={<General />} />
            <Route path="data-storage" element={<DataStorage />} />
            <Route path="data-sources" element={<div>Data Sources Settings (Coming Soon)</div>} />
            <Route path="downloads" element={<div>Download Settings (Coming Soon)</div>} />
            <Route path="security" element={<div>Security Settings (Coming Soon)</div>} />
          </Route>
        </Route>
      </Routes>
    </BrowserRouter>
  )
}

export default App