import React from "react";
import { render, screen } from "@testing-library/react";
import App from "./App";

test("renders the dashboard layout", async () => {
  const { getByText } = render(<App />);
  expect(getByText("Polaris Mission Console")).toBeInTheDocument();
  expect(getByText("Dashboard")).toBeInTheDocument();
  expect(getByText("Operator Cockpit")).toBeInTheDocument();
  expect(getByText("UniFi")).toBeInTheDocument();
  expect(await screen.findByText("OpenShift SSO")).toBeInTheDocument();
  expect(await screen.findByRole("button", { name: "Sign out" })).toBeInTheDocument();
});
