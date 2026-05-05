const CONFIG_URL = "/oidc-config.json";
const TOKEN_STORAGE_KEY = "asterism.oidc.token";
const VERIFIER_STORAGE_KEY = "asterism.oidc.verifier";
const STATE_STORAGE_KEY = "asterism.oidc.state";

export const disabledAuthState = {
  ready: true,
  enabled: false,
  authenticated: false,
  authorizationHeader: "",
  principal: "",
  error: "",
};

const base64Url = (bytes) => btoa(String.fromCharCode(...bytes))
  .replace(/\+/g, "-")
  .replace(/\//g, "_")
  .replace(/=+$/, "");

const randomString = (length = 48) => {
  const bytes = new Uint8Array(length);
  window.crypto.getRandomValues(bytes);
  return base64Url(bytes);
};

const sha256 = async (value) => {
  const bytes = new TextEncoder().encode(value);
  const digest = await window.crypto.subtle.digest("SHA-256", bytes);
  return base64Url(new Uint8Array(digest));
};

const redirectUri = () => `${window.location.origin}${window.location.pathname}`;

const normalizeConfig = (config) => {
  if (!config || config.enabled !== true) {
    return { enabled: false };
  }

  return {
    enabled: true,
    issuer: config.issuer || "",
    clientId: config.clientId || "",
    authorizationEndpoint: config.authorizationEndpoint || `${config.issuer}/oauth2/authorize`,
    tokenEndpoint: config.tokenEndpoint || `${config.issuer}/oauth2/token`,
    logoutEndpoint: config.logoutEndpoint || `${config.issuer}/logout`,
    scopes: Array.isArray(config.scopes) && config.scopes.length > 0
      ? config.scopes
      : ["openid", "profile", "email"],
  };
};

export const loadOidcConfig = async () => {
  if (typeof fetch !== "function") {
    return { enabled: false };
  }

  try {
    const response = await fetch(CONFIG_URL, {
      headers: {
        Accept: "application/json",
      },
    });

    if (!response.ok) {
      return { enabled: false };
    }

    return normalizeConfig(await response.json());
  } catch (error) {
    return { enabled: false };
  }
};

const decodeJwtPayload = (token) => {
  if (!token || !token.includes(".")) {
    return {};
  }

  try {
    const payload = token.split(".")[1].replace(/-/g, "+").replace(/_/g, "/");
    const padded = payload.padEnd(payload.length + ((4 - (payload.length % 4)) % 4), "=");
    return JSON.parse(atob(padded));
  } catch (error) {
    return {};
  }
};

const tokenPrincipal = (tokenSet) => {
  const payload = decodeJwtPayload(tokenSet.idToken || tokenSet.accessToken);
  return payload.email || payload.name || payload["cognito:username"] || payload.sub || "";
};

const readStoredToken = () => {
  try {
    const raw = window.localStorage.getItem(TOKEN_STORAGE_KEY);
    if (!raw) {
      return null;
    }

    const tokenSet = JSON.parse(raw);
    if (!tokenSet.accessToken || !tokenSet.expiresAt || tokenSet.expiresAt <= Date.now() + 30000) {
      window.localStorage.removeItem(TOKEN_STORAGE_KEY);
      return null;
    }

    return tokenSet;
  } catch (error) {
    window.localStorage.removeItem(TOKEN_STORAGE_KEY);
    return null;
  }
};

const storeToken = (payload) => {
  const expiresIn = Number(payload.expires_in || 3600);
  const tokenSet = {
    accessToken: payload.access_token,
    idToken: payload.id_token || "",
    refreshToken: payload.refresh_token || "",
    expiresAt: Date.now() + expiresIn * 1000,
  };
  window.localStorage.setItem(TOKEN_STORAGE_KEY, JSON.stringify(tokenSet));
  return tokenSet;
};

const exchangeCode = async (config, code, verifier) => {
  const body = new URLSearchParams({
    grant_type: "authorization_code",
    client_id: config.clientId,
    code,
    code_verifier: verifier,
    redirect_uri: redirectUri(),
  });

  const response = await fetch(config.tokenEndpoint, {
    method: "POST",
    headers: {
      "Content-Type": "application/x-www-form-urlencoded",
      Accept: "application/json",
    },
    body,
  });

  if (!response.ok) {
    throw new Error(`token exchange failed with HTTP ${response.status}`);
  }

  return storeToken(await response.json());
};

export const completeSignInIfNeeded = async (config) => {
  if (!config.enabled) {
    return null;
  }

  const params = new URLSearchParams(window.location.search);
  const code = params.get("code");
  const state = params.get("state");
  if (!code || !state) {
    return readStoredToken();
  }

  const expectedState = window.sessionStorage.getItem(STATE_STORAGE_KEY);
  const verifier = window.sessionStorage.getItem(VERIFIER_STORAGE_KEY);
  window.sessionStorage.removeItem(STATE_STORAGE_KEY);
  window.sessionStorage.removeItem(VERIFIER_STORAGE_KEY);

  if (state !== expectedState || !verifier) {
    throw new Error("OIDC sign-in state did not match");
  }

  const tokenSet = await exchangeCode(config, code, verifier);
  window.history.replaceState({}, document.title, redirectUri());
  return tokenSet;
};

export const buildAuthState = (config, tokenSet, error = "") => {
  if (!config.enabled) {
    return disabledAuthState;
  }

  if (!tokenSet?.accessToken) {
    return {
      ready: true,
      enabled: true,
      authenticated: false,
      authorizationHeader: "",
      principal: "",
      error,
    };
  }

  return {
    ready: true,
    enabled: true,
    authenticated: true,
    authorizationHeader: `Bearer ${tokenSet.accessToken}`,
    principal: tokenPrincipal(tokenSet),
    error,
  };
};

export const startSignIn = async (config) => {
  if (!config.enabled || !config.clientId || !config.authorizationEndpoint) {
    throw new Error("OIDC is not configured");
  }

  const verifier = randomString(64);
  const state = randomString(32);
  const challenge = await sha256(verifier);
  window.sessionStorage.setItem(VERIFIER_STORAGE_KEY, verifier);
  window.sessionStorage.setItem(STATE_STORAGE_KEY, state);

  const params = new URLSearchParams({
    client_id: config.clientId,
    response_type: "code",
    scope: config.scopes.join(" "),
    redirect_uri: redirectUri(),
    code_challenge: challenge,
    code_challenge_method: "S256",
    state,
  });

  window.location.assign(`${config.authorizationEndpoint}?${params.toString()}`);
};

export const signOut = (config) => {
  window.localStorage.removeItem(TOKEN_STORAGE_KEY);
  if (config.enabled && config.logoutEndpoint && config.clientId) {
    const params = new URLSearchParams({
      client_id: config.clientId,
      logout_uri: redirectUri(),
    });
    window.location.assign(`${config.logoutEndpoint}?${params.toString()}`);
  }
};
