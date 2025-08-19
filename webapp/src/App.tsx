import { Router, Route } from '@solidjs/router'
import Navigation from './components/Navigation'
import Homepage from './components/Homepage'
import DataSourceDetail from './components/DataSourceDetail'
import './App.css'

function App() {
  return (
    <div class="min-h-screen bg-gray-50">
      <Router>
        <Route path="/" component={() => (
          <div>
            <Navigation />
            <Homepage />
          </div>
        )} />
        <Route path="/sources/:sourceName" component={() => (
          <div>
            <Navigation />
            <DataSourceDetail />
          </div>
        )} />
      </Router>
    </div>
  )
}

export default App
