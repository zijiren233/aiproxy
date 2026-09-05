import type React from "react"
import { Link, useLocation, useNavigate } from "react-router"
import { Bot, Layers, BarChart2, Database, Calendar, ChevronLeft, ChevronRight, FileText, Github, LogOut, MessageCircle, Trophy, Users, Sparkles } from "lucide-react"
import { useTranslation } from "react-i18next"
import type { TFunction } from "i18next"
import { ROUTES } from "@/routes/constants"
import { cn } from "@/lib/utils"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"
import { Button } from "@/components/ui/button"
import useAuthStore from "@/store/auth"

interface SidebarItem { title: string; icon: React.ComponentType<{ className?: string }>; href: string; external?: boolean }
function createSidebarConfig(t: TFunction): SidebarItem[] { return [
    { title: t("sidebar.monitor"), icon: BarChart2, href: ROUTES.MONITOR }, { title: t("sidebar.channel"), icon: Database, href: ROUTES.CHANNEL }, { title: t("sidebar.model"), icon: Layers, href: ROUTES.MODEL }, { title: t("sidebar.log"), icon: Calendar, href: ROUTES.LOG }, { title: t("sidebar.group"), icon: Users, href: ROUTES.GROUP }, { title: t("sidebar.consumptionRanking"), icon: Trophy, href: ROUTES.CONSUMPTION_RANKING }, { title: t("sidebar.key"), icon: Bot, href: ROUTES.KEY }, { title: t("sidebar.mcp"), icon: MessageCircle, href: ROUTES.MCP }, { title: t("sidebar.doc"), icon: FileText, href: "https://sealos.run/docs/guides/ai-proxy", external: true }, { title: t("sidebar.github"), icon: Github, href: "https://github.com/labring/aiproxy", external: true },
] }
interface SidebarProps { displayConfig?: Record<string, boolean>; collapsed?: boolean; onToggle?: () => void; className?: string; onNavigate?: () => void }
export function Sidebar({ displayConfig = {}, collapsed = false, onToggle, className, onNavigate }: SidebarProps) {
    const location = useLocation(); const navigate = useNavigate(); const { t } = useTranslation(); const logout = useAuthStore((s) => s.logout)
    const currentPath = "/" + location.pathname.split("/")[1]
    const items = createSidebarConfig(t).filter((item) => { const key = Object.entries(ROUTES).find(([, value]) => value === item.href)?.[0]?.toLowerCase() || ""; const configuredKey = Object.keys(displayConfig).find((name) => name.toLowerCase() === key); return configuredKey ? displayConfig[configuredKey] !== false : true })
    return <aside className={cn("h-full relative flex flex-col transition-all duration-300 bg-[#111827] dark:bg-[#0b1120] border-r border-white/[0.08]", collapsed ? "w-20" : "w-64", className)}>
        <div className="flex items-center justify-between px-5 py-5 border-b border-white/[0.08]"><div className={cn("overflow-hidden transition-all", collapsed ? "w-0 opacity-0" : "w-auto opacity-100")}><div className="flex items-center gap-3 whitespace-nowrap"><div className="flex h-9 w-9 items-center justify-center rounded-xl bg-indigo-500 shadow-lg shadow-indigo-500/30"><Sparkles className="h-5 w-5 text-white" /></div><div><h1 className="text-[15px] font-semibold tracking-tight text-white">AI Proxy</h1><p className="text-[10px] text-slate-400">CONTROL CENTER</p></div></div></div><Button variant="ghost" size="icon" onClick={onToggle} className={cn("rounded-lg hover:bg-white/10 text-slate-400 hover:text-white", collapsed ? "mx-auto" : "ml-auto")}>{collapsed ? <ChevronRight className="h-4 w-4" /> : <ChevronLeft className="h-4 w-4" />}</Button></div>
        <nav className="flex-1 py-5 overflow-y-auto"><TooltipProvider delayDuration={300}>{items.map((item) => { const active = !item.external && currentPath === item.href; const content = <><item.icon className={cn("h-[18px] w-[18px] shrink-0", active ? "text-indigo-300" : "text-slate-500 group-hover:text-indigo-300")} /><span className={cn("ml-3 truncate text-[13px] font-medium", active ? "text-white" : "text-slate-400 group-hover:text-white", collapsed && "w-0 opacity-0")}>{item.title}</span></>; const cls = cn("group flex items-center px-4 py-2.5 my-1 mx-3 rounded-lg transition-colors", active ? "bg-indigo-500/20 text-white ring-1 ring-indigo-400/20" : "text-slate-400 hover:bg-white/[0.06] hover:text-white", collapsed && "justify-center px-0 mx-2"); return <Tooltip key={item.href}><TooltipTrigger asChild>{item.external ? <a href={item.href} target="_blank" rel="noopener noreferrer" className={cls}>{content}</a> : <Link to={item.href} onClick={onNavigate} className={cls}>{content}</Link>}</TooltipTrigger>{collapsed && <TooltipContent side="right">{item.title}</TooltipContent>}</Tooltip> })}</TooltipProvider></nav>
        <div className="p-3 border-t border-white/[0.08]"><Button variant="secondary" onClick={() => { onNavigate?.(); logout(); navigate("/login") }} className={cn("group w-full rounded-lg bg-white/[0.06] text-slate-300 hover:bg-white/[0.1] hover:text-white", collapsed ? "justify-center px-0" : "justify-start")}><LogOut className="h-[18px] w-[18px]" /><span className={cn("ml-3 text-[13px]", collapsed && "w-0 opacity-0")}>{t("sidebar.logout")}</span></Button></div>
    </aside>
}
