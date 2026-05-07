const LOCAL_HOSTS = new Set(["localhost", "127.0.0.1", "::1"]);

const isLocalDevelopment = () => {
  if (typeof window === "undefined") {
    return true;
  }

  return LOCAL_HOSTS.has(window.location.hostname);
};

export const buildShellAuthState = (principal = "") => ({
  ready: true,
  enabled: !isLocalDevelopment(),
  authenticated: !isLocalDevelopment(),
  principal,
  error: "",
});

export const signOut = () => {
  if (typeof window === "undefined") {
    return;
  }

  window.location.assign("/oauth/sign_out");
};
