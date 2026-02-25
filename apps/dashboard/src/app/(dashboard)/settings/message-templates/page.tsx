"use client";

import { useState } from "react";
import { Plus, Pencil, Trash2, MessageSquare } from "lucide-react";
import { toast } from "sonner";
import { AdminGuard } from "@/components/shared/admin-guard";
import { EmptyState } from "@/components/shared/empty-state";
import { LoadingSkeleton } from "@/components/shared/loading-skeleton";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { getErrorMessage } from "@/lib/api-client";
import { formatDate } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Switch } from "@/components/ui/switch";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  useMessageTemplates,
  useCreateMessageTemplate,
  useUpdateMessageTemplate,
  useDeleteMessageTemplate,
} from "@/hooks/use-message-templates";
import type { MessageTemplate } from "@/types/api";

const CHANNEL_LABELS: Record<string, string> = {
  allegro: "Allegro",
  email: "Email",
  sms: "SMS",
};

const CHANNEL_COLORS: Record<string, string> = {
  allegro:
    "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200",
  email:
    "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200",
  sms:
    "bg-orange-100 text-orange-800 dark:bg-orange-900 dark:text-orange-200",
};

interface FormState {
  name: string;
  channel: string;
  subject: string;
  body: string;
  variables: string;
  enabled: boolean;
}

const emptyForm: FormState = {
  name: "",
  channel: "allegro",
  subject: "",
  body: "",
  variables: "",
  enabled: true,
};

function templateToForm(t: MessageTemplate): FormState {
  return {
    name: t.name,
    channel: t.channel,
    subject: t.subject,
    body: t.body,
    variables: (t.variables ?? []).join(", "),
    enabled: t.enabled,
  };
}

