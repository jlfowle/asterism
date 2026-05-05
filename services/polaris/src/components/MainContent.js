import React, { useEffect, useMemo, useState } from "react";

const FALLBACK_SERVICES = [
  {
    id: "unifi",
    displayName: "UniFi",
    description: "Wireless controller health and client telemetry.",
    statusApi: "/api/services/unifi/api/v1/status",
    moduleManifest: "/ui/services/unifi/module.json",
  },
  {
    id: "cluster",
    displayName: "OpenShift",
    description: "Cluster workload and node status from in-cluster APIs.",
    statusApi: "/api/services/cluster/api/v1/status",
    moduleManifest: "/ui/services/cluster/module.json",
  },
  {
    id: "pfsense",
    displayName: "pfSense",
    description: "Gateway availability and network edge insights.",
    statusApi: "/api/services/pfsense/api/v1/status",
    moduleManifest: "/ui/services/pfsense/module.json",
  },
];

const STATUS_STATE = {
  IDLE: "idle",
  LOADING: "loading",
  READY: "ready",
  ERROR: "error",
};

const severityFromSnapshot = (snapshot) => {
  const integration = snapshot.payload?.integration || {};
  if (integration.severity) {
    return integration.severity;
  }
  if (snapshot.state === STATUS_STATE.ERROR) {
    return "critical";
  }
  if (!integration.configured) {
    return "unknown";
  }
  return integration.reachable ? "ok" : "critical";
};

const severityLabel = (severity) => {
  if (severity === "ok") {
    return "healthy";
  }
  if (severity === "warning") {
    return "review";
  }
  if (severity === "critical") {
    return "attention";
  }
  return "pending";
};

const authHeaders = (auth) => (
  auth?.authorizationHeader ? { Authorization: auth.authorizationHeader } : {}
);

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
    const modulePath = joinServicePath(manifest?.uiBasePath || manifest?.apiBasePath, dashboardCard.module);
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

const MainContent = ({ auth = { ready: true, enabled: false } }) => {
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
    if (!auth.ready) {
      return;
    }
    if (auth.enabled && !auth.authenticated) {
      setStatusByService(Object.fromEntries(services.map((service) => [
        service.id,
        {
          state: STATUS_STATE.ERROR,
          error: "Sign in required",
        },
      ])));
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
            ...authHeaders(auth),
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
  }, [services, auth.ready, auth.enabled, auth.authenticated, auth.authorizationHeader]);

  const liveCount = useMemo(() => {
    const entries = Object.values(statusByService);
    return entries.filter((entry) => severityFromSnapshot(entry) === "ok").length;
  }, [statusByService]);

  const reviewCount = useMemo(() => {
    const entries = Object.values(statusByService);
    return entries.filter((entry) => ["warning", "critical"].includes(severityFromSnapshot(entry))).length;
  }, [statusByService]);

  return (
    <main className="main-content" id="dashboard">
      <section className="hero-panel">
        <h2>Operator Cockpit</h2>
        <p>
          Plain-language status from UniFi, pfSense, OpenShift, and GitOps, with backend-owned
          configuration left in the systems that already manage it.
        </p>
        <div className="hero-stats">
          <div className="hero-stat">
            <strong>{services.length}</strong>
            <span>Registered Services</span>
          </div>
          <div className="hero-stat">
            <strong>{liveCount}</strong>
            <span>Healthy Services</span>
          </div>
          <div className="hero-stat">
            <strong>{reviewCount}</strong>
            <span>Need Review</span>
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
          const summary = integration.summary || {};
          const stats = Array.isArray(summary.stats) ? summary.stats : [];
          const highlights = Array.isArray(summary.highlights) ? summary.highlights : [];
          const degradedReasons = Array.isArray(integration.degradedReasons) ? integration.degradedReasons : [];
          const recommendedActions = Array.isArray(integration.recommendedActions) ? integration.recommendedActions : [];
          const authoritativeLinks = Array.isArray(integration.authoritativeLinks) ? integration.authoritativeLinks : [];
          const controls = Array.isArray(integration.controls) ? integration.controls : [];
          const severity = severityFromSnapshot(snapshot);
          const card = service.card || buildFallbackCard(service);

          return (
            <article className={`service-card service-card-${severity}`} key={service.id}>
              <div className="service-card-top">
                <h3>{card.title}</h3>
                <span className={`service-state service-state-${severity}`}>{severityLabel(severity)}</span>
              </div>

              <p>{summary.description || card.description}</p>

              {stats.length > 0 && (
                <div className="service-stat-grid">
                  {stats.slice(0, 4).map((stat) => (
                    <div className={`service-stat service-stat-${stat.state || "neutral"}`} key={`${service.id}-${stat.label}`}>
                      <span>{stat.label}</span>
                      <strong>{stat.value}{stat.unit ? ` ${stat.unit}` : ""}</strong>
                    </div>
                  ))}
                </div>
              )}

              {stats.length === 0 && (
                <>
                  <div className="service-detail-row">
                    <span>Connectivity</span>
                    <strong>{integration.reachable ? "reachable" : integration.configured ? "degraded" : "not configured"}</strong>
                  </div>
                  <div className="service-detail-row">
                    <span>Latency</span>
                    <strong>{integration.latencyMs ? `${integration.latencyMs} ms` : "n/a"}</strong>
                  </div>
                </>
              )}

              {Object.keys(metrics).slice(0, 3).map((key) => (
                <div className="service-detail-row" key={key}>
                  <span>{key}</span>
                  <strong>{String(metrics[key])}</strong>
                </div>
              ))}

              <div className="service-detail-row">
                <span>Last update</span>
                <strong>{integration.observedAt ? new Date(integration.observedAt).toLocaleTimeString() : "waiting"}</strong>
              </div>

              <p className="service-message">{integration.message || snapshot.error || "Waiting for first status poll."}</p>

              {highlights.length > 0 && (
                <ul className="status-list">
                  {highlights.slice(0, 3).map((item) => (
                    <li key={`${service.id}-highlight-${item}`}>{item}</li>
                  ))}
                </ul>
              )}

              {degradedReasons.length > 0 && (
                <div className="review-panel">
                  <strong>Review</strong>
                  <ul>
                    {degradedReasons.slice(0, 3).map((reason) => (
                      <li key={`${service.id}-reason-${reason}`}>{reason}</li>
                    ))}
                  </ul>
                </div>
              )}

              {recommendedActions.length > 0 && (
                <div className="next-steps">
                  {recommendedActions.slice(0, 2).map((action) => (
                    <span key={`${service.id}-action-${action}`}>{action}</span>
                  ))}
                </div>
              )}

              {controls.length > 0 && (
                <div className="control-row">
                  {controls.slice(0, 2).map((control) => (
                    <button
                      type="button"
                      disabled={!control.enabled}
                      title={control.description}
                      key={control.id}
                    >
                      {control.label}
                    </button>
                  ))}
                </div>
              )}

              <div className="link-row">
                {[...authoritativeLinks, ...card.links].slice(0, 3).map((link) => (
                  <a href={link.href} key={`${service.id}-${link.href}`}>
                    {link.label}
                  </a>
                ))}
              </div>
            </article>
          );
        })}
      </div>
    </main>
  );
};

export default MainContent;
