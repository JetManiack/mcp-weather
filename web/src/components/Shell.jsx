import { Outlet, NavLink } from "react-router-dom";

export default function Shell({ pages, user }) {
  return (
    <div className="app-layout">
      <aside className="sidebar">
        <div className="sidebar-brand">mcp-weather</div>
        <nav className="sidebar-nav">
          {user?.role === "admin" && (
            <NavLink
              to="/agents"
              className={({ isActive }) => "sidebar-link" + (isActive ? " active" : "")}
            >
              Agents
            </NavLink>
          )}
          <NavLink
            to="/history"
            className={({ isActive }) => "sidebar-link" + (isActive ? " active" : "")}
          >
            History
          </NavLink>
          {pages.map((p) => (
            <NavLink
              key={p.path}
              to={p.path}
              className={({ isActive }) => "sidebar-link" + (isActive ? " active" : "")}
            >
              {p.label}
            </NavLink>
          ))}
        </nav>
        {user && (
          <div className="sidebar-user">
            <span className="sidebar-user-name">{user.display_name}</span>
            <form method="post" action="/auth/logout">
              <button type="submit" className="sidebar-logout">Sign out</button>
            </form>
          </div>
        )}
      </aside>
      <div className="content-area">
        <Outlet />
      </div>
    </div>
  );
}
