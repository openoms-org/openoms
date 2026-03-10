"use client";

import { useTranslations } from "next-intl";
import { useWebSocket } from "@/hooks/use-websocket";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

export function ConnectionStatus() {
  const { isConnected } = useWebSocket();
  const t = useTranslations("layout.connection");

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="flex items-center">
            <span
              className={`inline-block h-2.5 w-2.5 rounded-full ${
                isConnected
                  ? "bg-success"
                  : "bg-destructive"
              }`}
            />
          </div>
        </TooltipTrigger>
        <TooltipContent>
          <p>
            {isConnected
              ? t("connected")
              : t("disconnected")}
          </p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