export default function MessageTemplatesPage() {
  const { data, isLoading, isError, refetch } = useMessageTemplates();
  const createTemplate = useCreateMessageTemplate();
  const deleteTemplate = useDeleteMessageTemplate();

  const [dialogOpen, setDialogOpen] = useState(false);
  const [editingTemplate, setEditingTemplate] =
    useState<MessageTemplate | null>(null);
  const [deleteId, setDeleteId] = useState<string | null>(null);
  const [form, setForm] = useState<FormState>(emptyForm);

  const openCreate = () => {
    setEditingTemplate(null);
    setForm(emptyForm);
    setDialogOpen(true);
  };

  const openEdit = (template: MessageTemplate) => {
    setEditingTemplate(template);
    setForm(templateToForm(template));
    setDialogOpen(true);
  };

  const closeDialog = () => {
    setDialogOpen(false);
    setEditingTemplate(null);
    setForm(emptyForm);
  };

  const parseVariables = (raw: string): string[] =>
    raw
      .split(",")
      .map((v) => v.trim())
      .filter(Boolean);

  // We need the update hook at the top level — the id is known from editingTemplate
  const updateTemplate = useUpdateMessageTemplate(
    editingTemplate?.id ?? ""
  );

  const handleSubmit = () => {
    if (!form.name.trim() || !form.body.trim()) return;

    const payload = {
      name: form.name.trim(),
      channel: form.channel,
      subject: form.subject.trim() || undefined,
      body: form.body.trim(),
      variables: parseVariables(form.variables),
      enabled: form.enabled,
    };

    if (editingTemplate) {
      updateTemplate.mutate(payload, {
        onSuccess: () => {
          toast.success("Szablon został zaktualizowany");
          closeDialog();
        },
        onError: (error) => {
          toast.error(getErrorMessage(error));
        },
      });
    } else {
      createTemplate.mutate(payload, {
        onSuccess: () => {
          toast.success("Szablon został utworzony");
          closeDialog();
        },
        onError: (error) => {
          toast.error(getErrorMessage(error));
        },
      });
    }
  };

  const handleDelete = () => {
    if (!deleteId) return;
    deleteTemplate.mutate(deleteId, {
      onSuccess: () => {
        toast.success("Szablon został usunięty");
        setDeleteId(null);
      },
      onError: (error) => {
        toast.error(getErrorMessage(error));
      },
    });
  };

  const isSaving = createTemplate.isPending || updateTemplate.isPending;

  if (isLoading) {
    return <LoadingSkeleton />;
  }

  const templates = data?.items ?? [];

  return (
    <AdminGuard>
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">
            Szablony wiadomości
          </h1>
          <p className="text-muted-foreground">
            Zarządzaj szablonami wiadomości dla automatyzacji i komunikacji
            z klientami
          </p>
        </div>
        <Button onClick={openCreate}>
          <Plus className="h-4 w-4 mr-2" />
          Nowy szablon
        </Button>
      </div>

      {isError && (
        <div className="rounded-md border border-destructive bg-destructive/10 p-4 mb-6">
          <p className="text-sm text-destructive">
            Wystąpił błąd podczas ładowania danych. Spróbuj odświeżyć
            stronę.
          </p>
          <Button
            variant="outline"
            size="sm"
            className="mt-2"
            onClick={() => refetch()}
          >
            Spróbuj ponownie
          </Button>
        </div>
      )}

      {templates.length === 0 ? (
        <EmptyState
          icon={MessageSquare}
          title="Brak szablonów"
          description="Utwórz pierwszy szablon wiadomości, aby używać go w automatyzacjach i komunikacji z klientami."
        />
      ) : (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Nazwa</TableHead>
                <TableHead>Kanał</TableHead>
                <TableHead>Temat</TableHead>
                <TableHead>Aktywny</TableHead>
                <TableHead>Utworzono</TableHead>
                <TableHead className="w-[80px]" />
              </TableRow>
            </TableHeader>
            <TableBody>
              {templates.map((template) => (
                <TableRow key={template.id}>
                  <TableCell className="font-medium">
                    {template.name}
                  </TableCell>
                  <TableCell>
                    <Badge
                      variant="outline"
                      className={
                        CHANNEL_COLORS[template.channel] ?? ""
                      }
                    >
                      {CHANNEL_LABELS[template.channel] ??
                        template.channel}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    {template.subject || (
                      <span className="text-muted-foreground">
                        ---
                      </span>
                    )}
                  </TableCell>
                  <TableCell>
                    {template.enabled ? (
                      <Badge
                        variant="outline"
                        className="bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-200"
                      >
                        Aktywny
                      </Badge>
                    ) : (
                      <Badge
                        variant="outline"
                        className="bg-gray-100 text-gray-800 dark:bg-gray-800 dark:text-gray-200"
                      >
                        Nieaktywny
                      </Badge>
                    )}
                  </TableCell>
                  <TableCell>
                    {formatDate(template.created_at)}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-1">
                      <Button
                        variant="ghost"
                        size="icon-xs"
                        onClick={() => openEdit(template)}
                      >
                        <Pencil className="h-4 w-4" />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon-xs"
                        onClick={() => setDeleteId(template.id)}
                      >
                        <Trash2 className="h-4 w-4 text-destructive" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      {/* Create / Edit dialog */}
      <Dialog open={dialogOpen} onOpenChange={(open) => !open && closeDialog()}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {editingTemplate ? "Edytuj szablon" : "Nowy szablon"}
            </DialogTitle>
            <DialogDescription>
              {editingTemplate
                ? "Zaktualizuj szczegóły szablonu wiadomości"
                : "Dodaj nowy szablon wiadomości do systemu"}
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="tpl-name">Nazwa</Label>
              <Input
                id="tpl-name"
                value={form.name}
                onChange={(e) =>
                  setForm((prev) => ({ ...prev, name: e.target.value }))
                }
                placeholder="np. Potwierdzenie wysyłki"
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="tpl-channel">Kanał</Label>
              <Select
                value={form.channel}
                onValueChange={(value) =>
                  setForm((prev) => ({ ...prev, channel: value }))
                }
              >
                <SelectTrigger id="tpl-channel">
                  <SelectValue placeholder="Wybierz kanał" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="allegro">Allegro</SelectItem>
                  <SelectItem value="email">Email</SelectItem>
                  <SelectItem value="sms">SMS</SelectItem>
                </SelectContent>
              </Select>
            </div>

            {form.channel === "email" && (
              <div className="space-y-2">
                <Label htmlFor="tpl-subject">Temat</Label>
                <Input
                  id="tpl-subject"
                  value={form.subject}
                  onChange={(e) =>
                    setForm((prev) => ({
                      ...prev,
                      subject: e.target.value,
                    }))
                  }
                  placeholder="np. Zamówienie {{order_number}} — aktualizacja"
                />
              </div>
            )}

            <div className="space-y-2">
              <Label htmlFor="tpl-body">Treść</Label>
              <Textarea
                id="tpl-body"
                value={form.body}
                onChange={(e) =>
                  setForm((prev) => ({ ...prev, body: e.target.value }))
                }
                placeholder="Treść wiadomości..."
                rows={5}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="tpl-variables">Zmienne</Label>
              <Input
                id="tpl-variables"
                value={form.variables}
                onChange={(e) =>
                  setForm((prev) => ({
                    ...prev,
                    variables: e.target.value,
                  }))
                }
                placeholder="order_number, customer_name, tracking_number"
              />
              <p className="text-xs text-muted-foreground">
                Dostępne zmienne: {"{{order_number}}"},{" "}
                {"{{customer_name}}"}, {"{{tracking_number}}"},{" "}
                {"{{status}}"}
              </p>
            </div>

            <div className="flex items-center gap-3">
              <Switch
                id="tpl-active"
                checked={form.enabled}
                onCheckedChange={(checked) =>
                  setForm((prev) => ({ ...prev, enabled: checked }))
                }
              />
              <Label htmlFor="tpl-active">Aktywny</Label>
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={closeDialog}>
              Anuluj
            </Button>
            <Button
              onClick={handleSubmit}
              disabled={
                !form.name.trim() || !form.body.trim() || isSaving
              }
            >
              {isSaving
                ? "Zapisywanie..."
                : editingTemplate
                  ? "Zapisz"
                  : "Utwórz"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Delete confirmation */}
      <ConfirmDialog
        open={!!deleteId}
        onOpenChange={(open) => !open && setDeleteId(null)}
        title="Usuń szablon"
        description="Czy na pewno chcesz usunąć ten szablon wiadomości? Ta operacja jest nieodwracalna."
        confirmLabel="Usuń"
        variant="destructive"
        onConfirm={handleDelete}
        isLoading={deleteTemplate.isPending}
      />
    </AdminGuard>
  );
}
