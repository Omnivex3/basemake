import type { SVGProps } from "react"

function IconBase({ children, ...props }: SVGProps<SVGSVGElement> & { children: React.ReactNode }) {
  return (
    <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" {...props}>
      {children}
    </svg>
  )
}

export function DatabaseIcon(props: SVGProps<SVGSVGElement>) {
  return (
    <IconBase {...props}>
      <ellipse cx="12" cy="6" rx="9" ry="3" stroke="currentColor" strokeWidth="1.5" />
      <path d="M3 6v12c0 1.66 4 3 9 3s9-1.34 9-3V6" stroke="currentColor" strokeWidth="1.5" />
      <path d="M3 12c0 1.66 4 3 9 3s9-1.34 9-3" stroke="currentColor" strokeWidth="1.5" />
    </IconBase>
  )
}

export function BrainIcon(props: SVGProps<SVGSVGElement>) {
  return (
    <IconBase {...props}>
      <path d="M12 3c-1.5 0-3 1-3 3 0-2-1.5-3-3-3S4 5 4 7c0 1.5.5 2.5 1 3" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
      <path d="M12 3c1.5 0 3 1 3 3 0-2 1.5-3 3-3s2 2 2 4c0 1.5-.5 2.5-1 3" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
      <path d="M8 13c0-2 1-3 2-3.5" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
      <path d="M16 13c0-2-1-3-2-3.5" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
      <path d="M9 17c0 1.5 1 2 3 2s3-.5 3-2" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
      <path d="M6 21c0-1.5 2-2 6-2s6 .5 6 2" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
    </IconBase>
  )
}

export function TerminalIcon(props: SVGProps<SVGSVGElement>) {
  return (
    <IconBase {...props}>
      <rect x="2" y="4" width="20" height="16" rx="2" stroke="currentColor" strokeWidth="1.5" />
      <path d="M6 10l2 2-2 2" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
      <path d="M11 14h5" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
    </IconBase>
  )
}

export function PromptIcon(props: SVGProps<SVGSVGElement>) {
  return (
    <IconBase {...props}>
      <rect x="3" y="3" width="18" height="18" rx="3" stroke="currentColor" strokeWidth="1.5" />
      <path d="M7 9l2 2-2 2" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
      <path d="M13 13h4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
      <path d="M21 9h-2" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
    </IconBase>
  )
}

export function BoltIcon(props: SVGProps<SVGSVGElement>) {
  return (
    <IconBase {...props}>
      <path d="M13 3L4 14h7l-1 7 9-11h-7l1-7z" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round" />
    </IconBase>
  )
}

export function CompareIcon(props: SVGProps<SVGSVGElement>) {
  return (
    <IconBase {...props}>
      <rect x="3" y="4" width="7" height="16" rx="1.5" stroke="currentColor" strokeWidth="1.5" />
      <rect x="14" y="4" width="7" height="16" rx="1.5" stroke="currentColor" strokeWidth="1.5" />
      <path d="M6.5 10v4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
      <path d="M17.5 8v6" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
    </IconBase>
  )
}

export function EyeIcon(props: SVGProps<SVGSVGElement>) {
  return (
    <IconBase {...props}>
      <path d="M2 12s3-7 10-7 10 7 10 7-3 7-10 7-10-7-10-7z" stroke="currentColor" strokeWidth="1.5" />
      <circle cx="12" cy="12" r="3" stroke="currentColor" strokeWidth="1.5" />
    </IconBase>
  )
}

export function ShieldIcon(props: SVGProps<SVGSVGElement>) {
  return (
    <IconBase {...props}>
      <path d="M12 2l7 3v7c0 4-3 8-7 9-4-1-7-5-7-9V5l7-3z" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round" />
      <path d="M9 12l2 2 4-4" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" />
    </IconBase>
  )
}

export function CubeIcon(props: SVGProps<SVGSVGElement>) {
  return (
    <IconBase {...props}>
      <path d="M12 2L4 7v10l8 5 8-5V7l-8-5z" stroke="currentColor" strokeWidth="1.5" strokeLinejoin="round" />
      <path d="M4 7l8 5 8-5" stroke="currentColor" strokeWidth="1.5" />
      <path d="M12 22V12" stroke="currentColor" strokeWidth="1.5" />
    </IconBase>
  )
}
