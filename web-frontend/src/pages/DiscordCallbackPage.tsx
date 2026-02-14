import { useEffect, useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { exchangeDiscordCode } from '@/api'
import { setSession } from '@/auth'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

interface DiscordCallbackPageProps {
  onLogin: () => void
}

export default function DiscordCallbackPage({ onLogin }: DiscordCallbackPageProps) {
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const code = searchParams.get('code')
    if (!code) {
      setError('No authorization code received from Discord')
      return
    }

    exchangeDiscordCode(code)
      .then((resp) => {
        setSession(resp.token, resp.user)
        onLogin()
        navigate('/', { replace: true })
      })
      .catch((e) => {
        setError(e.message || 'Failed to authenticate with Discord')
      })
  }, [searchParams, navigate, onLogin])

  if (error) {
    return (
      <div className="flex items-center justify-center min-h-screen bg-slate-50 dark:bg-slate-950 p-4">
        <Card className="w-full max-w-md">
          <CardHeader>
            <CardTitle className="text-xl text-center text-destructive">Authentication Failed</CardTitle>
          </CardHeader>
          <CardContent className="text-center">
            <p className="text-muted-foreground mb-4">{error}</p>
            <a href="/login" className="text-primary hover:underline">
              Return to login
            </a>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="flex items-center justify-center min-h-screen bg-slate-50 dark:bg-slate-950">
      <div className="text-center">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary mx-auto mb-4"></div>
        <p className="text-muted-foreground">Authenticating with Discord...</p>
      </div>
    </div>
  )
}
