import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { SubscriptionBanner } from "@/components/subscription-banner";
import { useAuthStore } from "@/lib/auth";
import type { SubscriptionStatus } from "@/types/api";

const useSubscriptionMock = vi.hoisted(() => vi.fn());

vi.mock("@/hooks/use-billing", () => ({
  useSubscription: useSubscriptionMock,
}));

vi.mock("next-intl", () => ({
  useTranslations: () => (key: string, values?: Record<string, unknown>) =>
    values?.date ? `${key}:${values.date}` : key,
}));

const tenant = {
  id: "tenant-1",
  name: "OpenOMS",
  slug: "openoms",
  plan: "plus",
};

function setSubscription(subscription?: SubscriptionStatus) {
  useSubscriptionMock.mockReturnValue({ data: subscription });
}

describe("SubscriptionBanner", () => {
  beforeEach(() => {
    vi.unstubAllEnvs();
    useSubscriptionMock.mockReset();
    setSubscription();
    useAuthStore.setState({
      token: "token",
      tenant: tenant as never,
      user: null,
      isAuthenticated: true,
      isLoading: false,
      locale: "pl",
    });
  });

  it("does not render without tenant context", () => {
    useAuthStore.setState({ tenant: null });
    setSubscription({
      plan: "plus",
      status: "trialing",
      trial_end: "2099-01-10T00:00:00Z",
    });

    const { container } = render(<SubscriptionBanner />);

    expect(container).toBeEmptyDOMElement();
  });

  it("does not link to hidden billing settings in client-ready mode", () => {
    setSubscription({
      plan: "plus",
      status: "trialing",
      trial_end: "2099-01-10T00:00:00Z",
    });

    render(<SubscriptionBanner />);

    expect(screen.getByText("subscriptionManagedBySupport")).toBeInTheDocument();
    expect(
      screen.queryByRole("link", { name: "zarzadzajSubskrypcja" }),
    ).not.toBeInTheDocument();
  });

  it("links to billing settings only in full dashboard mode", () => {
    vi.stubEnv("NEXT_PUBLIC_OPENOMS_DASHBOARD_SURFACE", "full");
    setSubscription({
      plan: "plus",
      status: "trialing",
      trial_end: "2099-01-10T00:00:00Z",
    });

    render(<SubscriptionBanner />);

    expect(
      screen.getByRole("link", { name: "zarzadzajSubskrypcja" }),
    ).toHaveAttribute("href", "/settings/billing");
  });

  it("renders inactive subscription support copy without hidden billing links", () => {
    setSubscription({
      plan: "plus",
      status: "canceled",
      current_period_end: "2099-01-10T00:00:00Z",
    });

    render(<SubscriptionBanner />);

    expect(screen.getByText("renewViaSupport")).toBeInTheDocument();
    expect(
      screen.queryByRole("link", { name: "odnowSubskrypcje" }),
    ).not.toBeInTheDocument();
  });
});
