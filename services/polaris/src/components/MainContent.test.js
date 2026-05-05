import React from "react";
import { render, screen } from "@testing-library/react";
import MainContent from "./MainContent";

test("renders welcome message", () => {
  const { getByText } = render(<MainContent />);
  expect(getByText("Operator Cockpit")).toBeInTheDocument();
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
              moduleManifest: "/ui/services/unifi/module.json",
            },
          ],
        }),
      });
    }

    if (url === "/ui/services/unifi/module.json") {
      return Promise.resolve({
        ok: true,
        json: async () => ({
          service: "unifi",
          displayName: "UniFi Integration",
          apiBasePath: "/api/services/unifi",
          uiBasePath: "/ui/services/unifi",
          dashboardCard: {
            title: "UniFi Network",
            description: "Wireless, network device, and client status from the local UniFi Network API.",
            statusEndpoint: "/api/v1/status",
            module: "/dashboard-card.js",
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

  expect(await screen.findByText("UniFi Network")).toBeInTheDocument();
  expect(await screen.findByText("Wireless, network device, and client status from the local UniFi Network API.")).toBeInTheDocument();

  global.fetch = originalFetch;
});

test("sends bearer token when fetching protected service status", async () => {
  const originalFetch = global.fetch;
  const statusFetches = [];

  global.fetch = jest.fn((url, options = {}) => {
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
              moduleManifest: "/ui/services/unifi/module.json",
            },
          ],
        }),
      });
    }

    if (url === "/api/services/unifi/api/v1/status") {
      statusFetches.push(options);
      return Promise.resolve({
        ok: true,
        json: async () => ({
          integration: {
            configured: true,
            reachable: true,
            severity: "ok",
            observedAt: "2026-05-05T12:00:00Z",
            message: "UniFi Network is reachable.",
            summary: {
              stats: [
                { label: "Devices online", value: "2/2", state: "ok" },
              ],
            },
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

  render(<MainContent auth={{
    ready: true,
    enabled: true,
    authenticated: true,
    authorizationHeader: "Bearer test-token",
  }} />);

  expect(await screen.findByText("healthy")).toBeInTheDocument();
  expect(statusFetches[0].headers.Authorization).toBe("Bearer test-token");

  global.fetch = originalFetch;
});
