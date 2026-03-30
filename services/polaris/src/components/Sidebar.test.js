import React from "react";
import { render } from "@testing-library/react";
import Sidebar from "./Sidebar";

test("renders sidebar links", () => {
  const { getByText } = render(<Sidebar />);
  expect(getByText("Dashboard")).toBeInTheDocument();
  expect(getByText("Integrations")).toBeInTheDocument();
  expect(getByText("Security")).toBeInTheDocument();
});
