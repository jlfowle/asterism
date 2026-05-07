import React from "react";
import { render } from "@testing-library/react";
import Topbar from "./Topbar";

test("renders local development auth state when edge auth is disabled", () => {
  const { getByText, queryByRole } = render(<Topbar auth={{ ready: true, enabled: false, modeLabel: "Local development" }} />);
  expect(getByText("Polaris Mission Console")).toBeInTheDocument();
  expect(getByText("Local development")).toBeInTheDocument();
  expect(queryByRole("button", { name: "Sign out" })).not.toBeInTheDocument();
});

test("renders OpenShift sign-out when edge auth is enabled", () => {
  const { getByText, getByRole } = render(<Topbar auth={{ ready: true, enabled: true, modeLabel: "OpenShift SSO", principal: "" }} onSignOut={() => {}} />);
  expect(getByText("Polaris Mission Console")).toBeInTheDocument();
  expect(getByText("OpenShift SSO")).toBeInTheDocument();
  expect(getByRole("button", { name: "Sign out" })).toBeInTheDocument();
});
