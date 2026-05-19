import { Outlet } from "react-router-dom"
import Nav from "@/components/Nav"
import Footer from "@/components/Footer"

export default function RootLayout() {
  return (
    <div className="flex min-h-screen flex-col bg-background">
      <Nav />
      <main className="flex-1">
        <Outlet />
      </main>
      <Footer />
    </div>
  )
}
