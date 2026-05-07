import React, { useEffect, useState } from "react";
import Topbar from "./components/Topbar";
import Sidebar from "./components/Sidebar";
import MainContent from "./components/MainContent";
import { buildShellAuthState, signOut } from "./auth";
import "./App.css";

const DEFAULT_RUNTIME_AUTH_CONFIG = {
  edgeAuthEnabled: false,
  modeLabel: "Local development",
  signOutPath: "/oauth/sign_out",
};

const App = () => {
  const [principal, setPrincipal] = useState("");
  const [runtimeAuthConfig, setRuntimeAuthConfig] = useState(null);

  useEffect(() => {
    let isMounted = true;

    const loadRuntimeAuthConfig = async () => {
      if (typeof fetch !== "function") {
        if (isMounted) {
          setRuntimeAuthConfig(DEFAULT_RUNTIME_AUTH_CONFIG);
        }
        return;
      }

      try {
        const response = await fetch("/runtime-config.json", {
          headers: {
            Accept: "application/json",
          },
        });

        if (!response.ok) {
          throw new Error(`HTTP ${response.status}`);
        }

        const payload = await response.json();
        if (!isMounted) {
          return;
        }

        setRuntimeAuthConfig({
          edgeAuthEnabled: payload?.edgeAuthEnabled === true,
          modeLabel: typeof payload?.modeLabel === "string" && payload.modeLabel.trim() !== ""
            ? payload.modeLabel.trim()
            : (payload?.edgeAuthEnabled === true ? "OpenShift SSO" : "Local development"),
          signOutPath: typeof payload?.signOutPath === "string" && payload.signOutPath.trim() !== ""
            ? payload.signOutPath.trim()
            : DEFAULT_RUNTIME_AUTH_CONFIG.signOutPath,
        });
      } catch (error) {
        if (isMounted) {
          setRuntimeAuthConfig({
            edgeAuthEnabled: false,
            modeLabel: "Identity config unavailable",
            signOutPath: DEFAULT_RUNTIME_AUTH_CONFIG.signOutPath,
            error: "Identity config unavailable",
          });
        }
      }
    };

    loadRuntimeAuthConfig();

    return () => {
      isMounted = false;
    };
  }, []);

  const auth = buildShellAuthState(runtimeAuthConfig, principal);

  const handleSignOut = () => {
    signOut(auth.signOutPath);
  };

  return (
    <div className="dashboard-layout">
      <Topbar auth={auth} onSignOut={handleSignOut} />
      <div className="dashboard-body">
        <Sidebar auth={auth} />
        <MainContent onPrincipalChange={setPrincipal} />
      </div>
    </div>
  );
};

export default App;
