"use client";

import { useEffect, useState } from "react";
import { API_URL } from "@/lib/api-client";

interface PublicConfig {
  registration_mode: "open" | "invite" | "closed" | "disabled";
  license_enabled: boolean;
  billing_enabled: boolean;
  stripe_public_key?: string;
}

const defaultConfig: PublicConfig = {
  registration_mode: "open",
  license_enabled: false,
  billing_enabled: false,
};

let cachedConfig: PublicConfig | null = null;

export function usePublicConfig() {
  const [config, setConfig] = useState<PublicConfig>(cachedConfig ?? defaultConfig);

  useEffect(() => {
    if (cachedConfig) return;

    fetch(`${API_URL}/v1/config/public`, { credentials: "include" })
      .then((res) => res.json())
      .then((data: PublicConfig) => {
        cachedConfig = data;
        setConfig(data);
      })
      .catch(() => {
        // fallback to defaults if config endpoint unavailable
      });
  }, []);

  return config;
}
