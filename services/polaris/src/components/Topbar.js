import React from "react";

const authLabel = (auth) => {
  if (!auth?.ready) {
    return "Checking identity";
  }
  if (!auth.enabled) {
    return "Sign-in not configured";
  }
  if (auth.authenticated) {
    return auth.principal || "Signed in";
  }
  return "Sign-in required";
};

const Topbar = ({ auth, onSignIn, onSignOut }) => (
  <header className="topbar">
    <div>
      <p className="eyebrow">Asterism Control Plane</p>
      <h1>Polaris Mission Console</h1>
      <p className="subtitle">Read-first operations for home infrastructure, without shadow configuration.</p>
    </div>
    <div className="identity-panel">
      <span className="status-pill">{authLabel(auth)}</span>
      {auth?.enabled && auth.authenticated && (
        <button type="button" onClick={onSignOut}>Sign out</button>
      )}
      {auth?.enabled && !auth.authenticated && (
        <button type="button" onClick={onSignIn}>Sign in</button>
      )}
    </div>
  </header>
);

export default Topbar;
