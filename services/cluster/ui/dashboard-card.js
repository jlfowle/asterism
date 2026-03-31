export function createDashboardCard({ title, description, statusApi }) {
  return {
    title,
    description,
    links: [
      {
        label: "View status API",
        href: statusApi,
      },
    ],
  };
}
