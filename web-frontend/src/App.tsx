import { useState, useEffect } from 'react'
import { isLoggedIn } from './auth'
import LoginPage from './components/login-page'
import Dashboard from './components/dashboard'
import { Toaster } from './components/ui/toaster'

function App() {
  const [authenticated, setAuthenticated] = useState(false)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    setAuthenticated(isLoggedIn())
    setLoading(false)
  }, [])

  if (loading) {
    return <div className="flex items-center justify-center h-screen">Loading...</div>
  }

  return (
    <>
      {authenticated ? (
        <Dashboard />
      ) : (
        <LoginPage onLogin={() => setAuthenticated(true)} />
      )}
      <Toaster />
    </>
  )
}

export default App