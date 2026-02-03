import { useState, useEffect, useMemo } from 'react'
import { createBrowserRouter, RouterProvider, Navigate } from 'react-router-dom'
import { isLoggedIn } from './auth'
import LoginPage from './components/login-page'
import Dashboard from './components/dashboard'
import { Layout } from './components/layout'
import { CreateModPage } from './pages/mods/CreateModPage'
import { EditModPage } from './pages/mods/EditModPage'
import { UploadVersionPage } from './pages/mods/UploadVersionPage'
import DashboardOverview from './pages/DashboardOverview'

import SettingsPage from './pages/SettingsPage'
import SystemPage from './pages/SystemPage'
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
          <DashboardOverview />
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
      path: "/settings",
      element: authenticated ? (
        <Layout>
          <SettingsPage />
        </Layout>
      ) : (
        <Navigate to="/login" replace />
      )
    },
    {
      path: "/system",
      element: authenticated ? (
        <Layout>
          <SystemPage />
        </Layout>
      ) : (
        <Navigate to="/login" replace />
      )
    },

    {
      path: "/mods/new",
      element: authenticated ? (
        <Layout>
          <CreateModPage />
        </Layout>
      ) : (
        <Navigate to="/login" replace />
      )
    },
        {
      path: "/mods/:id/edit",
      element: authenticated ? (
        <Layout>
          <EditModPage />
        </Layout>
      ) : (
        <Navigate to="/login" replace />
      )
    },
    {
      path: "/mods/:id/versions/new",
      element: authenticated ? (
        <Layout>
          <UploadVersionPage />
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

