import * as React from "react"
import { cn } from "@/lib/utils"

// ------------------------------------------------------------------ //
//  Tabs Context                                                       //
// ------------------------------------------------------------------ //
interface TabsContextValue {
  value: string
  onValueChange: (v: string) => void
}
const TabsContext = React.createContext<TabsContextValue | null>(null)

// ------------------------------------------------------------------ //
//  TabsProvider — supports controlled (value+onValueChange) and       //
//  uncontrolled (defaultValue) modes                                  //
// ------------------------------------------------------------------ //
export function TabsProvider({
  children,
  value: controlledValue,
  defaultValue,
  onValueChange,
}: {
  children: React.ReactNode
  value?: string
  defaultValue?: string
  onValueChange?: (v: string) => void
}) {
  const isControlled = controlledValue !== undefined
  const [internal, setInternal] = React.useState(defaultValue || "")

  const ctx: TabsContextValue = {
    value: isControlled ? controlledValue : internal,
    onValueChange: (v: string) => {
      if (!isControlled) setInternal(v)
      onValueChange?.(v)
    },
  }

  return <TabsContext.Provider value={ctx}>{children}</TabsContext.Provider>
}

// ------------------------------------------------------------------ //
//  Components                                                         //
// ------------------------------------------------------------------ //
export function Tabs({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return <div className={cn("", className)} {...props} />
}

export function TabsList({ className, ...props }: React.HTMLAttributes<HTMLDivElement>) {
  return (
    <div
      className={cn(
        "inline-flex h-10 items-center justify-center rounded-lg bg-muted p-1 text-muted-foreground",
        className
      )}
      {...props}
    />
  )
}

export function TabsTrigger({
  className,
  value,
  ...props
}: React.ButtonHTMLAttributes<HTMLButtonElement> & { value: string }) {
  const ctx = React.useContext(TabsContext)!
  const isActive = ctx.value === value
  return (
    <button
      role="tab"
      data-state={isActive ? "active" : "inactive"}
      onClick={() => ctx.onValueChange(value)}
      className={cn(
        "inline-flex items-center justify-center whitespace-nowrap rounded-md px-3 py-1.5 text-sm font-medium ring-offset-background transition-all cursor-pointer",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
        "disabled:pointer-events-none disabled:opacity-50",
        isActive && "bg-background text-foreground shadow-sm",
        !isActive && "hover:bg-background/50 hover:text-foreground",
        className
      )}
      {...props}
    />
  )
}

export function TabsContent({
  className,
  value,
  ...props
}: React.HTMLAttributes<HTMLDivElement> & { value: string }) {
  const ctx = React.useContext(TabsContext)
  if (ctx?.value !== value) return null
  return (
    <div
      role="tabpanel"
      data-state="active"
      className={cn(
        "mt-2 ring-offset-background focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2",
        className
      )}
      {...props}
    />
  )
}
