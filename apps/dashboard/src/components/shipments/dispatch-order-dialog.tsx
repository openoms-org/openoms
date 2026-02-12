"use client";

import { useState } from "react";
import { Loader2 } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { useCreateDispatchOrder } from "@/hooks/use-shipments";
import { useCompanySettings } from "@/hooks/use-settings";

interface DispatchOrderDialogProps {
  shipmentIds: string[];
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function DispatchOrderDialog({
  shipmentIds,
  open,
  onOpenChange,
}: DispatchOrderDialogProps) {
  const createDispatchOrder = useCreateDispatchOrder();
  const { data: company } = useCompanySettings();

  const [formData, setFormData] = useState({
    name: "",
    phone: "",
    email: "",
    street: "",
    building_number: "",
    city: "",
    post_code: "",
    comment: "",
  });

  const handleOpen = (isOpen: boolean) => {
    if (isOpen && company) {
      setFormData((prev) => ({
        name: prev.name || company.company_name || "",
        phone: prev.phone || company.phone || "",
        email: prev.email || company.email || "",
        street: prev.street || "",
        building_number: prev.building_number || "",
        city: prev.city || company.city || "",
        post_code: prev.post_code || company.post_code || "",
        comment: prev.comment,
      }));
    }
    onOpenChange(isOpen);
  };

  const handleChange = (field: string, value: string) => {
    setFormData((prev) => ({ ...prev, [field]: value }));
  };

  const handleSubmit = () => {
    createDispatchOrder.mutate(
      {
        shipment_ids: shipmentIds,
        ...formData,
      },
      {
        onSuccess: (data) => {
          toast.success(`Zlecenie odbioru #${data.id} utworzone (status: ${data.status})`);
          onOpenChange(false);
        },
        onError: (error) => {
          toast.error(error.message || "Nie udalo sie utworzyc zlecenia odbioru");
        },
      }
    );
  };

  return (
    <Dialog open={open} onOpenChange={handleOpen}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Zamow kuriera InPost</DialogTitle>
          <DialogDescription>
            Zlecenie odbioru dla {shipmentIds.length} {shipmentIds.length === 1 ? "przesylki" : "przesylek"}.
            Kurier odbierze paczki pod wskazanym adresem.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-2">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>Imie i nazwisko / Firma</Label>
              <Input
                value={formData.name}
                onChange={(e) => handleChange("name", e.target.value)}
                placeholder="Jan Kowalski"
              />
            </div>
            <div className="space-y-2">
              <Label>Telefon</Label>
              <Input
                value={formData.phone}
                onChange={(e) => handleChange("phone", e.target.value)}
                placeholder="+48501501501"
              />
            </div>
          </div>

          <div className="space-y-2">
            <Label>Email</Label>
            <Input
              type="email"
              value={formData.email}
              onChange={(e) => handleChange("email", e.target.value)}
              placeholder="kontakt@firma.pl"
            />
          </div>

          <div className="grid grid-cols-3 gap-4">
            <div className="col-span-2 space-y-2">
              <Label>Ulica</Label>
              <Input
                value={formData.street}
                onChange={(e) => handleChange("street", e.target.value)}
                placeholder="ul. Testowa"
              />
            </div>
            <div className="space-y-2">
              <Label>Nr budynku</Label>
              <Input
                value={formData.building_number}
                onChange={(e) => handleChange("building_number", e.target.value)}
                placeholder="1"
              />
            </div>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>Miasto</Label>
              <Input
                value={formData.city}
                onChange={(e) => handleChange("city", e.target.value)}
                placeholder="Warszawa"
              />
            </div>
            <div className="space-y-2">
              <Label>Kod pocztowy</Label>
              <Input
                value={formData.post_code}
                onChange={(e) => handleChange("post_code", e.target.value)}
                placeholder="00-001"
              />
            </div>
          </div>

          <div className="space-y-2">
            <Label>Komentarz (opcjonalnie)</Label>
            <Textarea
              value={formData.comment}
              onChange={(e) => handleChange("comment", e.target.value)}
              placeholder="Dodatkowe informacje dla kuriera..."
              className="min-h-16"
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Anuluj
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={createDispatchOrder.isPending}
          >
            {createDispatchOrder.isPending && (
              <Loader2 className="h-4 w-4 animate-spin" />
            )}
            Zamow kuriera
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
