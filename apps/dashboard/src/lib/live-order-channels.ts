const NON_LIVE_ORDER_CHANNELS = new Set(["kaufland", "empik", "mirakl"]);

export function isLiveOrderChannel(provider: string): boolean {
  return !NON_LIVE_ORDER_CHANNELS.has(provider);
}

export function liveOrderSources<T extends string>(sources: readonly T[]): T[] {
  return sources.filter((source) => isLiveOrderChannel(source));
}

export function liveMarketplaceIntegrations<T extends { provider: string }>(
  integrations: readonly T[],
): T[] {
  return integrations.filter((integration) =>
    isLiveOrderChannel(integration.provider),
  );
}
