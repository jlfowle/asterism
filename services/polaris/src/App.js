import React, { useEffect, useState } from "react";
import Topbar from "./components/Topbar";
import Sidebar from "./components/Sidebar";
import MainContent from "./components/MainContent";
import {
  buildAuthState,
  completeSignInIfNeeded,
  disabledAuthState,
  loadOidcConfig,
  signOut,
  startSignIn,
} from "./auth";
import "./App.css";

const loadingAuthState = {
  ...disabledAuthState,
  ready: false,
};

const App = () => {
  const [authConfig, setAuthConfig] = useState({ enabled: false });
  const [auth, setAuth] = useState(loadingAuthState);

  useEffect(() => {
    let isMounted = true;

    const loadAuth = async () => {
      const config = await loadOidcConfig();
      if (!isMounted) {
        return;
      }
      setAuthConfig(config);

      try {
        const tokenSet = await completeSignInIfNeeded(config);
        if (isMounted) {
          setAuth(buildAuthState(config, tokenSet));
        }
      } catch (error) {
        if (isMounted) {
          setAuth(buildAuthState(config, null, "Sign-in could not be completed."));
        }
      }
    };

    loadAuth();

    return () => {
      isMounted = false;
    };
  }, []);

  const handleSignIn = () => {
    startSignIn(authConfig).catch(() => {
      setAuth(buildAuthState(authConfig, null, "Sign-in could not be started."));
    });
  };

  const handleSignOut = () => {
    signOut(authConfig);
    setAuth(buildAuthState(authConfig, null));
  };

  return (
    <div className="dashboard-layout">
      <Topbar auth={auth} onSignIn={handleSignIn} onSignOut={handleSignOut} />
      <div className="dashboard-body">
        <Sidebar />
        <MainContent auth={auth} />
      </div>
    </div>
  );
};

export default App;
