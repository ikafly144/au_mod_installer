import React, { useState, useEffect } from 'react'
import { createMod, updateMod } from '@/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from '@/components/ui/dialog'
import { useToast } from '@/hooks/use-toast'

interface ModDialogProps {
  mod?: any
  onSuccess: () => void
  children: React.ReactNode
}

export function ModDialog({ mod, onSuccess, children }: ModDialogProps) {
  const [open, setOpen] = useState(false)
  const [loading, setLoading] = useState(false)
  const isEdit = !!mod
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
    if (mod && open) {
      setFormData({
        id: mod.id || '',
        name: mod.name || '',
        author: mod.author || '',
        description: mod.description || '',
        website: mod.website || '',
        type: mod.type || 'mod'
      })
    } else if (!mod && open) {
      setFormData({
        id: '',
        name: '',
        author: '',
        description: '',
        website: '',
        type: 'mod'
      })
    }
  }, [mod, open])

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)
    try {
      const data = {
        ...formData,
        updated_at: new Date().toISOString()
      }

      if (isEdit) {
        await updateMod(formData.id, data)
        toast({ title: 'Mod updated successfully' })
      } else {
        await createMod({ ...data, created_at: new Date().toISOString() })
        toast({ title: 'Mod created successfully' })
      }
      setOpen(false)
      onSuccess()
    } catch (e: any) {
      toast({
        variant: 'destructive',
        title: 'Operation failed',
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
            <DialogTitle>{isEdit ? 'Edit Mod' : 'Create New Mod'}</DialogTitle>
            <DialogDescription>
              {isEdit ? 'Update the details for this mod.' : 'Add a new mod to the repository.'}
            </DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            <div className="grid gap-2">
              <Label htmlFor="id">ID</Label>
              <Input
                id="id"
                disabled={isEdit}
                value={formData.id}
                onChange={(e) => setFormData({ ...formData, id: e.target.value })}
                required
              />
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
          </div>
          <DialogFooter>
            <Button type="submit" disabled={loading}>
              {loading ? 'Saving...' : isEdit ? 'Update Mod' : 'Create Mod'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
