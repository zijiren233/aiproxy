import { useState, type ReactNode } from "react"
import { Outlet } from "react-router"
import { Sidebar } from "./SideBar"
import { cn } from "@/lib/utils"
import { ThemeToggle } from "@/components/common/ThemeToggle"
import { LanguageSelector } from "@/components/common/LanguageSelector"
import { Bell, CircleHelp } from "lucide-react"
import { useLocation } from "react-router"
import { useTranslation } from "react-i18next"

export function RootLayout() {
  const [collapsed, setCollapsed] = useState(false)
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

      <main className={cn("flex-1 flex flex-col overflow-hidden transition-all duration-300 bg-slate-50/80 dark:bg-slate-950")}>
        <header className="h-16 shrink-0 border-b bg-white/80 dark:bg-slate-950/80 backdrop-blur supports-[backdrop-filter]:bg-white/60 dark:border-slate-800 flex items-center justify-between px-5 sm:px-8">
          <div><p className="text-[11px] font-medium uppercase tracking-wider text-muted-foreground">AI Proxy / Workspace</p><h2 className="text-lg font-semibold tracking-tight">{titles[location.pathname] || "Workspace"}</h2></div>
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
