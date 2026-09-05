import { useState, type ReactNode } from "react"
import { Outlet } from "react-router"
import { Sidebar } from "./SideBar"
import { cn } from "@/lib/utils"
import { ThemeToggle } from "@/components/common/ThemeToggle"
import { LanguageSelector } from "@/components/common/LanguageSelector"
import { Bell, CircleHelp, Menu } from "lucide-react"
import { useLocation } from "react-router"
import { useTranslation } from "react-i18next"
import { Button } from "@/components/ui/button"
import { Sheet, SheetContent, SheetTitle, SheetTrigger } from "@/components/ui/sheet"

export function RootLayout() {
  const [collapsed, setCollapsed] = useState(false)
  const [mobileNavOpen, setMobileNavOpen] = useState(false)
  const location = useLocation()
  const { t } = useTranslation()
  const titles: Record<string, string> = {
    "/monitor": t("sidebar.monitor"), "/channel": t("sidebar.channel"), "/model": t("sidebar.model"),
    "/log": t("sidebar.log"), "/group": t("sidebar.group"), "/consumption-ranking": t("sidebar.consumptionRanking"),
    "/key": t("sidebar.key"), "/mcp-front": t("sidebar.mcp"),
  }

  return (
    <div className="flex h-screen bg-background">
      <Sidebar
        className="hidden lg:flex"
        displayConfig={{
          monitor: true,
          key: true,
          channel: true,
          model: true,
          log: true,
          doc: true,
          github: true,
        }}
        collapsed={collapsed}
        onToggle={() => setCollapsed(!collapsed)}
      />

      <main className={cn("min-w-0 flex-1 flex flex-col overflow-hidden transition-all duration-300 bg-slate-50/80 dark:bg-slate-950")}>
        <header className="h-16 shrink-0 border-b bg-white/80 dark:bg-slate-950/80 backdrop-blur supports-[backdrop-filter]:bg-white/60 dark:border-slate-800 flex items-center justify-between gap-3 px-4 sm:px-8">
          <div className="flex min-w-0 items-center gap-3"><Sheet open={mobileNavOpen} onOpenChange={setMobileNavOpen}><SheetTrigger asChild><Button variant="ghost" size="icon" className="lg:hidden shrink-0" aria-label="Open navigation"><Menu className="h-5 w-5" /></Button></SheetTrigger><SheetContent side="left" className="w-72 border-0 bg-transparent p-0"><SheetTitle className="sr-only">Navigation</SheetTitle><Sidebar collapsed={false} className="w-full border-0" onNavigate={() => setMobileNavOpen(false)} /></SheetContent></Sheet><div className="min-w-0"><p className="truncate text-[11px] font-medium uppercase tracking-wider text-muted-foreground">AI Proxy / Workspace</p><h2 className="truncate text-lg font-semibold tracking-tight">{titles[location.pathname] || "Workspace"}</h2></div></div>
          <div className="flex items-center gap-1"><ButtonIcon label="Help"><CircleHelp /></ButtonIcon><ButtonIcon label="Notifications"><Bell /></ButtonIcon><div className="mx-2 h-5 w-px bg-border" /><ThemeToggle /><LanguageSelector /></div>
        </header>
        <div className="flex-1 overflow-auto">
          <Outlet />
        </div>
      </main>
    </div>
  )
}

function ButtonIcon({ label, children }: { label: string; children: ReactNode }) {
  return <button aria-label={label} className="hidden sm:flex h-9 w-9 items-center justify-center rounded-lg text-muted-foreground transition-colors hover:bg-muted hover:text-foreground">{children}</button>
}
