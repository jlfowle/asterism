import React from "react";
import { render } from "@testing-library/react";
import Topbar from "./Topbar";

test("renders Polaris mission console title", () => {
  const { getByText } = render(<Topbar />);
  expect(getByText("Polaris Mission Console")).toBeInTheDocument();
});
