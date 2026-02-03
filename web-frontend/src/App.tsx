import { useState, useEffect, useMemo } from 'react'
import { createBrowserRouter, RouterProvider, Navigate } from 'react-router-dom'
import { isLoggedIn } from './auth'
import LoginPage from './components/login-page'
import Dashboard from './components/dashboard'
import { Layout } from './components/layout'
import { Toaster } from './components/ui/toaster'

function App() {
  const [authenticated, setAuthenticated] = useState(false)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    setAuthenticated(isLoggedIn())
    setLoading(false)
  }, [])

  const router = useMemo(() => createBrowserRouter([
    {
      path: "/login",
      element: authenticated ? (
        <Navigate to="/" replace />
      ) : (
        <LoginPage onLogin={() => setAuthenticated(true)} />
      )
    },
    {
      path: "/",
      element: authenticated ? (
        <Layout>
          <Dashboard />
        </Layout>
      ) : (
        <Navigate to="/login" replace />
      )
    },
    {
      path: "/mods",
      element: authenticated ? (
        <Layout>
          <Dashboard />
        </Layout>
      ) : (
        <Navigate to="/login" replace />
      )
    },
    {
      path: "*",
      element: <Navigate to="/" replace />
    }
  ]), [authenticated])

  if (loading) {
    return <div className="flex items-center justify-center h-screen">Loading...</div>
  }

  return (
    <>
      <RouterProvider router={router} />
      <Toaster />
    </>
  )
}

export default App

