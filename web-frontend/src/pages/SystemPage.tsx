import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Terminal } from "lucide-react"

export default function SystemPage() {
  return (
    <div className="space-y-6">
      <div className="flex items-center gap-2">
        <Terminal className="h-8 w-8 text-primary" />
        <h1 className="text-3xl font-bold tracking-tight">System Configuration</h1>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>API Status</CardTitle>
          <CardDescription>Monitor the backend API and server health.</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center gap-2">
            <div className="h-2 w-2 rounded-full bg-green-500" />
            <span className="text-sm font-medium">Connected to Backend</span>
          </div>
          <p className="mt-4 text-sm text-muted-foreground">
            Additional system logs and configuration options will be available here.
          </p>
        </CardContent>
      </Card>
    </div>
  )
}
