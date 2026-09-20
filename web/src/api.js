// apiCall wraps fetch with cookie-based session auth.
// On 401 it redirects the browser to /auth/login; on other errors it throws
// with the server's error message when available.
export async function apiCall(path, opts = {}) {
  const headers = { "Content-Type": "application/json", ...opts.headers };
  const r = await fetch("/api" + path, { ...opts, headers, credentials: "same-origin" });

  if (r.status === 401) {
    window.location.href = "/auth/login";
    throw new Error("unauthorized");
  }

  if (!r.ok) {
    const body = await r.json().catch(() => ({}));
    throw new Error(body.error || r.statusText);
  }

  if (r.status === 204) return null;
  return r.json();
}

export async function getCurrentUser() {
  const r = await fetch("/api/me", { credentials: "same-origin" });
  if (!r.ok) return null;
  return r.json();
}
