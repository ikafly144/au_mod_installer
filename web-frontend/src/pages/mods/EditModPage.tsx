import React, { useState, useEffect } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { getMod, updateMod } from '@/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { useToast } from '@/hooks/use-toast'
import { ArrowLeft } from 'lucide-react'
import { VersionList } from '@/components/version-list'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Skeleton } from '@/components/ui/skeleton'

export function EditModPage() {

  const { id } = useParams<{ id: string }>()
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const navigate = useNavigate()
  const { toast } = useToast()

  const [formData, setFormData] = useState({
    id: '',
    name: '',
    author: '',
    description: '',
    website: '',
    github_repo: '',
    type: 'mod'
  })

  useEffect(() => {
    if (id) {
      fetchMod()
    }
  }, [id])

  const fetchMod = async () => {
    try {
      const mod = await getMod(id!)
      setFormData({
        id: mod.id || '',
        name: mod.name || '',
        author: mod.author || '',
        description: mod.description || '',
        website: mod.website || '',
        github_repo: mod.github_repo || '',
        type: mod.type || 'mod'
      })
    } catch (e: any) {
      toast({
        variant: 'destructive',
        title: 'Failed to load mod',
        description: e.message,
      })
      navigate('/mods')
    } finally {
      setLoading(false)
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setSaving(true)
    try {
      const data = {
        ...formData,
        updated_at: new Date().toISOString()
      }

      await updateMod(id!, data)
      toast({ title: 'Mod updated successfully' })
    } catch (e: any) {
      toast({
        variant: 'destructive',
        title: 'Update failed',
        description: e.message,
      })
    } finally {
      setSaving(false)
    }
  }

    if (loading) {
    return (
      <div className="max-w-4xl mx-auto space-y-8">
        <div className="flex items-center gap-4">
          <Skeleton className="h-10 w-10 rounded-full" />
          <Skeleton className="h-10 w-64" />
        </div>
        <div className="space-y-4">
          <Skeleton className="h-10 w-[200px]" />
          <Skeleton className="h-[400px] w-full" />
        </div>
      </div>
    )
  }

  return (
    <div className="max-w-4xl mx-auto space-y-8">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" onClick={() => navigate('/mods')}>
          <ArrowLeft className="h-5 w-5" />
        </Button>
        <h1 className="text-3xl font-bold tracking-tight">Edit Mod: {formData.name}</h1>
      </div>

      <Tabs defaultValue="details" className="w-full">
        <TabsList className="grid w-full max-w-[400px] grid-cols-2">
          <TabsTrigger value="details">Mod Details</TabsTrigger>
          <TabsTrigger value="versions">Versions</TabsTrigger>
        </TabsList>
        
        <TabsContent value="details" className="mt-6">
          <Card>
            <CardHeader>
              <CardTitle>Mod Details</CardTitle>
              <CardDescription>Update the core information for this mod.</CardDescription>
            </CardHeader>
            <CardContent>
              <form onSubmit={handleSubmit} className="space-y-4">
                <div className="grid gap-2">
                  <Label htmlFor="id">ID (Read-only)</Label>
                  <Input id="id" value={formData.id} disabled />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="name">Name</Label>
                  <Input
                    id="name"
                    value={formData.name}
                    onChange={(e) => setFormData({ ...formData, name: e.target.value })}
                    required
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="author">Author</Label>
                  <Input
                    id="author"
                    value={formData.author}
                    onChange={(e) => setFormData({ ...formData, author: e.target.value })}
                    required
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="description">Description</Label>
                  <Textarea
                    id="description"
                    className="min-h-[100px]"
                    value={formData.description}
                    onChange={(e) => setFormData({ ...formData, description: e.target.value })}
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="website">Website URL</Label>
                  <Input
                    id="website"
                    type="url"
                    value={formData.website}
                    onChange={(e) => setFormData({ ...formData, website: e.target.value })}
                  />
                </div>
                <div className="grid gap-2">
                  <Label htmlFor="github_repo">GitHub Repository</Label>
                  <Input
                    id="github_repo"
                    placeholder="owner/repo"
                    value={formData.github_repo}
                    onChange={(e) => setFormData({ ...formData, github_repo: e.target.value })}
                  />
                  <p className="text-sm text-muted-foreground">
                    Link a GitHub repository to import versions from releases (e.g. <code>ikafly144/my-mod</code>)
                  </p>
                </div>
                <div className="pt-4">
                  <Button type="submit" disabled={saving} className="w-full">
                    {saving ? 'Saving...' : 'Update Mod Details'}
                  </Button>
                </div>
              </form>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="versions" className="mt-6">
          <Card>
            <CardHeader>
              <CardTitle>Versions</CardTitle>
              <CardDescription>Manage files and versions for this mod.</CardDescription>
            </CardHeader>
            <CardContent>
              <VersionList modID={id!} />
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}

