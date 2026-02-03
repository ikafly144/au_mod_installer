import React, { useState } from 'react'
import { createVersion, uploadFile } from '@/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { useToast } from '@/hooks/use-toast'

interface VersionDialogProps {
  modID: string
  onSuccess: () => void
  children: React.ReactNode
}

export function VersionDialog({ modID, onSuccess, children }: VersionDialogProps) {
  const [open, setOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const [versionID, setVersionID] = useState('')
  const [file, setFile] = useState<File | null>(null)
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
      
      await createVersion(modID, {
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
      setOpen(false)
      setVersionID('')
      setFile(null)
      onSuccess()
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
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        {children}
      </DialogTrigger>
      <DialogContent className="sm:max-w-[425px]">
        <form onSubmit={handleSubmit}>
          <DialogHeader>
            <DialogTitle>Upload New Version</DialogTitle>
            <DialogDescription>
              Provide version details and upload the mod file.
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="ver-id">Version ID (e.g. v1.0.0)</Label>
              <Input
                id="ver-id"
                value={versionID}
                onChange={(e) => setVersionID(e.target.value)}
                placeholder="v1.0.0"
                required
              />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="file">Mod File (.zip, .rar)</Label>
              <Input
                id="file"
                type="file"
                accept=".zip,.rar,.7z"
                onChange={(e) => setFile(e.target.files?.[0] || null)}
                required
              />
            </div>
          </div>
          <DialogFooter>
            <Button type="submit" disabled={loading}>
              {loading ? 'Uploading...' : 'Upload & Create'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
