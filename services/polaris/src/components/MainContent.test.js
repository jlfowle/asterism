import React from "react";
import { render } from "@testing-library/react";
import MainContent from "./MainContent";

test("renders welcome message", () => {
  const { getByText } = render(<MainContent />);
  expect(getByText("Welcome to Polaris")).toBeInTheDocument();
  expect(getByText("Select a section from the sidebar to get started.")).toBeInTheDocument();
});
