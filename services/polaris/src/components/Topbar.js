import React from "react";

const authLabel = (auth) => {
  if (!auth?.ready) {
    return "Checking identity";
  }
  if (auth.principal) {
    return auth.principal;
  }
  if (auth.error) {
    return auth.error;
  }
  return auth.modeLabel || "Local development";
};

const Topbar = ({ auth, onSignOut }) => (
  <header className="topbar">
    <div>
      <p className="eyebrow">Asterism Control Plane</p>
      <h1>Polaris Mission Console</h1>
      <p className="subtitle">Read-first operations for home infrastructure, without shadow configuration.</p>
    </div>
    <div className="identity-panel">
      <span className="status-pill">{authLabel(auth)}</span>
      {auth?.enabled && (
        <button type="button" onClick={onSignOut}>Sign out</button>
      )}
    </div>
  </header>
);

export default Topbar;
