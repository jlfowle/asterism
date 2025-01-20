import React from "react";
import Topbar from "./components/Topbar";
import Sidebar from "./components/Sidebar";
import MainContent from "./components/MainContent";
import "./App.css";

const App = () => (
  <div className="dashboard-layout">
    <Topbar />
    <div className="dashboard-body">
      <Sidebar />
      <MainContent />
    </div>
  </div>
);

export default App;
