"use client";

import { useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { ROLES } from "@/lib/constants";

const createUserSchema = z.object({
  email: z.string().email("Nieprawidlowy adres email"),
  name: z.string().min(1, "Nazwa jest wymagana"),
  role: z.enum(["owner", "admin", "member"], "Rola jest wymagana"),
});

type CreateUserFormValues = z.infer<typeof createUserSchema>;

interface UserFormProps {
  mode: "create" | "edit";
  defaultValues?: {
    email?: string;
    name?: string;
    role?: "owner" | "admin" | "member";
  };
  onSubmit: (data: CreateUserFormValues) => void;
  isLoading?: boolean;
  onCancel?: () => void;
}

export function UserForm({
  mode,
  defaultValues,
  onSubmit,
  isLoading = false,
  onCancel,
}: UserFormProps) {
  const isEdit = mode === "edit";

  const {
    register,
    handleSubmit,
    setValue,
    watch,
    formState: { errors },
  } = useForm<CreateUserFormValues>({
    resolver: zodResolver(createUserSchema),
    defaultValues: {
      email: defaultValues?.email || "",
      name: defaultValues?.name || "",
      role: defaultValues?.role || undefined,
    },
  });

  const selectedRole = watch("role");

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="email">Email</Label>
        <Input
          id="email"
          type="email"
          placeholder="jan@example.com"
          disabled={isEdit}
          {...register("email")}
        />
        {errors.email && (
          <p className="text-sm text-destructive">{errors.email.message}</p>
        )}
      </div>

      <div className="space-y-2">
        <Label htmlFor="name">Nazwa</Label>
        <Input
          id="name"
          placeholder="Jan Kowalski"
          {...register("name")}
        />
        {errors.name && (
          <p className="text-sm text-destructive">{errors.name.message}</p>
        )}
      </div>

      <div className="space-y-2">
        <Label htmlFor="role">Rola</Label>
        <Select
          value={selectedRole}
          onValueChange={(value) =>
            setValue("role", value as "owner" | "admin" | "member", {
              shouldValidate: true,
            })
          }
        >
          <SelectTrigger className="w-full">
            <SelectValue placeholder="Wybierz role" />
          </SelectTrigger>
          <SelectContent>
            {Object.entries(ROLES).map(([value, label]) => (
              <SelectItem key={value} value={value}>
                {label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
        {errors.role && (
          <p className="text-sm text-destructive">{errors.role.message}</p>
        )}
      </div>

      <div className="flex justify-end gap-3 pt-2">
        {onCancel && (
          <Button type="button" variant="outline" onClick={onCancel} disabled={isLoading}>
            Anuluj
          </Button>
        )}
        <Button type="submit" disabled={isLoading}>
          {isLoading
            ? "Zapisywanie..."
            : isEdit
              ? "Zapisz zmiany"
              : "Utwórz użytkownika"}
        </Button>
      </div>
    </form>
  );
}
