import React from "react";
import { render, screen } from "@testing-library/react";
import Sidebar from "./Sidebar";

test("renders sidebar links", () => {
  render(<Sidebar auth={{ ready: true, modeLabel: "OpenShift SSO" }} />);
  expect(screen.getByText("Dashboard")).toBeInTheDocument();
  expect(screen.getByText("Integrations")).toBeInTheDocument();
  expect(screen.queryByText("Security")).toBeNull();
  expect(screen.getByText(/External identity: OpenShift SSO/)).toBeInTheDocument();
});
