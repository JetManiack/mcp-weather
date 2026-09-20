// apiCall wraps fetch with admin-token auth from localStorage.
// On 401 it prompts the user for the token and reloads; on other errors
// it throws with the server's error message when available.
export async function apiCall(path, opts = {}) {
  const token = localStorage.getItem("admin_token") || "";
  const headers = { "Content-Type": "application/json", ...opts.headers };
  if (token) headers["Authorization"] = "Bearer " + token;

  const r = await fetch("/api" + path, { ...opts, headers });

  if (r.status === 401) {
    const newToken = prompt("Admin token required:");
    if (newToken) {
      localStorage.setItem("admin_token", newToken);
      window.location.reload();
    }
    throw new Error("unauthorized");
  }

  if (!r.ok) {
    const body = await r.json().catch(() => ({}));
    throw new Error(body.error || r.statusText);
  }

  if (r.status === 204) return null;
  return r.json();
}
