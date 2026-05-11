"use client";

import { useEffect, useState } from "react";
import { API_URL } from "@/lib/api-client";

interface PublicConfigData {
  registration_mode: "open" | "invite" | "closed" | "disabled";
  license_enabled: boolean;
  billing_enabled: boolean;
  stripe_public_key?: string;
}

interface PublicConfig extends PublicConfigData {
  isLoading: boolean;
}

const defaultConfig: PublicConfigData = {
  registration_mode: "open",
  license_enabled: false,
  billing_enabled: false,
};

let cachedConfig: PublicConfigData | null = null;

export function usePublicConfig() {
  const [config, setConfig] = useState<PublicConfig>(() => ({
    ...(cachedConfig ?? defaultConfig),
    isLoading: cachedConfig === null,
  }));

  useEffect(() => {
    if (cachedConfig) return;

    let cancelled = false;
    fetch(`${API_URL}/v1/config/public`, { credentials: "include" })
      .then((res) => res.json())
      .then((data: PublicConfigData) => {
        cachedConfig = data;
        if (!cancelled) {
          setConfig({ ...data, isLoading: false });
        }
      })
      .catch(() => {
        if (!cancelled) {
          setConfig({ ...defaultConfig, isLoading: false });
        }
      });

    return () => {
      cancelled = true;
    };
  }, []);

  return config;
}
