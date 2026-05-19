import { NavLink, Outlet } from "react-router-dom"
import { cn } from "@/lib/utils"

const sidebarLinks = [
  { to: "/docs/quickstart", label: "Quickstart" },
  { to: "/docs/commands", label: "Commands" },
  { to: "/docs/ai-providers", label: "AI Providers" },
  { to: "/docs/configuration", label: "Configuration" },
  { to: "/docs/ci-cd", label: "CI/CD Integration" },
  { to: "/docs/licensing", label: "Licensing" },
  { to: "/docs/team-server", label: "Team Server" },
  { to: "/docs/faq", label: "FAQ" },
]

export default function DocsLayout() {
  return (
    <div className="mx-auto flex w-full max-w-7xl gap-10 px-6 py-10">
      <aside className="hidden w-56 shrink-0 lg:block">
        <nav className="sticky top-24 space-y-1">
          <p className="mb-3 text-xs font-semibold tracking-wider uppercase text-muted-foreground">
            Documentation
          </p>
          {sidebarLinks.map((l) => (
            <NavLink
              key={l.to}
              to={l.to}
              className={({ isActive }) =>
                cn(
                  "block rounded-lg px-3 py-2 text-sm transition-colors",
                  isActive
                    ? "bg-accent text-accent-foreground font-medium"
                    : "text-muted-foreground hover:text-foreground hover:bg-accent/50"
                )
              }
            >
              {l.label}
            </NavLink>
          ))}
        </nav>
      </aside>
      <div className="min-w-0 flex-1">
        <Outlet />
      </div>
    </div>
  )
}
