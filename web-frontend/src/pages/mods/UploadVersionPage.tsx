import { useState, useEffect } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { getGitHubReleases, createVersionFromGitHub } from '@/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { useToast } from '@/hooks/use-toast'
import { ArrowLeft, GitBranch, Download, Loader2 } from 'lucide-react'

interface GitHubRelease {
  tag_name: string
  name: string
  body: string
  draft: boolean
  prerelease: boolean
  published_at: string
  assets: {
    name: string
    size: number
    browser_download_url: string
    content_type: string
  }[]
}

export function UploadVersionPage() {
  const { id: modID } = useParams<{ id: string }>()
  const [loading, setLoading] = useState(true)
  const [creating, setCreating] = useState<string | null>(null)
  const [releases, setReleases] = useState<GitHubRelease[]>([])
  const [error, setError] = useState<string | null>(null)
  const navigate = useNavigate()
  const { toast } = useToast()

  useEffect(() => {
    fetchReleases()
  }, [modID])

  const fetchReleases = async () => {
    setLoading(true)
    setError(null)
    try {
      const data = await getGitHubReleases(modID!)
      setReleases(data)
    } catch (e: any) {
      setError(e.message)
    } finally {
      setLoading(false)
    }
  }

  const handleImport = async (tag: string) => {
    setCreating(tag)
    try {
      await createVersionFromGitHub(modID!, tag)
      toast({ title: `Version ${tag} imported successfully` })
      navigate(`/mods/${modID}/edit`)
    } catch (e: any) {
      toast({
        variant: 'destructive',
        title: 'Import failed',
        description: e.message,
      })
    } finally {
      setCreating(null)
    }
  }

  const formatBytes = (bytes: number) => {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
  }

  return (
    <div className="max-w-3xl mx-auto space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" onClick={() => navigate(`/mods/${modID}/edit`)}>
          <ArrowLeft className="h-5 w-5" />
        </Button>
        <h1 className="text-3xl font-bold tracking-tight">Import from GitHub Release</h1>
      </div>

      {loading && (
        <Card>
          <CardContent className="flex items-center justify-center py-12">
            <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
            <span className="ml-3 text-muted-foreground">Loading releases...</span>
          </CardContent>
        </Card>
      )}

      {error && (
        <Card>
          <CardContent className="py-8">
            <p className="text-destructive text-center">{error}</p>
            <p className="text-sm text-muted-foreground text-center mt-2">
              Make sure the mod has a linked GitHub repository in the mod settings.
            </p>
          </CardContent>
        </Card>
      )}

      {!loading && !error && releases.length === 0 && (
        <Card>
          <CardContent className="py-8">
            <p className="text-muted-foreground text-center">No releases found for this repository.</p>
          </CardContent>
        </Card>
      )}

      {!loading && !error && releases.length > 0 && (
        <div className="space-y-3">
          {releases.map((release) => (
            <Card key={release.tag_name}>
              <CardHeader className="pb-3">
                <div className="flex items-start justify-between">
                  <div>
                    <CardTitle className="flex items-center gap-2 text-lg">
                      <GitBranch className="h-4 w-4" />
                      {release.name || release.tag_name}
                      {release.prerelease && (
                        <span className="text-xs bg-yellow-100 dark:bg-yellow-900 text-yellow-800 dark:text-yellow-200 px-2 py-0.5 rounded-full">
                          Pre-release
                        </span>
                      )}
                      {release.draft && (
                        <span className="text-xs bg-gray-100 dark:bg-gray-800 text-gray-600 dark:text-gray-300 px-2 py-0.5 rounded-full">
                          Draft
                        </span>
                      )}
                    </CardTitle>
                    <CardDescription className="mt-1">
                      Tag: <code className="text-xs">{release.tag_name}</code>
                      {release.published_at && (
                        <> · {new Date(release.published_at).toLocaleDateString()}</>
                      )}
                    </CardDescription>
                  </div>
                  <Button
                    size="sm"
                    onClick={() => handleImport(release.tag_name)}
                    disabled={creating !== null}
                  >
                    {creating === release.tag_name ? (
                      <Loader2 className="h-4 w-4 animate-spin mr-1" />
                    ) : (
                      <Download className="h-4 w-4 mr-1" />
                    )}
                    Import
                  </Button>
                </div>
              </CardHeader>
              {release.assets.length > 0 && (
                <CardContent className="pt-0">
                  <div className="text-sm text-muted-foreground">
                    {release.assets.length} asset{release.assets.length !== 1 ? 's' : ''}:
                    <ul className="mt-1 space-y-0.5">
                      {release.assets.map((asset, i) => (
                        <li key={i} className="flex items-center gap-2">
                          <span className="font-mono text-xs">{asset.name}</span>
                          <span className="text-xs">({formatBytes(asset.size)})</span>
                        </li>
                      ))}
                    </ul>
                  </div>
                </CardContent>
              )}
            </Card>
          ))}
        </div>
      )}
    </div>
  )
}
