"use client";

import { useWebSocket } from "@/hooks/use-websocket";
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip";

export function ConnectionStatus() {
  const { isConnected } = useWebSocket();

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="flex items-center">
            <span
              className={`inline-block h-2.5 w-2.5 rounded-full ${
                isConnected
                  ? "bg-green-500"
                  : "bg-red-500"
              }`}
            />
          </div>
        </TooltipTrigger>
        <TooltipContent>
          <p>
            {isConnected
              ? "Połączenie aktywne"
              : "Brak połączenia z serwerem"}
          </p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}
