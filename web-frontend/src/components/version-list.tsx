import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { getModVersions, deleteVersion } from '@/api'
import { Button } from '@/components/ui/button'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Plus, Trash2 } from 'lucide-react'
import { useToast } from '@/hooks/use-toast'

export function VersionList({ modID }: { modID: string }) {
  const [versions, setVersions] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const navigate = useNavigate()
  const { toast } = useToast()


  const fetchVersions = async () => {
        try {
      const data = await getModVersions(modID)
      setVersions(data || [])
    } catch (e: any) {

      toast({
        variant: 'destructive',
        title: 'Error loading versions',
        description: e.message,
      })
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchVersions()
  }, [modID])

  const handleDeleteVersion = async (versionID: string) => {
    if (!confirm(`Delete version ${versionID}?`)) return
    try {
      await deleteVersion(modID, versionID)
      toast({ title: 'Version deleted' })
      fetchVersions()
    } catch (e: any) {
      toast({
        variant: 'destructive',
        title: 'Delete failed',
        description: e.message,
      })
    }
  }

  return (
    <div className="space-y-4">
            <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold uppercase tracking-wider text-slate-500">Versions</h3>
        <Button size="xs" variant="outline" className="h-8" onClick={() => navigate(`/mods/${modID}/versions/new`)}>
          <Plus className="h-3 w-3 mr-1" /> Upload
        </Button>
      </div>


      {loading ? (
        <div className="text-sm text-slate-500 py-2">Loading versions...</div>
      ) : versions.length === 0 ? (
        <div className="text-sm text-slate-500 py-2 text-center border border-dashed rounded-md">No versions found.</div>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead className="h-10">Version</TableHead>
              <TableHead className="h-10">Created At</TableHead>
              <TableHead className="h-10 text-right">Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {versions.map((v) => (
              <TableRow key={v.id}>
                <TableCell className="py-2 font-medium">{v.id}</TableCell>
                <TableCell className="py-2 text-xs text-slate-500">
                  {new Date(v.created_at).toLocaleString()}
                </TableCell>
                <TableCell className="py-2 text-right">
                  <Button variant="ghost" size="icon" className="h-8 w-8 text-red-500" onClick={() => handleDeleteVersion(v.id)}>
                    <Trash2 className="h-3.5 w-3.5" />
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </div>
  )
}
