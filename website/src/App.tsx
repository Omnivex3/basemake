import { Routes, Route } from "react-router-dom"
import RootLayout from "@/layouts/RootLayout"
import DocsLayout from "@/layouts/DocsLayout"
import Landing from "@/pages/Landing"
import Pricing from "@/pages/Pricing"
import Quickstart from "@/pages/docs/Quickstart"
import Commands from "@/pages/docs/Commands"
import AIProviders from "@/pages/docs/AIProviders"
import Configuration from "@/pages/docs/Configuration"
import CICD from "@/pages/docs/CICD"
import Licensing from "@/pages/docs/Licensing"
import TeamServer from "@/pages/docs/TeamServer"
import FAQ from "@/pages/docs/FAQ"

export default function App() {
  return (
    <Routes>
      <Route element={<RootLayout />}>
        <Route index element={<Landing />} />
        <Route path="features" element={<Pricing />} />
        <Route path="pricing" element={<Pricing />} />
        <Route path="docs" element={<DocsLayout />}>
          <Route index element={<Quickstart />} />
          <Route path="quickstart" element={<Quickstart />} />
          <Route path="commands" element={<Commands />} />
          <Route path="ai-providers" element={<AIProviders />} />
          <Route path="configuration" element={<Configuration />} />
          <Route path="ci-cd" element={<CICD />} />
          <Route path="licensing" element={<Licensing />} />
          <Route path="team-server" element={<TeamServer />} />
          <Route path="faq" element={<FAQ />} />
        </Route>
      </Route>
    </Routes>
  )
}
