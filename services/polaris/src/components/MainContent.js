import React, { useEffect, useMemo, useState } from "react";

const FALLBACK_SERVICES = [
  {
    id: "unifi",
    displayName: "UniFi",
    description: "Wireless controller health and client telemetry.",
    statusApi: "/api/services/unifi/api/v1/status",
    moduleManifest: "/api/services/unifi/ui/module.json",
  },
  {
    id: "cluster",
    displayName: "OpenShift",
    description: "Cluster workload and node status from in-cluster APIs.",
    statusApi: "/api/services/cluster/api/v1/status",
    moduleManifest: "/api/services/cluster/ui/module.json",
  },
  {
    id: "pfsense",
    displayName: "pfSense",
    description: "Gateway availability and network edge insights.",
    statusApi: "/api/services/pfsense/api/v1/status",
    moduleManifest: "/api/services/pfsense/ui/module.json",
  },
];

const STATUS_STATE = {
  IDLE: "idle",
  LOADING: "loading",
  READY: "ready",
  ERROR: "error",
};

const buildFallbackCard = (service) => ({
  title: service.displayName,
  description: service.description,
  links: [
    {
      label: "Open status API",
      href: service.statusApi,
    },
  ],
});

const joinServicePath = (basePath, assetPath) => {
  if (!assetPath || typeof assetPath !== "string") {
    return "";
  }

  if (/^https?:\/\//.test(assetPath)) {
    return assetPath;
  }

  if (!basePath) {
    return assetPath.startsWith("/") ? assetPath : `/${assetPath}`;
  }

  const normalizedBase = basePath.replace(/\/$/, "");
  const normalizedPath = assetPath.startsWith("/") ? assetPath : `/${assetPath}`;
  return `${normalizedBase}${normalizedPath}`;
};

const normalizeCard = (service, card) => {
  const fallback = buildFallbackCard(service);
  const links = Array.isArray(card?.links)
    ? card.links.filter((link) => link && typeof link.label === "string" && typeof link.href === "string")
    : fallback.links;

  return {
    title: card?.title || fallback.title,
    description: card?.description || fallback.description,
    links: links.length > 0 ? links : fallback.links,
  };
};

const loadRuntimeCard = async (service, manifest, modulePath, fallbackCard) => {
  if (!modulePath) {
    return normalizeCard(service, fallbackCard);
  }

  try {
    const runtimeModule = await import(/* webpackIgnore: true */ modulePath);
    const exportName = manifest.dashboardCard?.export || "createDashboardCard";
    const createDashboardCard = runtimeModule?.[exportName];

    if (typeof createDashboardCard !== "function") {
      return normalizeCard(service, fallbackCard);
    }

    const card = await createDashboardCard({
      title: fallbackCard.title,
      description: fallbackCard.description,
      displayName: service.displayName,
      statusApi: service.statusApi,
      serviceId: service.id,
      moduleManifest: manifest,
    });

    return normalizeCard(service, card);
  } catch (error) {
    return normalizeCard(service, fallbackCard);
  }
};

const enrichService = async (service) => {
  if (typeof fetch !== "function" || !service.moduleManifest) {
    return {
      ...service,
      card: buildFallbackCard(service),
    };
  }

  try {
    const response = await fetch(service.moduleManifest, {
      headers: {
        Accept: "application/json",
      },
    });

    if (!response.ok) {
      return {
        ...service,
        card: buildFallbackCard(service),
      };
    }

    const manifest = await response.json();
    const dashboardCard = manifest?.dashboardCard && typeof manifest.dashboardCard === "object"
      ? manifest.dashboardCard
      : {};
    const statusApi = joinServicePath(manifest?.apiBasePath, dashboardCard.statusEndpoint) || service.statusApi;
    const enrichedService = {
      ...service,
      displayName: manifest?.displayName || service.displayName,
      statusApi,
    };
    const fallbackCard = {
      title: dashboardCard.title || enrichedService.displayName,
      description: dashboardCard.description || service.description,
      links: [
        {
          label: "Open status API",
          href: statusApi,
        },
      ],
    };
    const modulePath = joinServicePath(manifest?.apiBasePath, dashboardCard.module);
    const card = await loadRuntimeCard(enrichedService, manifest, modulePath, fallbackCard);

    return {
      ...enrichedService,
      card,
      manifest,
    };
  } catch (error) {
    return {
      ...service,
      card: buildFallbackCard(service),
    };
  }
};

