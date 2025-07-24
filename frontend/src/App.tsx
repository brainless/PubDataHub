import { BrowserRouter, Routes, Route } from "react-router-dom"
import { AppLayout } from "./components/layout/AppLayout"
import { Home } from "./pages/Home"
import { DataSources } from "./pages/DataSources"
import { Downloads } from "./pages/Downloads"
import { Settings } from "./pages/Settings"

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<AppLayout />}>
          <Route index element={<Home />} />
          <Route path="data-sources" element={<DataSources />} />
          <Route path="downloads" element={<Downloads />} />
          <Route path="settings" element={<Settings />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}

export default App