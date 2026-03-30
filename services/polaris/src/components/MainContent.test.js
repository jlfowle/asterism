import React from "react";
import { render, screen } from "@testing-library/react";
import MainContent from "./MainContent";

test("renders welcome message", () => {
  const { getByText } = render(<MainContent />);
  expect(getByText("Operational Dashboard")).toBeInTheDocument();
  expect(getByText("UniFi")).toBeInTheDocument();
  expect(getByText("OpenShift")).toBeInTheDocument();
  expect(getByText("pfSense")).toBeInTheDocument();
});

test("hydrates cards from service-owned module manifests", async () => {
  const originalFetch = global.fetch;

  global.fetch = jest.fn((url) => {
    if (url === "/microfrontends.json") {
      return Promise.resolve({
        ok: true,
        json: async () => ({
          services: [
            {
              id: "unifi",
              displayName: "UniFi",
              description: "Fallback description",
              statusApi: "/api/services/unifi/api/v1/status",
              moduleManifest: "/api/services/unifi/ui/module.json",
            },
          ],
        }),
      });
    }

    if (url === "/api/services/unifi/ui/module.json") {
      return Promise.resolve({
        ok: true,
        json: async () => ({
          service: "unifi",
          displayName: "UniFi Integration",
          apiBasePath: "/api/services/unifi",
          dashboardCard: {
            title: "UniFi Status",
            description: "Runtime-loaded card owned by the unifi service.",
            statusEndpoint: "/api/v1/status",
            module: "/ui/dashboard-card.js",
            export: "createDashboardCard",
          },
        }),
      });
    }

    if (url === "/api/services/unifi/api/v1/status") {
      return Promise.resolve({
        ok: true,
        json: async () => ({
          integration: {
            configured: true,
            reachable: true,
            latencyMs: 18,
            httpStatus: 200,
            message: "Connected",
            metrics: {},
          },
        }),
      });
    }

    return Promise.resolve({
      ok: false,
      json: async () => ({}),
    });
  });

  render(<MainContent />);

  expect(await screen.findByText("UniFi Status")).toBeInTheDocument();
  expect(await screen.findByText("Runtime-loaded card owned by the unifi service.")).toBeInTheDocument();

  global.fetch = originalFetch;
});
