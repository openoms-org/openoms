"use client";

import { useEffect, useCallback } from "react";
import { useRouter } from "next/navigation";

interface UseKeyboardShortcutsOptions {
  onToggleCommandPalette: () => void;
}

export function useKeyboardShortcuts({
  onToggleCommandPalette,
}: UseKeyboardShortcutsOptions) {
  const router = useRouter();

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      const target = e.target as HTMLElement;
      const isInput =
        target.tagName === "INPUT" ||
        target.tagName === "TEXTAREA" ||
        target.tagName === "SELECT" ||
        target.isContentEditable;

      // Ctrl+K / Cmd+K — open command palette (always works)
      if ((e.ctrlKey || e.metaKey) && e.key === "k") {
        e.preventDefault();
        onToggleCommandPalette();
        return;
      }

      // Skip other shortcuts if user is typing in an input
      if (isInput) return;

      // Ctrl+N / Cmd+N — new order
      if ((e.ctrlKey || e.metaKey) && !e.shiftKey && e.key === "n") {
        e.preventDefault();
        router.push("/orders/new");
        return;
      }

      // Ctrl+Shift+N / Cmd+Shift+N — new product
      if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key === "N") {
        e.preventDefault();
        router.push("/products/new");
        return;
      }

      // Alt+1 — Dashboard
      if (e.altKey && e.key === "1") {
        e.preventDefault();
        router.push("/");
        return;
      }

      // Alt+2 — Orders
      if (e.altKey && e.key === "2") {
        e.preventDefault();
        router.push("/orders");
        return;
      }

      // Alt+3 — Products
      if (e.altKey && e.key === "3") {
        e.preventDefault();
        router.push("/products");
        return;
      }
    },
    [onToggleCommandPalette, router]
  );

  useEffect(() => {
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [handleKeyDown]);
}
