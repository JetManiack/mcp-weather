import { useState, useEffect } from "react";
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom";
import Shell from "./components/Shell.jsx";
import AgentsPage from "./pages/Agents.jsx";
import HistoryPage from "./pages/History.jsx";
import { domainPages } from "./pages/domain/index.js";
import { getCurrentUser } from "./api.js";

function useCurrentUser() {
  const [user, setUser] = useState(undefined);
  useEffect(() => {
    getCurrentUser().then(setUser).catch(() => setUser(null));
  }, []);
  return user;
}

export default function App() {
  const user = useCurrentUser();

  // Still loading
  if (user === undefined) return null;

  // Not authenticated — redirect to login
  if (user === null) {
    window.location.href = "/auth/login";
    return null;
  }

  return (
    <BrowserRouter>
      <Routes>
        <Route element={<Shell pages={domainPages} user={user} />}>
          {user.role === "admin" && (
            <Route path="/agents" element={<AgentsPage />} />
          )}
          <Route path="/history" element={<HistoryPage />} />
          {domainPages.map((p) => (
            <Route key={p.path} path={p.path} element={<p.component />} />
          ))}
          <Route path="*" element={<Navigate to="/history" replace />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
