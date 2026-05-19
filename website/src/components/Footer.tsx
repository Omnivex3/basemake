import { Link } from "react-router-dom"

const footerLinks = {
  Product: [
    { label: "Features", to: "/#features" },
    { label: "Pricing", to: "/pricing" },
    { label: "Docs", to: "/docs/quickstart" },
    { label: "Changelog", to: "https://github.com/DynamicKarabo/basemake/releases" },
  ],
  Docs: [
    { label: "Quickstart", to: "/docs/quickstart" },
    { label: "Commands", to: "/docs/commands" },
    { label: "AI Providers", to: "/docs/ai-providers" },
    { label: "CI/CD", to: "/docs/ci-cd" },
    { label: "FAQ", to: "/docs/faq" },
  ],
  Company: [
    { label: "GitHub", to: "https://github.com/DynamicKarabo/basemake" },
    { label: "License", to: "/docs/licensing" },
  ],
}

export default function Footer() {
  return (
    <footer className="border-t border-border/40 bg-background">
      <div className="mx-auto max-w-7xl px-6 py-16">
        <div className="grid gap-8 sm:grid-cols-2 lg:grid-cols-4">
          <div>
            <Link to="/" className="flex items-center mb-4">
              <span className="text-base font-semibold">
                <span className="text-[#FC0E22]">b</span>asemake
              </span>
            </Link>
            <p className="text-sm text-muted-foreground max-w-xs">
              Talk to your database in plain English. All local. All private. All yours.
            </p>
          </div>
          {Object.entries(footerLinks).map(([title, links]) => (
            <div key={title}>
              <h4 className="text-sm font-semibold mb-3">{title}</h4>
              <ul className="space-y-2">
                {links.map((l) => (
                  <li key={l.label}>
                    <Link
                      to={l.to}
                      className="text-sm text-muted-foreground hover:text-foreground transition-colors"
                    >
                      {l.label}
                    </Link>
                  </li>
                ))}
              </ul>
            </div>
          ))}
        </div>
        <div className="mt-12 border-t border-border/40 pt-6 flex flex-col sm:flex-row items-center justify-between gap-4">
          <p className="text-xs text-muted-foreground">
            &copy; {new Date().getFullYear()} basemake. All rights reserved.
          </p>
          <p className="text-xs text-muted-foreground">
            Built for developers who ship.
          </p>
        </div>
      </div>
    </footer>
  )
}
