import { Link, useLocation } from "react-router-dom"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"

const links = [
  { to: "/", label: "Home" },
  { to: "/pricing", label: "Pricing" },
  { to: "/docs/quickstart", label: "Docs" },
]

export default function Nav() {
  const { pathname } = useLocation()

  return (
    <header className="sticky top-0 z-50 w-full border-b border-border/40 bg-background/95 backdrop-blur-sm">
      <div className="mx-auto flex h-16 max-w-7xl items-center justify-between px-6">
        <Link to="/" className="flex items-center">
          <span className="text-lg font-semibold tracking-tight">
            <span className="text-[#FC0E22]">b</span>asemake
          </span>
        </Link>

        <nav className="flex items-center gap-1">
          {links.map((l) => (
            <Link
              key={l.to}
              to={l.to}
              className={cn(
                "px-4 py-2 text-sm rounded-lg transition-colors",
                pathname === l.to
                  ? "text-foreground font-medium"
                  : "text-muted-foreground hover:text-foreground"
              )}
            >
              {l.label}
            </Link>
          ))}
          <Link to="/docs/quickstart">
            <Button size="sm" className="ml-2 bg-[#FC0E22] hover:bg-[#d90c18] text-white border-none">Get Started</Button>
          </Link>
        </nav>
      </div>
    </header>
  )
}
