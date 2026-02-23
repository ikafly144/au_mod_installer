import { useState } from 'react'
import { useParams } from 'react-router-dom'
import { useForm, useFieldArray } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import * as z from 'zod'

import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { useToast } from '@/hooks/use-toast'
import { PlusCircle, MinusCircle } from 'lucide-react'
import { createVersion } from '@/api'

const formSchema = z.object({
  id: z.string().min(1, 'Version ID is required'),
  files: z
    .array(
      z.object({
        url: z.string().url('Must be a valid URL'),
        file_type: z.enum(['zip', 'normal', 'plugin']),
      }),
    )
    .min(1, 'At least one mod file is required'),
  game_versions: z.string().optional(), // For now, allow free text, later multi-select
  dependencies: z.array(
    z.object({
      id: z.string().min(1, 'Mod ID is required'),
      version: z.string().optional(),
    }),
  ).optional(),
})

type FormValues = z.infer<typeof formSchema>

export function ManualVersionForm() {
  const { id: modID } = useParams<{ id: string }>()
  const { toast } = useToast()
  const [isSubmitting, setIsSubmitting] = useState(false)

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      id: '',
      files: [{ url: '', file_type: 'normal' }],
      game_versions: '',
      dependencies: [],
    },
  })

  const { fields: fileFields, append: appendFile, remove: removeFile } = useFieldArray({
    control: form.control,
    name: 'files',
  })

  const { fields: dependencyFields, append: appendDependency, remove: removeDependency } = useFieldArray({
    control: form.control,
    name: 'dependencies',
  })

  async function onSubmit(values: FormValues) {
    if (!modID) {
      toast({
        variant: 'destructive',
        title: 'Error',
        description: 'Mod ID is missing.',
      })
      return
    }
    setIsSubmitting(true)

    const versionData = {
      id: values.id,
      mod_id: modID, // The backend expects mod_id in the payload
      files: values.files,
      game_versions: values.game_versions ? values.game_versions.split(',').map(v => v.trim()) : [],
      dependencies: values.dependencies,
    }

    try {
      await createVersion(modID, versionData)
      toast({
        title: 'Success',
        description: `Version ${values.id} created successfully.`,
      })
      form.reset() // Reset form after successful submission
    } catch (error: any) {
      toast({
        variant: 'destructive',
        title: 'Error',
        description: `Failed to create version: ${error.message}`,
      })
    } finally {
      setIsSubmitting(false)
    }
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
        <FormField
          control={form.control}
          name="id"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Version ID</FormLabel>
              <FormControl>
                <Input placeholder="e.g., v1.0.0" {...field} />
              </FormControl>
              <FormDescription>
                Unique identifier for this version (e.g., semantic version).
              </FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />

        <div>
          <h3 className="mb-2 text-lg font-medium">Mod Files</h3>
          {fileFields.map((field, index) => (
            <div key={field.id} className="grid grid-cols-1 md:grid-cols-3 gap-2 mb-4 p-4 border rounded-md">
              <FormField
                control={form.control}
                name={`files.${index}.url`}
                render={({ field: urlField }) => (
                  <FormItem>
                    <FormLabel>File URL</FormLabel>
                    <FormControl>
                      <Input placeholder="http://example.com/mod.zip" {...urlField} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name={`files.${index}.file_type`}
                render={({ field: typeField }) => (
                  <FormItem>
                    <FormLabel>File Type</FormLabel>
                    <Select onValueChange={typeField.onChange} defaultValue={typeField.value}>
                      <FormControl>
                        <SelectTrigger>
                          <SelectValue placeholder="Select a file type" />
                        </SelectTrigger>
                      </FormControl>
                      <SelectContent>
                        <SelectItem value="zip">ZIP Archive</SelectItem>
                        <SelectItem value="normal">Normal File</SelectItem>
                        <SelectItem value="plugin">Plugin (DLL/SO)</SelectItem>
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <div className="flex items-end justify-end">
                <Button type="button" variant="destructive" size="icon" onClick={() => removeFile(index)}>
                  <MinusCircle className="h-4 w-4" />
                </Button>
              </div>
            </div>
          ))}
          <Button type="button" variant="outline" onClick={() => appendFile({ url: '', file_type: 'normal' })}>
            <PlusCircle className="mr-2 h-4 w-4" /> Add File
          </Button>
        </div>

        <FormField
          control={form.control}
          name="game_versions"
          render={({ field }) => (
            <FormItem>
              <FormLabel>Compatible Game Versions</FormLabel>
              <FormControl>
                <Input placeholder="e.g., 2023.11.28.0, 2024.1.1.0" {...field} />
              </FormControl>
              <FormDescription>
                Comma-separated list of game versions this mod is compatible with.
              </FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />

        <div>
          <h3 className="mb-2 text-lg font-medium">Dependencies</h3>
          {dependencyFields.map((field, index) => (
            <div key={field.id} className="grid grid-cols-1 md:grid-cols-3 gap-2 mb-4 p-4 border rounded-md">
              <FormField
                control={form.control}
                name={`dependencies.${index}.id`}
                render={({ field: depIdField }) => (
                  <FormItem>
                    <FormLabel>Mod ID</FormLabel>
                    <FormControl>
                      <Input placeholder="e.g., other-mod-id" {...depIdField} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name={`dependencies.${index}.version`}
                render={({ field: depVersionField }) => (
                  <FormItem>
                    <FormLabel>Required Version (optional)</FormLabel>
                    <FormControl>
                      <Input placeholder="e.g., v1.0.0" {...depVersionField} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <div className="flex items-end justify-end">
                <Button type="button" variant="destructive" size="icon" onClick={() => removeDependency(index)}>
                  <MinusCircle className="h-4 w-4" />
                </Button>
              </div>
            </div>
          ))}
          <Button type="button" variant="outline" onClick={() => appendDependency({ id: '', version: '' })}>
            <PlusCircle className="mr-2 h-4 w-4" /> Add Dependency
          </Button>
        </div>

        <Button type="submit" disabled={isSubmitting}>
          {isSubmitting && <PlusCircle className="mr-2 h-4 w-4 animate-spin" />}
          Create Version
        </Button>
      </form>
    </Form>
  )
}
