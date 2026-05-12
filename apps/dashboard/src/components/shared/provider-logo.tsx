"use client";

import { Factory, Package, Receipt, Store, Truck } from "lucide-react";
import { cn } from "@/lib/utils";
import {
  getProviderDisplayName,
  getProviderInfo,
  type ProviderInfo,
} from "@/lib/provider-info";

type ProviderLogoSize = "sm" | "md" | "lg";

interface ProviderLogoProps {
  provider?: ProviderInfo;
  providerKey?: string;
  fallbackName?: string;
  category?: ProviderInfo["category"];
  size?: ProviderLogoSize;
  showName?: boolean;
  className?: string;
}

const categoryIcons: Record<ProviderInfo["category"], typeof Store> = {
  marketplace: Store,
  carrier: Truck,
  invoicing: Receipt,
  supplier: Factory,
};

const sizeClasses: Record<
  ProviderLogoSize,
  {
    mark: string;
    icon: string;
    wrapper: string;
    text: string;
  }
> = {
  sm: {
    mark: "h-8 min-w-8 rounded-md px-2 text-[10px]",
    icon: "h-3 w-3",
    wrapper: "gap-2",
    text: "text-sm",
  },
  md: {
    mark: "h-10 min-w-10 rounded-md px-2.5 text-xs",
    icon: "h-3.5 w-3.5",
    wrapper: "gap-3",
    text: "text-sm",
  },
  lg: {
    mark: "h-12 min-w-12 rounded-lg px-3 text-sm",
    icon: "h-4 w-4",
    wrapper: "gap-3",
    text: "text-base",
  },
};

const fallbackBrandClass =
  "border-border bg-muted text-muted-foreground";

export function ProviderLogo({
  provider,
  providerKey,
  fallbackName,
  category,
  size = "md",
  showName = false,
  className,
}: ProviderLogoProps) {
  const resolvedProvider = provider ?? (providerKey ? getProviderInfo(providerKey) : undefined);
  const key = resolvedProvider?.key ?? providerKey ?? "unknown";
  const name =
    resolvedProvider?.name ??
    fallbackName ??
    (providerKey ? getProviderDisplayName(providerKey) : "Provider");
  const resolvedCategory = resolvedProvider?.category ?? category;
  const brand = resolvedProvider?.brand;
  const officialAsset = brand?.officialAsset;
  const Icon =
    resolvedProvider?.icon ??
    (resolvedCategory ? categoryIcons[resolvedCategory] : Package);
  const logoText = brand
    ? showName && size === "sm"
      ? brand.shortMark
      : brand.wordmark
    : initialsForName(name);
  const classes = sizeClasses[size];

  const mark = officialAsset ? (
    <span
      title={name}
      data-provider-key={key}
      className={cn(
        "inline-flex shrink-0 items-center justify-center border bg-white shadow-xs",
        classes.mark,
        !showName && className,
      )}
    >
      <img
        src={officialAsset.src}
        alt={`${name} logo`}
        data-provider-key={key}
        className="h-full max-h-6 w-auto object-contain"
      />
    </span>
  ) : (
    <span
      role="img"
      aria-label={`${name} logo`}
      title={name}
      data-provider-key={key}
      className={cn(
        "inline-flex shrink-0 items-center justify-center border font-semibold leading-none tracking-normal shadow-xs",
        classes.mark,
        brand?.className ?? fallbackBrandClass,
        !brand && "gap-1.5",
        !showName && className,
      )}
    >
      {!brand && <Icon className={classes.icon} aria-hidden="true" />}
      <span>{logoText}</span>
    </span>
  );

  if (!showName) {
    return mark;
  }

  return (
    <span className={cn("inline-flex min-w-0 items-center", classes.wrapper, className)}>
      {mark}
      <span className={cn("min-w-0 truncate font-medium text-foreground", classes.text)}>
        {name}
      </span>
    </span>
  );
}

function initialsForName(name: string): string {
  const normalized = name.replace(/[_-]+/g, " ").trim();
  const words = normalized.split(/\s+/).filter(Boolean);

  if (words.length >= 2) {
    return words
      .slice(0, 2)
      .map((word) => word.charAt(0).toUpperCase())
      .join("");
  }

  return normalized.slice(0, 2).toUpperCase() || "?";
}
