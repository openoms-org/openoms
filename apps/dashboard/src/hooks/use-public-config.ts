"use client";

import { useEffect, useState } from "react";
import { API_URL } from "@/lib/api-client";

interface PublicConfig {
  registration_mode: "open" | "invite" | "disabled";
}

const defaultConfig: PublicConfig = { registration_mode: "open" };

let cachedConfig: PublicConfig | null = null;

export function usePublicConfig() {
  const [config, setConfig] = useState<PublicConfig>(cachedConfig ?? defaultConfig);

  useEffect(() => {
    if (cachedConfig) return;

    fetch(`${API_URL}/v1/config/public`)
      .then((res) => res.json())
      .then((data: PublicConfig) => {
        cachedConfig = data;
        setConfig(data);
      })
      .catch(() => {
        // fallback to open if config endpoint unavailable
      });
  }, []);

  return config;
}
