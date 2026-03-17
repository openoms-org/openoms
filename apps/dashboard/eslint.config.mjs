import { defineConfig, globalIgnores } from "eslint/config";
import nextVitals from "eslint-config-next/core-web-vitals";
import nextTs from "eslint-config-next/typescript";

const eslintConfig = defineConfig([
  ...nextVitals,
  ...nextTs,
  // Override default ignores of eslint-config-next.
  globalIgnores([
    // Default ignores of eslint-config-next:
    ".next/**",
    "out/**",
    "build/**",
    "next-env.d.ts",
  ]),
  {
    rules: {
      // React 19 strict rules — must pass for auto-merge.
      "react-hooks/set-state-in-effect": "error",
      "react-hooks/incompatible-library": "error",
      "react-hooks/refs": "error",
      "react-hooks/immutability": "error",
      // TypeScript strict rules — must pass for auto-merge.
      "@typescript-eslint/no-explicit-any": "error",
      "@typescript-eslint/no-empty-object-type": "error",
    },
  },
]);

export default eslintConfig;
