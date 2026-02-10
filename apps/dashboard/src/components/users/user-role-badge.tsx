import { Badge } from "@/components/ui/badge";
import { ROLES } from "@/lib/constants";

export function UserRoleBadge({ role }: { role: string }) {
  const variants: Record<string, "default" | "secondary" | "outline"> = {
    owner: "default",
    admin: "default",
    member: "secondary",
  };

  return (
    <Badge variant={variants[role] || "outline"}>
      {ROLES[role] || role}
    </Badge>
  );
}
