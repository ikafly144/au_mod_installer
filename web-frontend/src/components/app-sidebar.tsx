import { LayoutDashboard, Package, Settings, Terminal, LogOut } from "lucide-react"
import { Link, useLocation } from "react-router-dom"
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar"
import { logout, getUser } from "@/auth"

const items = [
  {
    title: "Dashboard",
    url: "/",
    icon: LayoutDashboard,
  },
  {
    title: "Mod Management",
    url: "/mods",
    icon: Package,
  },
  {
    title: "User Settings",
    url: "/settings",
    icon: Settings,
  },
  {
    title: "System Configuration",
    url: "/system",
    icon: Terminal,
  },
]

export function AppSidebar() {
  const location = useLocation()
  const user = getUser()

  return (
    <Sidebar>
      <SidebarHeader className="border-b p-4">
        <div className="flex items-center gap-2 font-bold">
          <Package className="h-6 w-6 text-primary" />
          <span>Au Mod Installer</span>
        </div>
      </SidebarHeader>
      <SidebarContent>
        <SidebarGroup>
          <SidebarGroupLabel>Menu</SidebarGroupLabel>
          <SidebarGroupContent>
            <SidebarMenu>
              {items.map((item) => (
                <SidebarMenuItem key={item.title}>
                  <SidebarMenuButton asChild isActive={location.pathname === item.url} tooltip={item.title}>
                    <Link to={item.url}>
                      <item.icon />
                      <span>{item.title}</span>
                    </Link>
                  </SidebarMenuButton>
                </SidebarMenuItem>
              ))}
            </SidebarMenu>
          </SidebarGroupContent>
        </SidebarGroup>
      </SidebarContent>
      <SidebarFooter className="border-t p-4">
        <div className="flex items-center justify-between gap-4">
            <div className="flex flex-col overflow-hidden">
                <span className="truncate text-xs font-medium">{user?.username}</span>
                <span className="truncate text-[10px] text-muted-foreground">Administrator</span>
            </div>
            <SidebarMenuButton onClick={logout} className="h-8 w-8" tooltip="Logout">
                <LogOut className="h-4 w-4" />
            </SidebarMenuButton>
        </div>
      </SidebarFooter>
    </Sidebar>
  )
}
