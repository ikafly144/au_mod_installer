import React, { useState, useEffect } from 'react'
import { getMods, deleteMod } from '@/api'
import { logout, getUser } from '@/auth'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Plus, LogOut, Edit, Trash2, ChevronDown, ChevronUp } from 'lucide-react'
import { ModDialog } from './mod-dialog'
import { VersionList } from './version-list'
import { useToast } from '@/hooks/use-toast'

export default function Dashboard() {

  const [mods, setMods] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [expandedModId, setExpandedModId] = useState<string | null>(null)
  const user = getUser()
  const { toast } = useToast()

    const fetchMods = async () => {
    try {
      const data = await getMods()
      setMods(data || [])
    } catch (e: any) {

      toast({
        variant: 'destructive',
        title: 'Error loading mods',
        description: e.message,
      })
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchMods()
  }, [])

  const handleDeleteMod = async (id: string) => {
    if (!confirm(`Are you sure you want to delete mod ${id}?`)) return
    try {
      await deleteMod(id)
      toast({ title: 'Mod deleted' })
      fetchMods()
    } catch (e: any) {
      toast({
        variant: 'destructive',
        title: 'Delete failed',
        description: e.message,
      })
    }
  }

    return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h2 className="text-2xl font-bold tracking-tight">Mods Repository</h2>
        <ModDialog onSuccess={fetchMods}>
          <Button size="sm">
            <Plus className="h-4 w-4 mr-2" /> Create Mod
          </Button>
        </ModDialog>
      </div>

      <Card>
        <CardContent className="p-0">
          {loading ? (
            <div className="p-8 text-center text-slate-500">Loading mods...</div>
          ) : mods.length === 0 ? (
            <div className="p-8 text-center text-slate-500">No mods found.</div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-[50px]"></TableHead>
                  <TableHead>ID</TableHead>
                  <TableHead>Name</TableHead>
                  <TableHead>Author</TableHead>
                  <TableHead className="hidden md:table-cell">Type</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {mods.map((mod) => (
                  <React.Fragment key={mod.id}>
                    <TableRow className="cursor-pointer" onClick={() => setExpandedModId(expandedModId === mod.id ? null : mod.id)}>
                      <TableCell>
                        {expandedModId === mod.id ? <ChevronUp className="h-4 w-4" /> : <ChevronDown className="h-4 w-4" />}
                      </TableCell>
                      <TableCell className="font-mono text-xs">{mod.id}</TableCell>
                      <TableCell className="font-medium">{mod.name}</TableCell>
                      <TableCell>{mod.author}</TableCell>
                      <TableCell className="hidden md:table-cell">{mod.type}</TableCell>
                      <TableCell className="text-right">
                        <div className="flex justify-end gap-2" onClick={(e) => e.stopPropagation()}>
                          <ModDialog mod={mod} onSuccess={fetchMods}>
                            <Button variant="ghost" size="icon">
                              <Edit className="h-4 w-4" />
                            </Button>
                          </ModDialog>
                          <Button variant="ghost" size="icon" className="text-red-500 hover:text-red-600" onClick={() => handleDeleteMod(mod.id)}>
                            <Trash2 className="h-4 w-4" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                    {expandedModId === mod.id && (
                      <TableRow>
                        <TableCell colSpan={6} className="bg-slate-50/50 dark:bg-slate-900/50 p-0">
                          <div className="p-4 pl-12 border-b">
                            <VersionList modID={mod.id} />
                          </div>
                        </TableCell>
                      </TableRow>
                    )}
                  </React.Fragment>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  )
}


