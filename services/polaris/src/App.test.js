import React from "react";
import { render, screen } from "@testing-library/react";
import App from "./App";

test("renders the dashboard layout", async () => {
  const originalFetch = global.fetch;
  global.fetch = undefined;

  try {
    const { getByText } = render(<App />);
    expect(getByText("Polaris Mission Console")).toBeInTheDocument();
    expect(getByText("Dashboard")).toBeInTheDocument();
    expect(getByText("Operator Cockpit")).toBeInTheDocument();
    expect(getByText("UniFi")).toBeInTheDocument();
    expect(await screen.findByText("Local development")).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Sign out" })).not.toBeInTheDocument();
  } finally {
    global.fetch = originalFetch;
  }
});
