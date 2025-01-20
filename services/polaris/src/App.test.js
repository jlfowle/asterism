import React from "react";
import { render } from "@testing-library/react";
import App from "./App";

test("renders the dashboard layout", () => {
  const { getByText } = render(<App />);
  expect(getByText("Polaris Dashboard")).toBeInTheDocument();
  expect(getByText("Dashboard")).toBeInTheDocument();
  expect(getByText("Welcome to Polaris")).toBeInTheDocument();
});
