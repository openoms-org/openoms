export const integrationQueryKeys = {
  all: ["integrations"] as const,
  detail: (id: string) => [...integrationQueryKeys.all, id] as const,
};
