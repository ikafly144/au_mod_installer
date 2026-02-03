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
    return <div className="flex items-center justify-center h-64">Loading mod details...</div>
  }

  return (
    <div className="max-w-4xl mx-auto space-y-8">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" onClick={() => navigate('/mods')}>
          <ArrowLeft className="h-5 w-5" />
        </Button>
        <h1 className="text-3xl font-bold tracking-tight">Edit Mod: {formData.name}</h1>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-8">
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
              <div className="pt-4">
                <Button type="submit" disabled={saving} className="w-full">
                  {saving ? 'Saving...' : 'Update Mod Details'}
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle>Versions</CardTitle>
            <CardDescription>Manage files and versions for this mod.</CardDescription>
          </CardHeader>
          <CardContent>
            <VersionList modID={id!} />
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
