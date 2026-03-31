import React from "react";

const Sidebar = () => (
  <aside className="sidebar">
    <p className="sidebar-heading">Navigation</p>
    <ul>
      <li><a href="#dashboard">Dashboard</a></li>
      <li><a href="#integrations">Integrations</a></li>
      <li><a href="#security">Security</a></li>
      <li><a href="#events">Events</a></li>
    </ul>
    <div className="sidebar-footnote">
      External identity: Cognito OIDC
      <br />
      In-cluster trust: Service Mesh mTLS
    </div>
  </aside>
);

export default Sidebar;
