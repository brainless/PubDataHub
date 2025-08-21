import { Router, Route } from '@solidjs/router'
import Navigation from './components/Navigation'
import Homepage from './components/Homepage'
import DataSourceDetail from './components/DataSourceDetail'
import Settings from './components/Settings'
import './App.css'

function Root(props: any) {
  return (
    <div class="min-h-screen bg-gray-50">
      <Navigation />
      {props.children}
    </div>
  )
}

function App() {
  return (
    <Router root={Root}>
      <Route path="/" component={Homepage} />
      <Route path="/sources/:sourceName" component={DataSourceDetail} />
      <Route path="/settings" component={Settings} />
    </Router>
  )
}

export default App
