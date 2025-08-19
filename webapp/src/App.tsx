import { Router, Route } from 'solid-app-router'
import Navigation from './components/Navigation'
import Homepage from './components/Homepage'
import DataSourceDetail from './components/DataSourceDetail'
import Settings from './components/Settings'
import './App.css'

function App() {
  return (
    <div class="min-h-screen bg-gray-50">
      <Router>
        <Navigation />
        <Route path="/" component={Homepage} />
        <Route path="/sources/:sourceName" component={DataSourceDetail} />
        <Route path="/settings" component={Settings} />
      </Router>
    </div>
  )
}

export default App
