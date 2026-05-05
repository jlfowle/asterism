export function createDashboardCard({ title, description, statusApi }) {
  return {
    title,
    description,
    links: [
      {
        label: "Status API",
        href: statusApi,
      },
    ],
  };
}
