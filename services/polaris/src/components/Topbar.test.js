import React from "react";
import { render } from "@testing-library/react";
import Topbar from "./Topbar";

test("renders Polaris Dashboard title", () => {
  const { getByText } = render(<Topbar />);
  expect(getByText("Polaris Dashboard")).toBeInTheDocument();
});
