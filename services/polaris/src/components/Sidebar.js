import React from "react";

const sidebarIdentityLabel = (auth) => {
  if (!auth?.ready) {
    return "Checking identity";
  }

  if (auth.error) {
    return auth.error;
  }

  return auth.modeLabel || "Local development";
};

const Sidebar = ({ auth }) => (
  <aside className="sidebar">
    <p className="sidebar-heading">Navigation</p>
    <ul>
      <li><a href="#dashboard">Dashboard</a></li>
      <li><a href="#integrations">Integrations</a></li>
    </ul>
    <div className="sidebar-footnote">
      External identity: {sidebarIdentityLabel(auth)}
      <br />
      In-cluster trust: Service Mesh mTLS
    </div>
  </aside>
);

export default Sidebar;
