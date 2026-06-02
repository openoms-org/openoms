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
import { useTranslations } from "next-intl";

export interface UserFormValues {
  email: string;
  name: string;
  role: "owner" | "admin" | "member";
  password?: string;
  confirm_password?: string;
}

function validatePassword(password: string, t: (key: string) => string): string | null {
  if (password.length < 8) return t("passwordTooShort");
  if (password.length > 72) return t("passwordTooLong");
  if (!/[A-Z]/.test(password)) return t("passwordNeedsUppercase");
  if (!/[a-z]/.test(password)) return t("passwordNeedsLowercase");
  if (!/[0-9]/.test(password)) return t("passwordNeedsDigit");
  return null;
}

function createUserSchema(t: (key: string) => string, isEdit: boolean) {
  return z.object({
    email: z.string().email(t("invalidEmail")),
    name: z.string().min(1, t("nameRequired")),
    role: z.enum(["owner", "admin", "member"], { message: t("roleRequired") }),
    password: z.string().optional(),
    confirm_password: z.string().optional(),
  }).superRefine((data, ctx) => {
    if (isEdit) return;

    if (!data.password) {
      ctx.addIssue({
        code: "custom",
        path: ["password"],
        message: t("passwordRequired"),
      });
      return;
    }

    const passwordError = validatePassword(data.password, t);
    if (passwordError) {
      ctx.addIssue({
        code: "custom",
        path: ["password"],
        message: passwordError,
      });
    }

    if (data.password !== data.confirm_password) {
      ctx.addIssue({
        code: "custom",
        path: ["confirm_password"],
        message: t("passwordMismatch"),
      });
    }
  });
}

interface UserFormProps {
  mode: "create" | "edit";
  defaultValues?: {
    email?: string;
    name?: string;
    role?: "owner" | "admin" | "member";
  };
  onSubmit: (data: UserFormValues) => void;
  isPending?: boolean;
  onCancel?: () => void;
}

export function UserForm({
  mode,
  defaultValues,
  onSubmit,
  isPending = false,
  onCancel,
}: UserFormProps) {
  const t = useTranslations("users");
  const isEdit = mode === "edit";

  const {
    register,
    handleSubmit,
    setValue,
    watch,
    formState: { errors },
  } = useForm<UserFormValues>({
    resolver: zodResolver(createUserSchema(t, isEdit)),
    defaultValues: {
      email: defaultValues?.email || "",
      name: defaultValues?.name || "",
      role: defaultValues?.role || undefined,
      password: "",
      confirm_password: "",
    },
  });

  // eslint-disable-next-line react-hooks/incompatible-library
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
            <SelectValue placeholder={t("wybierzRole")} />
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

      {!isEdit && (
        <>
          <div className="space-y-2">
            <Label htmlFor="password">{t("initialPassword")}</Label>
            <Input
              id="password"
              type="password"
              autoComplete="new-password"
              {...register("password")}
            />
            {errors.password && (
              <p className="text-sm text-destructive">{errors.password.message}</p>
            )}
          </div>

          <div className="space-y-2">
            <Label htmlFor="confirm_password">{t("confirmPassword")}</Label>
            <Input
              id="confirm_password"
              type="password"
              autoComplete="new-password"
              {...register("confirm_password")}
            />
            {errors.confirm_password && (
              <p className="text-sm text-destructive">
                {errors.confirm_password.message}
              </p>
            )}
          </div>
        </>
      )}

      <div className="flex justify-end gap-3 pt-2">
        {onCancel && (
          <Button type="button" variant="outline" onClick={onCancel} disabled={isPending}>
            Anuluj
          </Button>
        )}
        <Button type="submit" disabled={isPending}>
          {isPending
            ? "Zapisywanie..."
            : isEdit
              ? "Zapisz zmiany"
              : t("utworzUzytkownika")}
        </Button>
      </div>
    </form>
  );
}
