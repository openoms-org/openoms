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
      // OPE-167 / OPE-508: toast messages must be localized via t(), not hardcoded
      // string literals. All existing violations have been cleared (OPE-508), so this
      // is now `error` to block any regression at CI/auto-merge. Template-literal
      // toasts are caught by the i18n-hardcoded-copy vitest regression test instead.
      // See .interface-design/conventions.md.
      "no-restricted-syntax": [
        "error",
        {
          selector:
            "CallExpression[callee.type='MemberExpression'][callee.object.name='toast'] > Literal",
          message:
            "Toast messages must use t() from useTranslations, not hardcoded strings (OPE-167).",
        },
      ],
    },
  },
]);

export default eslintConfig;
