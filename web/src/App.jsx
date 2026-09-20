import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import Shell from "./components/Shell.jsx";
import AgentsPage from "./pages/Agents.jsx";
import HistoryPage from "./pages/History.jsx";
import { domainPages } from "./pages/domain/index.js";

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<Shell pages={domainPages} />}>
          <Route path="/agents" element={<AgentsPage />} />
          <Route path="/history" element={<HistoryPage />} />
          {domainPages.map((p) => (
            <Route key={p.path} path={p.path} element={<p.component />} />
          ))}
          <Route path="*" element={<Navigate to="/agents" replace />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
