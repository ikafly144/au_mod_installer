import React, { useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { createVersion, uploadFile } from '@/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { useToast } from '@/hooks/use-toast'
import { ArrowLeft, Upload } from 'lucide-react'

export function UploadVersionPage() {
  const { id: modID } = useParams<{ id: string }>()
  const [loading, setLoading] = useState(false)
  const [versionID, setVersionID] = useState('')
  const [file, setFile] = useState<File | null>(null)
  const navigate = useNavigate()
  const { toast } = useToast()

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!file) {
      toast({ variant: 'destructive', title: 'File required' })
      return
    }

    setLoading(true)
    try {
      const url = await uploadFile(file)
      
      await createVersion(modID!, {
        id: versionID,
        mod_id: modID,
        created_at: new Date().toISOString(),
        files: [
          {
            url: url,
            file_type: "zip",
            compatible: ["windows", "linux"]
          }
        ]
      })

      toast({ title: 'Version uploaded successfully' })
      navigate(`/mods/${modID}/edit`)
    } catch (e: any) {
      toast({
        variant: 'destructive',
        title: 'Upload failed',
        description: e.message,
      })
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" onClick={() => navigate(`/mods/${modID}/edit`)}>
          <ArrowLeft className="h-5 w-5" />
        </Button>
        <h1 className="text-3xl font-bold tracking-tight">Upload New Version</h1>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Version Details</CardTitle>
          <CardDescription>Provide version information and upload the mod file.</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="grid gap-2">
              <Label htmlFor="ver-id">Version ID (e.g. v1.0.0)</Label>
              <Input
                id="ver-id"
                placeholder="v1.0.0"
                value={versionID}
                onChange={(e) => setVersionID(e.target.value)}
                required
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="file">Mod File (.zip, .rar, .7z)</Label>
              <div className="grid w-full items-center gap-1.5">
                <Input
                  id="file"
                  type="file"
                  accept=".zip,.rar,.7z"
                  onChange={(e) => setFile(e.target.files?.[0] || null)}
                  required
                  className="cursor-pointer"
                />
              </div>
            </div>
            <div className="pt-4 flex justify-end gap-2">
              <Button type="button" variant="outline" onClick={() => navigate(`/mods/${modID}/edit`)}>
                Cancel
              </Button>
              <Button type="submit" disabled={loading}>
                <Upload className="h-4 w-4 mr-2" />
                {loading ? 'Uploading...' : 'Upload & Create'}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
