const DEFAULT_SIGN_OUT_PATH = "/oauth/sign_out";

export const buildShellAuthState = (runtimeAuthConfig = null, principal = "") => {
  const principalName = typeof principal === "string" ? principal.trim() : "";

  if (!runtimeAuthConfig) {
    return {
      ready: false,
      enabled: false,
      authenticated: false,
      principal: principalName,
      modeLabel: "Checking identity",
      signOutPath: DEFAULT_SIGN_OUT_PATH,
      error: "",
    };
  }

  const edgeAuthEnabled = runtimeAuthConfig.edgeAuthEnabled === true;
  const signOutPath = typeof runtimeAuthConfig.signOutPath === "string" && runtimeAuthConfig.signOutPath.trim() !== ""
    ? runtimeAuthConfig.signOutPath.trim()
    : DEFAULT_SIGN_OUT_PATH;
  const modeLabel = typeof runtimeAuthConfig.modeLabel === "string" && runtimeAuthConfig.modeLabel.trim() !== ""
    ? runtimeAuthConfig.modeLabel.trim()
    : (edgeAuthEnabled ? "OpenShift SSO" : "Local development");

  return {
    ready: true,
    enabled: edgeAuthEnabled,
    authenticated: edgeAuthEnabled || principalName !== "",
    principal: principalName,
    modeLabel,
    signOutPath,
    error: typeof runtimeAuthConfig.error === "string" ? runtimeAuthConfig.error.trim() : "",
  };
};

export const signOut = (signOutPath = DEFAULT_SIGN_OUT_PATH) => {
  if (typeof window === "undefined") {
    return;
  }

  window.location.assign(signOutPath);
};
