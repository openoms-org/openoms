import { z } from "zod";

export const integrationSchema = z.object({
  provider: z.string().min(1),
  settings: z.string().optional().refine(
    (val) => {
      if (!val || val.trim() === "") return true;
      try {
        JSON.parse(val);
        return true;
      } catch {
        return false;
      }
    },
    { message: "Invalid JSON format" }
  ),
});

export type IntegrationFormValues = z.infer<typeof integrationSchema>;
