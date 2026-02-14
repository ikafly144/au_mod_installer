import React, { useState, useEffect } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { getModVersions, updateVersion } from '@/api'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card'
import { useToast } from '@/hooks/use-toast'
import { ArrowLeft, Loader2, Plus, Trash2 } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Checkbox } from "@/components/ui/checkbox"

export function EditVersionPage() {
  const { id: modID, versionID } = useParams<{ id: string, versionID: string }>()
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const navigate = useNavigate()
  const { toast } = useToast()

  const [version, setVersion] = useState<any>(null)
  
  // Form state
  const [gameVersions, setGameVersions] = useState<string[]>([])
  const [newGameVersion, setNewGameVersion] = useState('')
  const [files, setFiles] = useState<any[]>([])

  useEffect(() => {
    if (modID && versionID) {
      fetchData()
    }
  }, [modID, versionID])

  const fetchData = async () => {
    try {
      // We don't have getModVersion API yet, so fetch all and find
      // Ideally we should add getModVersion to API
      const versions = await getModVersions(modID!)
      const v = versions.find((v: any) => v.id === versionID)
      
      if (!v) {
        throw new Error('Version not found')
      }

      setVersion(v)
      setGameVersions(v.game_versions || [])
      setFiles(v.files || [])
    } catch (e: any) {
      toast({
        variant: 'destructive',
        title: 'Failed to load version',
        description: e.message,
      })
      navigate(`/mods/${modID}/edit`)
    } finally {
      setLoading(false)
    }
  }

  const handleAddGameVersion = () => {
    if (!newGameVersion.trim()) return
    if (gameVersions.includes(newGameVersion.trim())) return
    setGameVersions([...gameVersions, newGameVersion.trim()])
    setNewGameVersion('')
  }

  const handleRemoveGameVersion = (gv: string) => {
    setGameVersions(gameVersions.filter(v => v !== gv))
  }

  const handleAddFile = () => {
    setFiles([...files, { url: '', file_type: 'zip', compatible: [] }])
  }

  const handleRemoveFile = (index: number) => {
    setFiles(files.filter((_, i) => i !== index))
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setSaving(true)
    try {
      const updatedVersion = {
        ...version,
        game_versions: gameVersions,
        files: files
      }

      await updateVersion(modID!, versionID!, updatedVersion)
      toast({ title: 'Version updated successfully' })
      navigate(`/mods/${modID}/edit`)
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
      <div className="flex justify-center py-12">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  return (
    <div className="max-w-2xl mx-auto space-y-6">
      <div className="flex items-center gap-4">
        <Button variant="ghost" size="icon" onClick={() => navigate(`/mods/${modID}/edit`)}>
          <ArrowLeft className="h-5 w-5" />
        </Button>
        <h1 className="text-3xl font-bold tracking-tight">Edit Version {versionID}</h1>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Version Details</CardTitle>
          <CardDescription>Update compatibility and metadata for this version.</CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-6">
            <div className="grid gap-2">
              <Label>Compatible Game Versions</Label>
              <div className="flex gap-2">
                <Input 
                  placeholder="e.g. 2024.12.10" 
                  value={newGameVersion}
                  onChange={(e) => setNewGameVersion(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') {
                      e.preventDefault()
                      handleAddGameVersion()
                    }
                  }}
                />
                <Button type="button" size="icon" onClick={handleAddGameVersion}>
                  <Plus className="h-4 w-4" />
                </Button>
              </div>
              
              <div className="flex flex-wrap gap-2 mt-2">
                {gameVersions.length === 0 && (
                  <span className="text-sm text-muted-foreground italic">No specific versions (compatible with all?)</span>
                )}
                {gameVersions.map(gv => (
                  <Badge key={gv} variant="secondary" className="pl-2 pr-1 py-1 flex items-center gap-1">
                    {gv}
                    <Button 
                      type="button" 
                      variant="ghost" 
                      size="icon" 
                      className="h-4 w-4 rounded-full hover:bg-destructive/20 hover:text-destructive p-0"
                      onClick={() => handleRemoveGameVersion(gv)}
                    >
                      <Trash2 className="h-3 w-3" />
                    </Button>
                  </Badge>
                ))}
              </div>
            </div>

            <div className="grid gap-2">
              <div className="flex items-center justify-between">
                <Label>Files</Label>
                <Button type="button" variant="outline" size="sm" onClick={handleAddFile}>
                  <Plus className="h-3.5 w-3.5 mr-1" /> Add File
                </Button>
              </div>
              <div className="space-y-4">
                {files.map((file, index) => (
                  <Card key={index}>
                    <CardContent className="p-4 space-y-4">
                      <div className="flex gap-4 items-start">
                        <div className="grid gap-2 flex-1">
                          <Label>URL</Label>
                          <Input 
                            value={file.url} 
                            onChange={(e) => {
                              const newFiles = [...files]
                              newFiles[index].url = e.target.value
                              setFiles(newFiles)
                            }}
                          />
                        </div>
                        <Button
                          type="button"
                          variant="ghost" 
                          size="icon"
                          className="mt-6 text-destructive hover:text-destructive hover:bg-destructive/10"
                          onClick={() => handleRemoveFile(index)}
                        >
                          <Trash2 className="h-4 w-4" />
                        </Button>
                      </div>
                      <div className="grid grid-cols-2 gap-4">
                        <div className="grid gap-2">
                          <Label>Type</Label>
                          <Select 
                            value={file.file_type} 
                            onValueChange={(val) => {
                              const newFiles = [...files]
                              newFiles[index].file_type = val
                              setFiles(newFiles)
                            }}
                          >
                            <SelectTrigger>
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value="zip">Zip</SelectItem>
                              <SelectItem value="normal">Normal</SelectItem>
                              <SelectItem value="plugin">Plugin</SelectItem>
                            </SelectContent>
                          </Select>
                        </div>
                        <div className="grid gap-2">
                          <Label>Path (if applicable)</Label>
                          <Input 
                            value={file.path || ''} 
                            onChange={(e) => {
                              const newFiles = [...files]
                              newFiles[index].path = e.target.value
                              setFiles(newFiles)
                            }}
                          />
                        </div>
                      </div>
                      <div className="grid gap-2">
                        <Label>Compatibility</Label>
                        <div className="flex gap-4">
                          <div className="flex items-center space-x-2">
                            <Checkbox 
                              id={`x86-${index}`}
                              checked={file.compatible?.includes('x86')}
                              onCheckedChange={(checked) => {
                                const newFiles = [...files]
                                let compatible = newFiles[index].compatible || []
                                if (checked) {
                                  if (!compatible.includes('x86')) compatible.push('x86')
                                } else {
                                  compatible = compatible.filter((c: string) => c !== 'x86')
                                }
                                newFiles[index].compatible = compatible
                                setFiles(newFiles)
                              }}
                            />
                            <label htmlFor={`x86-${index}`} className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">
                              x86 (32-bit)
                            </label>
                          </div>
                          <div className="flex items-center space-x-2">
                            <Checkbox 
                              id={`x64-${index}`}
                              checked={file.compatible?.includes('x64')}
                              onCheckedChange={(checked) => {
                                const newFiles = [...files]
                                let compatible = newFiles[index].compatible || []
                                if (checked) {
                                  if (!compatible.includes('x64')) compatible.push('x64')
                                } else {
                                  compatible = compatible.filter((c: string) => c !== 'x64')
                                }
                                newFiles[index].compatible = compatible
                                setFiles(newFiles)
                              }}
                            />
                            <label htmlFor={`x64-${index}`} className="text-sm font-medium leading-none peer-disabled:cursor-not-allowed peer-disabled:opacity-70">
                              x64 (64-bit)
                            </label>
                          </div>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                ))}
              </div>
            </div>

            <div className="pt-4 flex justify-end gap-2">
              <Button type="button" variant="outline" onClick={() => navigate(`/mods/${modID}/edit`)}>
                Cancel
              </Button>
              <Button type="submit" disabled={saving}>
                {saving ? 'Saving...' : 'Save Changes'}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  )
}
