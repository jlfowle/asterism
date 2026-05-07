import React, { useState } from "react";
import Topbar from "./components/Topbar";
import Sidebar from "./components/Sidebar";
import MainContent from "./components/MainContent";
import { buildShellAuthState, signOut } from "./auth";
import "./App.css";

const App = () => {
  const [principal, setPrincipal] = useState("");
  const auth = buildShellAuthState(principal);

  const handleSignOut = () => {
    signOut();
  };

  return (
    <div className="dashboard-layout">
      <Topbar auth={auth} onSignOut={handleSignOut} />
      <div className="dashboard-body">
        <Sidebar />
        <MainContent onPrincipalChange={setPrincipal} />
      </div>
    </div>
  );
};

export default App;
