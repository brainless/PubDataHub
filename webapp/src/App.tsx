import { Router, Routes, Route } from 'solid-app-router'
import Navigation from './components/Navigation'
import Homepage from './components/Homepage'
import DataSourceDetail from './components/DataSourceDetail'
import './App.css'

function App() {
  return (
    <div class="min-h-screen bg-gray-50">
      <Router>
        <Navigation />
        <Routes>
          <Route path="/" element={<Homepage />} />
          <Route path="/sources/:sourceName" element={<DataSourceDetail />} />
        </Routes>
      </Router>
    </div>
  )
}

export default App