const MainContent = () => {
  const [services, setServices] = useState(() => (
    FALLBACK_SERVICES.map((service) => ({
      ...service,
      card: buildFallbackCard(service),
    }))
  ));
  const [statusByService, setStatusByService] = useState({});

  useEffect(() => {
    let isMounted = true;

    const loadServices = async () => {
      let registryServices = FALLBACK_SERVICES;

      if (typeof fetch !== "function") {
        return;
      }

      try {
        const response = await fetch("/microfrontends.json");
        if (response.ok) {
          const payload = await response.json();
          if (Array.isArray(payload.services) && payload.services.length > 0) {
            registryServices = payload.services;
          }
        }
      } catch (error) {
        // Keep static fallback list when runtime manifest lookup fails.
      }

      const enrichedServices = await Promise.all(registryServices.map((service) => enrichService(service)));
      if (isMounted) {
        setServices(enrichedServices);
      }
    };

    loadServices();

    return () => {
      isMounted = false;
    };
  }, []);

  useEffect(() => {
    if (typeof fetch !== "function") {
      return;
    }

    let isMounted = true;

    const fetchStatus = async (service) => {
      setStatusByService((current) => ({
        ...current,
        [service.id]: {
          state: STATUS_STATE.LOADING,
        },
      }));

      try {
        const response = await fetch(service.statusApi, {
          headers: {
            Accept: "application/json",
          },
        });

        if (!response.ok) {
          if (!isMounted) {
            return;
          }

          setStatusByService((current) => ({
            ...current,
            [service.id]: {
              state: STATUS_STATE.ERROR,
              error: `HTTP ${response.status}`,
            },
          }));
          return;
        }

        const payload = await response.json();
        if (!isMounted) {
          return;
        }

        setStatusByService((current) => ({
          ...current,
          [service.id]: {
            state: STATUS_STATE.READY,
            payload,
          },
        }));
      } catch (error) {
        if (!isMounted) {
          return;
        }

        setStatusByService((current) => ({
          ...current,
          [service.id]: {
            state: STATUS_STATE.ERROR,
            error: "Unavailable",
          },
        }));
      }
    };

    services.forEach((service) => {
      fetchStatus(service);
    });

    return () => {
      isMounted = false;
    };
  }, [services]);

  const liveCount = useMemo(() => {
    const entries = Object.values(statusByService);
    return entries.filter((entry) => entry?.state === STATUS_STATE.READY).length;
  }, [statusByService]);

  return (
    <main className="main-content" id="dashboard">
      <section className="hero-panel">
        <h2>Operational Dashboard</h2>
        <p>
          Runtime microfrontend modules are resolved from service manifests and rendered as
          live API-backed status cards.
        </p>
        <div className="hero-stats">
          <div className="hero-stat">
            <strong>{services.length}</strong>
            <span>Registered Services</span>
          </div>
          <div className="hero-stat">
            <strong>{liveCount}</strong>
            <span>Live Status Feeds</span>
          </div>
        </div>
      </section>

      <div className="service-grid" id="integrations">
        {services.map((service) => {
          const snapshot = statusByService[service.id] || { state: STATUS_STATE.IDLE };
          const integration = snapshot.payload?.integration || {};
          const metrics = integration.metrics && typeof integration.metrics === "object"
            ? integration.metrics
            : {};
          const card = service.card || buildFallbackCard(service);

          return (
            <article className="service-card" key={service.id}>
              <div className="service-card-top">
                <h3>{card.title}</h3>
                <span className={`service-state service-state-${snapshot.state}`}>{snapshot.state}</span>
              </div>

              <p>{card.description}</p>

              <div className="service-detail-row">
                <span>Connectivity</span>
                <strong>{integration.reachable ? "reachable" : integration.configured ? "degraded" : "not configured"}</strong>
              </div>
              <div className="service-detail-row">
                <span>Latency</span>
                <strong>{integration.latencyMs ? `${integration.latencyMs} ms` : "n/a"}</strong>
              </div>
              <div className="service-detail-row">
                <span>HTTP</span>
                <strong>{integration.httpStatus || "n/a"}</strong>
              </div>

              {Object.keys(metrics).slice(0, 3).map((key) => (
                <div className="service-detail-row" key={key}>
                  <span>{key}</span>
                  <strong>{String(metrics[key])}</strong>
                </div>
              ))}

              <p className="service-message">{integration.message || snapshot.error || "Waiting for first status poll."}</p>
              {card.links.map((link) => (
                <a href={link.href} key={`${service.id}-${link.href}`}>
                  {link.label}
                </a>
              ))}
            </article>
          );
        })}
      </div>
    </main>
  );
};

export default MainContent;
